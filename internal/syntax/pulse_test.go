package syntax

import (
	"testing"
)

func TestPulseInfo_FieldTypes(t *testing.T) {
	var p PulseInfo

	// Verify field existence and types by assignment
	p.NumberPulse = 0
	p.PulseStartSFB = 0
	p.PulseOffset[0] = 0
	p.PulseAmp[0] = 0

	// Verify array sizes
	if len(p.PulseOffset) != 4 {
		t.Errorf("PulseOffset should have 4 elements, got %d", len(p.PulseOffset))
	}
	if len(p.PulseAmp) != 4 {
		t.Errorf("PulseAmp should have 4 elements, got %d", len(p.PulseAmp))
	}
}

func TestPulseInfo_ZeroValue(t *testing.T) {
	var p PulseInfo

	// Zero value should indicate no pulse data
	if p.NumberPulse != 0 {
		t.Errorf("Zero PulseInfo should have NumberPulse=0")
	}
}
