// Package spectrum provides spectral processing functions for AAC decoding.

package spectrum

import "math"

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
