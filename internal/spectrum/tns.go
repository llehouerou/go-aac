// internal/spectrum/tns.go
package spectrum

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
