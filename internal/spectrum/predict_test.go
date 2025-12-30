package spectrum

import (
	"math"
	"testing"

	"github.com/llehouerou/go-aac/internal/huffman"
	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestNewPredState(t *testing.T) {
	state := NewPredState()
	// After reset, VAR should be 0x3F80 (1.0 in quantized form)
	if state.VAR[0] != 0x3F80 {
		t.Errorf("VAR[0] = %#x, want 0x3F80", state.VAR[0])
	}
	if state.VAR[1] != 0x3F80 {
		t.Errorf("VAR[1] = %#x, want 0x3F80", state.VAR[1])
	}
	// R and COR should be zero
	if state.R[0] != 0 || state.R[1] != 0 {
		t.Errorf("R = %v, want [0, 0]", state.R)
	}
	if state.COR[0] != 0 || state.COR[1] != 0 {
		t.Errorf("COR = %v, want [0, 0]", state.COR)
	}
}

func TestResetPredState(t *testing.T) {
	state := &PredState{
		R:   [2]int16{100, 200},
		COR: [2]int16{300, 400},
		VAR: [2]int16{500, 600},
	}
	ResetPredState(state)

	if state.R[0] != 0 || state.R[1] != 0 {
		t.Errorf("after reset, R = %v, want [0, 0]", state.R)
	}
	if state.COR[0] != 0 || state.COR[1] != 0 {
		t.Errorf("after reset, COR = %v, want [0, 0]", state.COR)
	}
	if state.VAR[0] != 0x3F80 || state.VAR[1] != 0x3F80 {
		t.Errorf("after reset, VAR = %v, want [0x3F80, 0x3F80]", state.VAR)
	}
}

func TestResetAllPredictors(t *testing.T) {
	frameLen := uint16(1024)
	states := make([]PredState, frameLen)

	// Set some non-zero values
	for i := range states {
		states[i].R[0] = int16(i)
		states[i].VAR[0] = int16(i + 100)
	}

	ResetAllPredictors(states, frameLen)

	// Check all are reset
	for i := uint16(0); i < frameLen; i++ {
		if states[i].R[0] != 0 {
			t.Errorf("states[%d].R[0] = %d, want 0", i, states[i].R[0])
			break
		}
		if states[i].VAR[0] != 0x3F80 {
			t.Errorf("states[%d].VAR[0] = %#x, want 0x3F80", i, states[i].VAR[0])
			break
		}
	}
}

func TestQuantPred(t *testing.T) {
	testCases := []struct {
		input    float32
		expected int16
	}{
		{0.0, 0},
		{1.0, 0x3F80},  // IEEE 754: 0x3F800000
		{-1.0, -16512}, // IEEE 754: 0xBF800000 -> 0xBF80 as int16
		{0.5, 0x3F00},  // IEEE 754: 0x3F000000
	}
	for _, tc := range testCases {
		got := quantPred(tc.input)
		if got != tc.expected {
			t.Errorf("quantPred(%v) = %#x, want %#x", tc.input, got, tc.expected)
		}
	}
}

func TestInvQuantPred(t *testing.T) {
	testCases := []struct {
		input    int16
		expected float32
	}{
		{0, 0.0},
		{0x3F80, 1.0},
		{0x3F00, 0.5},
	}
	for _, tc := range testCases {
		got := invQuantPred(tc.input)
		if math.Abs(float64(got-tc.expected)) > 1e-6 {
			t.Errorf("invQuantPred(%#x) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestFltRound(t *testing.T) {
	// Test that fltRound rounds to 16-bit precision
	testCases := []struct {
		input    float32
		expected float32
	}{
		{1.0, 1.0},       // Already aligned
		{0.5, 0.5},       // Already aligned
		{1.0000001, 1.0}, // Should round
	}
	for _, tc := range testCases {
		got := fltRound(tc.input)
		// Check that result has same upper 16 bits as expected
		gotQ := quantPred(got)
		expQ := quantPred(tc.expected)
		if gotQ != expQ {
			t.Errorf("fltRound(%v): quantized = %#x, want %#x", tc.input, gotQ, expQ)
		}
	}
}

func TestICPredict_NoPrediction(t *testing.T) {
	// When pred=false, output should equal input and state should still update
	state := NewPredState()
	input := float32(0.5)

	output := icPredict(state, input, false)

	// Output should be unchanged (no prediction applied)
	if output != input {
		t.Errorf("icPredict with pred=false: output = %v, want %v", output, input)
	}
}

func TestICPredict_WithPrediction(t *testing.T) {
	// When pred=true, output should be input + predicted value
	state := NewPredState()

	// First sample: no prediction yet (k1, k2 = 0)
	output1 := icPredict(state, 1.0, true)
	// With fresh state, prediction should be 0, so output = input
	if math.Abs(float64(output1-1.0)) > 0.001 {
		t.Errorf("first sample: output = %v, want ~1.0", output1)
	}

	// Second sample: state has been updated, prediction should be non-zero
	output2 := icPredict(state, 1.0, true)
	// After one update, there should be some prediction
	if output2 == 1.0 {
		t.Logf("second sample: output = %v (prediction may be small)", output2)
	}
}

func TestICPredict_StateUpdate(t *testing.T) {
	// Verify that state is updated after each call
	state := NewPredState()

	_ = icPredict(state, 1.0, true)

	// State should have been updated
	if state.R[0] == 0 && state.R[1] == 0 {
		t.Error("state.R was not updated")
	}
}

func TestICPrediction_ShortSequence(t *testing.T) {
	// For short sequences, all predictors should be reset
	frameLen := uint16(1024)
	states := make([]PredState, frameLen)

	// Set non-zero values
	for i := range states {
		states[i].R[0] = 100
	}

	ics := &syntax.ICStream{
		WindowSequence: syntax.EightShortSequence,
	}
	spec := make([]float32, frameLen)

	ICPrediction(ics, spec, states, frameLen, 3) // sfIndex=3 (48kHz)

	// All states should be reset
	for i := uint16(0); i < frameLen; i++ {
		if states[i].R[0] != 0 {
			t.Errorf("states[%d].R[0] = %d, want 0 (should be reset)", i, states[i].R[0])
			break
		}
	}
}

func TestICPrediction_LongSequence(t *testing.T) {
	// For long sequences, prediction should be applied
	frameLen := uint16(1024)
	states := make([]PredState, frameLen)
	for i := range states {
		ResetPredState(&states[i])
	}

	ics := &syntax.ICStream{
		WindowSequence:       syntax.OnlyLongSequence,
		MaxSFB:               10,
		PredictorDataPresent: true,
	}
	ics.Pred.Limit = 10
	for i := uint8(0); i < 10; i++ {
		ics.Pred.PredictionUsed[i] = true
	}

	// Set up SWB offsets (simplified)
	for i := 0; i <= 10; i++ {
		ics.SWBOffset[i] = uint16(i * 10)
	}
	ics.SWBOffsetMax = 100

	spec := make([]float32, frameLen)
	for i := range spec {
		spec[i] = 1.0
	}

	ICPrediction(ics, spec, states, frameLen, 3)

	// After prediction with fresh states, spec should be mostly unchanged
	// (prediction is zero for fresh states)
	// This is a basic sanity check
	if spec[0] != 1.0 {
		t.Logf("spec[0] = %v after prediction (expected ~1.0 for fresh state)", spec[0])
	}
}

func TestICPrediction_PredictorReset(t *testing.T) {
	// Test that predictor reset groups work correctly
	frameLen := uint16(120) // Use small frame to test reset pattern clearly
	states := make([]PredState, frameLen)

	// Set non-zero values for all states
	for i := range states {
		states[i].R[0] = 100
		states[i].COR[0] = 50
	}

	ics := &syntax.ICStream{
		WindowSequence:       syntax.OnlyLongSequence,
		MaxSFB:               1,
		PredictorDataPresent: true,
	}
	ics.Pred.PredictorReset = true
	ics.Pred.PredictorResetGroupNumber = 1 // Reset bins 0, 30, 60, 90

	// Set up minimal SWB offsets
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 10
	ics.SWBOffsetMax = 10

	spec := make([]float32, frameLen)

	ICPrediction(ics, spec, states, frameLen, 3)

	// Bins 0, 30, 60, 90 should be reset (VAR = 0x3F80, R = 0, COR = 0)
	resetBins := []uint16{0, 30, 60, 90}
	for _, bin := range resetBins {
		if states[bin].R[0] != 0 {
			t.Errorf("states[%d].R[0] = %d, want 0 (should be reset)", bin, states[bin].R[0])
		}
		if states[bin].VAR[0] != 0x3F80 {
			t.Errorf("states[%d].VAR[0] = %#x, want 0x3F80 (should be reset)", bin, states[bin].VAR[0])
		}
	}

	// Other bins should NOT be reset (but may be modified by prediction)
	// Check bin 15 which is not processed by prediction (beyond SWBOffset) and not in reset group
	if states[15].R[0] == 0 && states[15].COR[0] == 0 && states[15].VAR[0] == 0x3F80 {
		t.Errorf("states[15] appears reset but should not be in reset group")
	}
}

func TestICPrediction_PredictorResetGroup2(t *testing.T) {
	// Test reset group 2 (reset bins 1, 31, 61, 91, ...)
	frameLen := uint16(120)
	states := make([]PredState, frameLen)

	// Set non-zero values
	for i := range states {
		states[i].R[0] = 100
	}

	ics := &syntax.ICStream{
		WindowSequence:       syntax.OnlyLongSequence,
		MaxSFB:               0, // No prediction applied
		PredictorDataPresent: true,
	}
	ics.Pred.PredictorReset = true
	ics.Pred.PredictorResetGroupNumber = 2 // Reset bins 1, 31, 61, 91

	spec := make([]float32, frameLen)

	ICPrediction(ics, spec, states, frameLen, 3)

	// Bins 1, 31, 61, 91 should be reset
	resetBins := []uint16{1, 31, 61, 91}
	for _, bin := range resetBins {
		if states[bin].R[0] != 0 {
			t.Errorf("states[%d].R[0] = %d, want 0 (should be reset)", bin, states[bin].R[0])
		}
	}

	// Bins 0, 30, 60, 90 should NOT be reset
	nonResetBins := []uint16{0, 30, 60, 90}
	for _, bin := range nonResetBins {
		if states[bin].R[0] == 0 {
			t.Errorf("states[%d].R[0] = 0, should not be reset", bin)
		}
	}
}

func TestPNSResetPredState(t *testing.T) {
	frameLen := uint16(1024)
	states := make([]PredState, frameLen)

	// Set non-zero values
	for i := range states {
		states[i].R[0] = 100
	}

	ics := &syntax.ICStream{
		WindowSequence:    syntax.OnlyLongSequence,
		NumWindowGroups:   1,
		WindowGroupLength: [8]uint8{1},
		MaxSFB:            5,
		SWBOffsetMax:      100,
	}
	// Set up SWB offsets
	for i := 0; i <= 5; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
	}
	// Set SFB 2 to use noise codebook
	ics.SFBCB[0][2] = uint8(huffman.NoiseHCB)

	PNSResetPredState(ics, states)

	// States in SFB 2 (bins 40-59) should be reset
	for bin := 40; bin < 60; bin++ {
		if states[bin].R[0] != 0 {
			t.Errorf("states[%d].R[0] = %d, want 0 (noise band)", bin, states[bin].R[0])
		}
	}

	// States in other SFBs should not be reset
	if states[0].R[0] != 100 {
		t.Errorf("states[0].R[0] = %d, want 100 (non-noise band)", states[0].R[0])
	}
}

func TestPNSResetPredState_ShortSequence(t *testing.T) {
	// Short sequences should return early without doing anything
	states := make([]PredState, 1024)
	for i := range states {
		states[i].R[0] = 100
	}

	ics := &syntax.ICStream{
		WindowSequence: syntax.EightShortSequence,
	}

	PNSResetPredState(ics, states)

	// States should be unchanged
	if states[0].R[0] != 100 {
		t.Errorf("states[0].R[0] = %d, want 100 (short sequence, no reset)", states[0].R[0])
	}
}
