// internal/syntax/cpe_test.go
package syntax

import (
	"testing"
)

func TestCPEConfig_Fields(t *testing.T) {
	cfg := &CPEConfig{
		SFIndex:     4,
		FrameLength: 1024,
		ObjectType:  ObjectTypeLC,
	}

	if cfg.SFIndex != 4 {
		t.Errorf("SFIndex = %d, want 4", cfg.SFIndex)
	}
	if cfg.FrameLength != 1024 {
		t.Errorf("FrameLength = %d, want 1024", cfg.FrameLength)
	}
	if cfg.ObjectType != ObjectTypeLC {
		t.Errorf("ObjectType = %d, want %d", cfg.ObjectType, ObjectTypeLC)
	}
}

func TestCPEResult_Fields(t *testing.T) {
	result := &CPEResult{
		Tag:       5,
		SpecData1: make([]int16, 1024),
		SpecData2: make([]int16, 1024),
	}

	if result.Tag != 5 {
		t.Errorf("Tag = %d, want 5", result.Tag)
	}
	if len(result.SpecData1) != 1024 {
		t.Errorf("len(SpecData1) = %d, want 1024", len(result.SpecData1))
	}
	if len(result.SpecData2) != 1024 {
		t.Errorf("len(SpecData2) = %d, want 1024", len(result.SpecData2))
	}
}

func TestCPEResult_ElementInitialization(t *testing.T) {
	result := &CPEResult{}

	// Verify Element can hold two channels
	result.Element.Channel = 0
	result.Element.PairedChannel = 1
	result.Element.CommonWindow = true

	if result.Element.PairedChannel != 1 {
		t.Errorf("PairedChannel = %d, want 1", result.Element.PairedChannel)
	}
	if !result.Element.CommonWindow {
		t.Error("CommonWindow should be true")
	}
}
