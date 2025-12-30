// Package spectrum provides spectral processing functions for AAC decoding.

package spectrum

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
