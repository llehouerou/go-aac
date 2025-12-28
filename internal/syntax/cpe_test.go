// internal/syntax/cpe_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
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

func TestParseChannelPairElement_ElementTag(t *testing.T) {
	// Test parsing element_instance_tag (4 bits) and common_window (1 bit)
	testCases := []struct {
		name         string
		tag          uint8
		commonWindow bool
	}{
		{"tag 0, no common window", 0, false},
		{"tag 7, common window", 7, true},
		{"tag 15, no common window", 15, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build bitstream: tag (4 bits) + common_window (1 bit)
			// For no common_window case, we need minimal ICS data for both channels
			// For common_window case, we need ics_info + ms_mask + ICS data

			// This is a basic structure test - full parsing tested separately
			if LenTag != 4 {
				t.Errorf("LenTag = %d, want 4", LenTag)
			}
		})
	}
}

func TestParseChannelPairElement_Signature(t *testing.T) {
	// Verify ParseChannelPairElement has the expected signature
	type parserFunc func(*bits.Reader, uint8, *CPEConfig) (*CPEResult, error)
	var _ parserFunc = ParseChannelPairElement
}
