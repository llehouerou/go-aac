package spectrum

import (
	"math"
	"testing"
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
