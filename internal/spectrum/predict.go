// Package spectrum provides spectral processing functions for AAC decoding.

package spectrum

import (
	"math"

	"github.com/llehouerou/go-aac/internal/syntax"
	"github.com/llehouerou/go-aac/internal/tables"
)

// PredState holds the state for one spectral coefficient's predictor.
// The values are quantized to 16-bit for memory efficiency and stability.
//
// Ported from: pred_state in ~/dev/faad2/libfaad/structs.h:51-55
type PredState struct {
	R   [2]int16 // Predictor state (past output)
	COR [2]int16 // Correlation accumulators
	VAR [2]int16 // Variance accumulators
}

// NewPredState creates a new predictor state with initial values.
func NewPredState() *PredState {
	s := &PredState{}
	ResetPredState(s)
	return s
}

// ResetPredState resets a single predictor state to initial values.
// After reset, the predictor will output zero prediction.
//
// Ported from: reset_pred_state() in ~/dev/faad2/libfaad/ic_predict.c:198-206
func ResetPredState(state *PredState) {
	state.R[0] = 0
	state.R[1] = 0
	state.COR[0] = 0
	state.COR[1] = 0
	state.VAR[0] = 0x3F80 // 1.0 in quantized form
	state.VAR[1] = 0x3F80 // 1.0 in quantized form
}

// ResetAllPredictors resets all predictor states in the array.
//
// Ported from: reset_all_predictors() in ~/dev/faad2/libfaad/ic_predict.c:236-241
func ResetAllPredictors(states []PredState, frameLen uint16) {
	for i := uint16(0); i < frameLen && int(i) < len(states); i++ {
		ResetPredState(&states[i])
	}
}

// floatToBits converts a float32 to its IEEE 754 bit representation.
func floatToBits(f float32) uint32 {
	return math.Float32bits(f)
}

// bitsToFloat converts IEEE 754 bits to a float32.
func bitsToFloat(u uint32) float32 {
	return math.Float32frombits(u)
}

// fltRound rounds a float32 to 16-bit mantissa precision.
// This matches FAAD2's flt_round() which rounds 0.5 LSB toward infinity.
//
// Ported from: flt_round() in ~/dev/faad2/libfaad/ic_predict.c:53-74
func fltRound(pf float32) float32 {
	tmp := floatToBits(pf)
	flg := tmp & 0x00008000

	tmp &= 0xffff0000
	tmp1 := tmp

	// Round 0.5 LSB toward infinity
	if flg != 0 {
		tmp &= 0xff800000 // Extract exponent and sign
		tmp |= 0x00010000 // Insert 1 LSB
		tmp2 := tmp       // Add 1 LSB and elided one
		tmp &= 0xff800000 // Extract exponent and sign

		return bitsToFloat(tmp1) + bitsToFloat(tmp2) - bitsToFloat(tmp)
	}
	return bitsToFloat(tmp)
}

// quantPred quantizes a float32 to 16-bit by taking the upper 16 bits.
//
// Ported from: quant_pred() in ~/dev/faad2/libfaad/ic_predict.c:76-79
func quantPred(x float32) int16 {
	return int16(floatToBits(x) >> 16)
}

// invQuantPred dequantizes a 16-bit value back to float32.
//
// Ported from: inv_quant_pred() in ~/dev/faad2/libfaad/ic_predict.c:81-85
func invQuantPred(q int16) float32 {
	u16 := uint16(q)
	return bitsToFloat(uint32(u16) << 16)
}

