// internal/spectrum/tns.go
package spectrum

import (
	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/syntax"
	"github.com/llehouerou/go-aac/internal/tables"
)

// tnsARFilter applies an all-pole (AR) IIR filter to spectral coefficients.
// This is the core TNS decoding filter operation.
//
// The filter is defined by:
//
//	y[n] = x[n] - lpc[1]*y[n-1] - lpc[2]*y[n-2] - ... - lpc[order]*y[n-order]
//
// Parameters:
//   - spectrum: spectral data to filter (modified in-place), starting at offset 0
//   - size: number of samples to filter
//   - inc: direction (+1 for forward, -1 for backward)
//   - lpc: LPC filter coefficients (lpc[0] is always 1.0)
//   - order: filter order
//
// For forward filtering, pass the slice starting at the first sample.
// For backward filtering, this is a convenience wrapper that calls tnsARFilterWithOffset.
//
// Uses a double ringbuffer for efficient state management.
//
// Ported from: tns_ar_filter() in ~/dev/faad2/libfaad/tns.c:244-293
func tnsARFilter(spectrum []float64, size int16, inc int8, lpc []float64, order uint8) {
	tnsARFilterWithOffset(spectrum, 0, size, inc, lpc, order)
}

// tnsARFilterWithOffset applies an all-pole (AR) IIR filter to spectral coefficients
// starting at a specific offset within the spectrum slice.
//
// This version allows backward filtering by specifying a starting offset (e.g., the
// last element index for backward filtering) and a negative increment.
//
// Parameters:
//   - spectrum: full spectral data buffer (modified in-place)
//   - startOffset: index of first sample to process
//   - size: number of samples to filter
//   - inc: direction (+1 for forward, -1 for backward)
//   - lpc: LPC filter coefficients (lpc[0] is always 1.0)
//   - order: filter order
//
// Ported from: tns_ar_filter() in ~/dev/faad2/libfaad/tns.c:244-293
func tnsARFilterWithOffset(spectrum []float64, startOffset int, size int16, inc int8, lpc []float64, order uint8) {
	if size <= 0 || order == 0 {
		return
	}

	// State is stored as a double ringbuffer for efficient wraparound
	state := make([]float64, 2*TNSMaxOrder)
	stateIndex := int8(0)

	// Process each sample
	idx := startOffset
	for i := int16(0); i < size; i++ {
		// Compute filter output: y = x - sum(lpc[j+1] * state[j])
		y := 0.0
		for j := uint8(0); j < order; j++ {
			y += state[int(stateIndex)+int(j)] * lpc[j+1]
		}
		y = spectrum[idx] - y

		// Update double ringbuffer state
		stateIndex--
		if stateIndex < 0 {
			stateIndex = int8(order - 1)
		}
		state[stateIndex] = y
		state[int(stateIndex)+int(order)] = y

		// Write output and advance
		spectrum[idx] = y
		idx += int(inc)
	}
}

// tnsDecodeCoef converts transmitted TNS coefficients to LPC filter coefficients.
// Uses Levinson-Durbin recursion to convert reflection coefficients to direct form.
//
// Parameters:
//   - order: filter order (0-20)
//   - coefRes: coefficient resolution (0=3-bit, 1=4-bit)
//   - coefCompress: compression flag (0 or 1)
//   - coef: transmitted coefficient indices
//   - lpc: output LPC coefficients (must be len >= order+1)
//
// Ported from: tns_decode_coef() in ~/dev/faad2/libfaad/tns.c:193-242
func tnsDecodeCoef(order uint8, coefRes uint8, coefCompress uint8, coef []uint8, lpc []float64) {
	// Get the appropriate coefficient table
	tnsCoef := getTNSCoefTable(coefCompress, coefRes)

	// Convert transmitted indices to coefficient values
	tmp2 := make([]float64, TNSMaxOrder+1)
	for i := uint8(0); i < order; i++ {
		tmp2[i] = tnsCoef[coef[i]]
	}

	// Levinson-Durbin recursion to convert reflection coefficients to LPC
	// a[0] is always 1.0
	lpc[0] = 1.0

	b := make([]float64, TNSMaxOrder+1)
	for m := uint8(1); m <= order; m++ {
		// Set a[m] = reflection coefficient
		lpc[m] = tmp2[m-1]

		// Update previous coefficients
		for i := uint8(1); i < m; i++ {
			b[i] = lpc[i] + lpc[m]*lpc[m-i]
		}
		for i := uint8(1); i < m; i++ {
			lpc[i] = b[i]
		}
	}
}

