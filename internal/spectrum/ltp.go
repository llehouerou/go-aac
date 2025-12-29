// internal/spectrum/ltp.go
package spectrum

import (
	"math"

	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/syntax"
)

// ltpCodebook contains the 8 LTP coefficient values.
// The transmitted coef index (0-7) indexes into this table.
//
// Ported from: codebook[] in ~/dev/faad2/libfaad/lt_predict.c:68-78
var ltpCodebook = [8]float64{
	0.570829,
	0.696616,
	0.813004,
	0.911304,
	0.984900,
	1.067894,
	1.194601,
	1.369533,
}

// IsLTPObjectType returns true if the given object type supports LTP.
//
// Ported from: is_ltp_ot() in ~/dev/faad2/libfaad/lt_predict.c:49-66
func IsLTPObjectType(objectType aac.ObjectType) bool {
	switch objectType {
	case aac.ObjectTypeLTP, aac.ObjectTypeERLTP, aac.ObjectTypeLD:
		return true
	default:
		return false
	}
}

// realToInt16 converts a floating-point sample to int16 with rounding and clamping.
// Uses round-half-away-from-zero for .5 values.
//
// Ported from: real_to_int16() in ~/dev/faad2/libfaad/lt_predict.c:152-170
func realToInt16(sigIn float64) int16 {
	// Round to nearest integer (away from zero for .5)
	var rounded float64
	if sigIn >= 0 {
		rounded = math.Floor(sigIn + 0.5)
		if rounded >= 32768.0 {
			return 32767
		}
	} else {
		rounded = math.Ceil(sigIn - 0.5)
		if rounded <= -32768.0 {
			return -32768
		}
	}
	return int16(rounded)
}

// LTPUpdateState updates the LTP state buffer with the latest decoded samples.
// This must be called after each frame is decoded to maintain the prediction state.
//
// The state buffer layout is:
//   - Non-LD: [old_half | time_samples | overlap_samples | zeros]
//   - LD: [extra_512 | old_half | time_samples | overlap_samples]
//
// Parameters:
//   - ltPredStat: LTP state buffer (4*frameLen samples for LTP, or 4*512 for LD)
//   - time: decoded time-domain samples for current frame
//   - overlap: overlap samples from filter bank
//   - frameLen: frame length (1024 or 960)
//   - objectType: AAC object type
//
// Ported from: lt_update_state() in ~/dev/faad2/libfaad/lt_predict.c:173-213
func LTPUpdateState(ltPredStat []int16, time, overlap []float64, frameLen uint16, objectType aac.ObjectType) {
	if objectType == aac.ObjectTypeLD {
		// LD mode: extra 512 samples lookback
		for i := uint16(0); i < frameLen; i++ {
			ltPredStat[i] = ltPredStat[i+frameLen]             // Shift down
			ltPredStat[frameLen+i] = ltPredStat[i+2*frameLen]  // Shift down
			ltPredStat[2*frameLen+i] = realToInt16(time[i])    // New time samples
			ltPredStat[3*frameLen+i] = realToInt16(overlap[i]) // New overlap samples
		}
	} else {
		// Non-LD mode (LTP, etc.)
		for i := uint16(0); i < frameLen; i++ {
			ltPredStat[i] = ltPredStat[i+frameLen]             // Shift down
			ltPredStat[frameLen+i] = realToInt16(time[i])      // New time samples
			ltPredStat[2*frameLen+i] = realToInt16(overlap[i]) // New overlap samples
			// ltPredStat[3*frameLen+i] stays zero (initialized once)
		}
	}
}

// ForwardMDCT is the interface for forward MDCT transformation.
// This is provided by the filterbank package (Phase 5).
type ForwardMDCT interface {
	// FilterBankLTP applies forward MDCT for LTP.
	// Transforms time-domain samples to frequency-domain coefficients.
	FilterBankLTP(windowSequence uint8, windowShape, windowShapePrev uint8,
		inData []float64, outMDCT []float64, objectType aac.ObjectType, frameLen uint16)
}

