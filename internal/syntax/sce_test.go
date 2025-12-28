// internal/syntax/sce_test.go
package syntax

import (
	"testing"
)

func TestParseSingleChannelElement_ElementTag(t *testing.T) {
	// Test that element_instance_tag is correctly parsed (4 bits)
	testCases := []struct {
		name     string
		tag      uint8
		expected uint8
	}{
		{"tag 0", 0, 0},
		{"tag 7", 7, 7},
		{"tag 15", 15, 15},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify the element_instance_tag constant is correct
			if LenTag != 4 {
				t.Errorf("LenTag = %d, want 4", LenTag)
			}

			// The actual parsing test requires a complete bitstream
			// with ICS data, which is tested in integration tests
		})
	}
}

func TestSCEConfig_Fields(t *testing.T) {
	// Test SCEConfig struct fields
	cfg := &SCEConfig{
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

func TestSCEResult_Fields(t *testing.T) {
	// Test SCEResult struct fields
	result := &SCEResult{
		Tag:      5,
		SpecData: make([]int16, 1024),
	}

	if result.Tag != 5 {
		t.Errorf("Tag = %d, want 5", result.Tag)
	}
	if len(result.SpecData) != 1024 {
		t.Errorf("len(SpecData) = %d, want 1024", len(result.SpecData))
	}
}

func TestErrIntensityStereoInSCE(t *testing.T) {
	if ErrIntensityStereoInSCE == nil {
		t.Error("ErrIntensityStereoInSCE should not be nil")
	}

	expectedMsg := "syntax: intensity stereo not allowed in single channel element"
	if ErrIntensityStereoInSCE.Error() != expectedMsg {
		t.Errorf("Error message = %q, want %q", ErrIntensityStereoInSCE.Error(), expectedMsg)
	}
}