// TNSDecodeConfig holds configuration for TNS decoding.
type TNSDecodeConfig struct {
	// ICS is the individual channel stream containing TNS data
	ICS *syntax.ICStream

	// SRIndex is the sample rate index (0-15)
	SRIndex uint8

	// ObjectType is the AAC object type
	ObjectType aac.ObjectType

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16
}

// TNSDecodeFrame applies TNS (Temporal Noise Shaping) decoding to one channel.
// TNS applies all-pole IIR filters to spectral coefficients to shape
// the temporal envelope of quantization noise.
//
// Ported from: tns_decode_frame() in ~/dev/faad2/libfaad/tns.c:84-136
func TNSDecodeFrame(spec []float64, cfg *TNSDecodeConfig) {
	ics := cfg.ICS

	if !ics.TNSDataPresent {
		return
	}

	tns := &ics.TNS
	nshort := cfg.FrameLength / 8
	isShort := ics.WindowSequence == syntax.EightShortSequence

	lpc := make([]float64, TNSMaxOrder+1)

	for w := uint8(0); w < ics.NumWindows; w++ {
		bottom := ics.NumSWB

		for f := uint8(0); f < tns.NFilt[w]; f++ {
			top := bottom
			// Compute bottom, ensuring non-negative
			if tns.Length[w][f] > top {
				bottom = 0
			} else {
				bottom = top - tns.Length[w][f]
			}

			// Clamp order to TNSMaxOrder
			tnsOrder := tns.Order[w][f]
			if tnsOrder > TNSMaxOrder {
				tnsOrder = TNSMaxOrder
			}
			if tnsOrder == 0 {
				continue
			}

			// Decode LPC coefficients
			tnsDecodeCoef(tnsOrder, tns.CoefRes[w], tns.CoefCompress[w][f], tns.Coef[w][f][:], lpc)

			// Calculate filter region bounds
			maxTNS := tables.MaxTNSSFB(cfg.SRIndex, cfg.ObjectType, isShort)

			// Start position
			start := bottom
			if start > maxTNS {
				start = maxTNS
			}
			if start > ics.MaxSFB {
				start = ics.MaxSFB
			}
			startSample := ics.SWBOffset[start]
			if startSample > ics.SWBOffsetMax {
				startSample = ics.SWBOffsetMax
			}

			// End position
			end := top
			if end > maxTNS {
				end = maxTNS
			}
			if end > ics.MaxSFB {
				end = ics.MaxSFB
			}
			endSample := ics.SWBOffset[end]
			if endSample > ics.SWBOffsetMax {
				endSample = ics.SWBOffsetMax
			}

			size := int16(endSample) - int16(startSample)
			if size <= 0 {
				continue
			}

			// Determine filter direction and starting position
			var inc int8
			var filterStart uint16
			if tns.Direction[w][f] != 0 {
				// Backward filtering
				inc = -1
				filterStart = endSample - 1
			} else {
				// Forward filtering
				inc = 1
				filterStart = startSample
			}

			// Apply the filter
			windowOffset := uint16(w) * nshort
			tnsARFilterWithOffset(spec, int(windowOffset+filterStart), size, inc, lpc, tnsOrder)
		}
	}
}