// icPredict applies backward-adaptive prediction to one spectral coefficient.
// If pred is true, the prediction is added to the input.
// The state is always updated regardless of pred.
//
// Ported from: ic_predict() in ~/dev/faad2/libfaad/ic_predict.c:87-196
func icPredict(state *PredState, input float32, pred bool) float32 {
	// Dequantize state
	r0 := invQuantPred(state.R[0])
	r1 := invQuantPred(state.R[1])
	cor0 := invQuantPred(state.COR[0])
	cor1 := invQuantPred(state.COR[1])
	var0 := invQuantPred(state.VAR[0])
	var1 := invQuantPred(state.VAR[1])

	// Calculate k1 coefficient using table lookup
	var k1 float32
	tmp := uint16(state.VAR[0])
	j := int(tmp >> 7)
	i := int(tmp & 0x7f)
	if j >= 128 {
		j -= 128
		k1 = cor0 * expTable[j] * mntTable[i]
	} else {
		k1 = 0
	}

	var output float32
	if pred {
		// Calculate k2 coefficient
		var k2 float32
		tmp = uint16(state.VAR[1])
		j = int(tmp >> 7)
		i = int(tmp & 0x7f)
		if j >= 128 {
			j -= 128
			k2 = cor1 * expTable[j] * mntTable[i]
		} else {
			k2 = 0
		}

		// Calculate predicted value
		predictedValue := k1*r0 + k2*r1
		predictedValue = fltRound(predictedValue)
		output = input + predictedValue
	} else {
		output = input
	}

	// Calculate new state data
	e0 := output
	e1 := e0 - k1*r0
	dr1 := k1 * e0

	// Update variance and correlation
	var0 = predAlpha*var0 + 0.5*(r0*r0+e0*e0)
	cor0 = predAlpha*cor0 + r0*e0
	var1 = predAlpha*var1 + 0.5*(r1*r1+e1*e1)
	cor1 = predAlpha*cor1 + r1*e1

	// Update predictor state
	r1 = predA * (r0 - dr1)
	r0 = predA * e0

	// Quantize and store state
	state.R[0] = quantPred(r0)
	state.R[1] = quantPred(r1)
	state.COR[0] = quantPred(cor0)
	state.COR[1] = quantPred(cor1)
	state.VAR[0] = quantPred(var0)
	state.VAR[1] = quantPred(var1)

	return output
}

// PNSResetPredState resets predictor states for bands that use PNS (noise) coding.
// This is called after PNS decoding to prevent prediction from affecting noise bands.
// Only applies to long blocks.
//
// Ported from: pns_reset_pred_state() in ~/dev/faad2/libfaad/ic_predict.c:208-234
func PNSResetPredState(ics *syntax.ICStream, states []PredState) {
	// Prediction only for long blocks
	if ics.WindowSequence == syntax.EightShortSequence {
		return
	}

	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		for b := uint8(0); b < ics.WindowGroupLength[g]; b++ {
			for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
				if IsNoiseICS(ics, g, sfb) {
					offs := ics.SWBOffset[sfb]
					offs2 := ics.SWBOffset[sfb+1]
					if offs2 > ics.SWBOffsetMax {
						offs2 = ics.SWBOffsetMax
					}

					for i := offs; i < offs2 && int(i) < len(states); i++ {
						ResetPredState(&states[i])
					}
				}
			}
		}
	}
}

// ICPrediction applies intra-channel prediction to the spectral coefficients.
// For short sequences, all predictors are reset.
// For long sequences, prediction is applied per SFB based on prediction_used flags.
//
// Ported from: ic_prediction() in ~/dev/faad2/libfaad/ic_predict.c:245-279
func ICPrediction(ics *syntax.ICStream, spec []float32, states []PredState, frameLen uint16, sfIndex uint8) {
	if ics.WindowSequence == syntax.EightShortSequence {
		// Short sequence: reset all predictors
		ResetAllPredictors(states, frameLen)
		return
	}

	// Long sequence: apply prediction per SFB
	maxPredSfb := tables.MaxPredSFB(sfIndex)

	for sfb := uint8(0); sfb < maxPredSfb; sfb++ {
		low := ics.SWBOffset[sfb]
		high := ics.SWBOffset[sfb+1]
		if high > ics.SWBOffsetMax {
			high = ics.SWBOffsetMax
		}

		// Determine if prediction is used for this SFB
		usePred := ics.PredictorDataPresent && ics.Pred.PredictionUsed[sfb]

		for bin := low; bin < high && int(bin) < len(spec) && int(bin) < len(states); bin++ {
			spec[bin] = icPredict(&states[bin], spec[bin], usePred)
		}
	}

	// Handle predictor reset groups
	if ics.PredictorDataPresent && ics.Pred.PredictorReset {
		resetGroup := ics.Pred.PredictorResetGroupNumber
		if resetGroup > 0 {
			// Reset every 30th predictor starting from (resetGroup - 1)
			for bin := uint16(resetGroup - 1); bin < frameLen && int(bin) < len(states); bin += 30 {
				ResetPredState(&states[bin])
			}
		}
	}
}
