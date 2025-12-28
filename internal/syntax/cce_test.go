// internal/syntax/cce_test.go
package syntax

import (
	"testing"
)

func TestCCEConfig_Initialization(t *testing.T) {
	cfg := &CCEConfig{
		SFIndex:     4, // 44100 Hz
		FrameLength: 1024,
		ObjectType:  2, // AAC-LC
	}

	if cfg.SFIndex != 4 {
		t.Errorf("SFIndex: got %d, want 4", cfg.SFIndex)
	}
	if cfg.FrameLength != 1024 {
		t.Errorf("FrameLength: got %d, want 1024", cfg.FrameLength)
	}
	if cfg.ObjectType != 2 {
		t.Errorf("ObjectType: got %d, want 2", cfg.ObjectType)
	}
}

func TestCCEResult_Initialization(t *testing.T) {
	result := &CCEResult{}

	if result.Tag != 0 {
		t.Errorf("Tag should be zero-initialized")
	}
	if result.IndSwCCEFlag {
		t.Errorf("IndSwCCEFlag should be false initially")
	}
	if result.NumCoupledElements != 0 {
		t.Errorf("NumCoupledElements should be zero-initialized")
	}
}

func TestCCECoupledElement_Initialization(t *testing.T) {
	elem := CCECoupledElement{
		TargetIsCPE: true,
		TargetTag:   5,
		CCL:         true,
		CCR:         false,
	}

	if !elem.TargetIsCPE {
		t.Errorf("TargetIsCPE should be true")
	}
	if elem.TargetTag != 5 {
		t.Errorf("TargetTag: got %d, want 5", elem.TargetTag)
	}
	if !elem.CCL {
		t.Errorf("CCL should be true")
	}
	if elem.CCR {
		t.Errorf("CCR should be false")
	}
}
