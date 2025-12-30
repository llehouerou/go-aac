package spectrum

import (
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