// LTPConfig holds configuration for LTP prediction.
type LTPConfig struct {
	// ICS is the individual channel stream
	ICS *syntax.ICStream

	// LTP is the LTP info from parsing
	LTP *syntax.LTPInfo

	// SRIndex is the sample rate index
	SRIndex uint8

	// ObjectType is the AAC object type
	ObjectType aac.ObjectType

	// FrameLength is the frame length (1024 or 960)
	FrameLength uint16

	// WindowShape is the current window shape
	WindowShape uint8

	// WindowShapePrev is the previous window shape
	WindowShapePrev uint8
}

// LTPPrediction applies Long Term Prediction to spectral coefficients.
// LTP uses previously decoded samples to predict and enhance the current frame.
//
// The prediction process:
// 1. Looks back into the state buffer by 'lag' samples
// 2. Multiplies by the LTP coefficient
// 3. Applies forward MDCT to get frequency-domain prediction
// 4. Applies TNS encoding to match original processing
// 5. Adds prediction to spectrum for bands where LTP is active
//
// Parameters:
//   - spec: spectral coefficients to modify (input/output)
//   - ltPredStat: LTP state buffer (previous decoded samples as int16)
//   - cfg: LTP configuration
//
// Note: LTP is only applied to long windows, not short blocks.
//
// Ported from: lt_prediction() in ~/dev/faad2/libfaad/lt_predict.c:80-133
func LTPPrediction(spec []float64, ltPredStat []int16, cfg *LTPConfig) {
	LTPPredictionWithMDCT(spec, ltPredStat, nil, cfg)
}

// LTPPredictionWithMDCT applies Long Term Prediction with an explicit forward MDCT.
// Use this when the filterbank is available.
func LTPPredictionWithMDCT(spec []float64, ltPredStat []int16, fb ForwardMDCT, cfg *LTPConfig) {
	ics := cfg.ICS
	ltp := cfg.LTP

	// LTP is not applied to short blocks
	if ics.WindowSequence == syntax.EightShortSequence {
		return
	}

	// Check if LTP data is present
	if !ltp.DataPresent {
		return
	}

	// Forward MDCT is required for actual prediction
	if fb == nil {
		// TODO: Remove this check once filterbank is implemented
		// For now, return early if no filterbank is provided
		return
	}

	numSamples := cfg.FrameLength * 2

	// Create time-domain estimate from state buffer
	xEst := make([]float64, numSamples)
	coef := ltpCodebook[ltp.Coef]

	for i := uint16(0); i < numSamples; i++ {
		// Look back by 'lag' samples and multiply by coefficient
		stateIdx := numSamples + i - ltp.Lag
		xEst[i] = float64(ltPredStat[stateIdx]) * coef
	}

	// Apply forward MDCT to get frequency-domain prediction
	XEst := make([]float64, numSamples)
	fb.FilterBankLTP(uint8(ics.WindowSequence), cfg.WindowShape, cfg.WindowShapePrev,
		xEst, XEst, cfg.ObjectType, cfg.FrameLength)

	// Apply TNS encoding to match the processing applied to the original spectrum
	tnsCfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     cfg.SRIndex,
		ObjectType:  cfg.ObjectType,
		FrameLength: cfg.FrameLength,
	}
	TNSEncodeFrame(XEst, tnsCfg)

	// Add prediction to spectrum for SFBs where LTP is used
	for sfb := uint8(0); sfb < ltp.LastBand; sfb++ {
		if ltp.LongUsed[sfb] {
			low := ics.SWBOffset[sfb]
			high := ics.SWBOffset[sfb+1]
			if high > ics.SWBOffsetMax {
				high = ics.SWBOffsetMax
			}

			for bin := low; bin < high; bin++ {
				spec[bin] += XEst[bin]
			}
		}
	}
}
