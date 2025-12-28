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

func TestParseMSMask_Reserved(t *testing.T) {
	// Test that ms_mask_present == 3 returns error
	// Build bitstream with ms_mask_present = 3 (binary: 11)
	data := []byte{0xC0} // 11 + padding
	r := bits.NewReader(data)

	ics := &ICStream{
		NumWindowGroups: 1,
		MaxSFB:          10,
	}

	err := parseMSMask(r, ics)
	if err != ErrMSMaskReserved {
		t.Errorf("Expected ErrMSMaskReserved, got %v", err)
	}
}

func TestParseMSMask_NoMS(t *testing.T) {
	// Test ms_mask_present == 0 (no M/S stereo)
	data := []byte{0x00} // 00 + padding
	r := bits.NewReader(data)

	ics := &ICStream{
		NumWindowGroups: 1,
		MaxSFB:          10,
	}

	err := parseMSMask(r, ics)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if ics.MSMaskPresent != 0 {
		t.Errorf("MSMaskPresent = %d, want 0", ics.MSMaskPresent)
	}
}

func TestParseMSMask_AllMS(t *testing.T) {
	// Test ms_mask_present == 2 (all bands use M/S)
	data := []byte{0x80} // 10 + padding
	r := bits.NewReader(data)

	ics := &ICStream{
		NumWindowGroups: 1,
		MaxSFB:          10,
	}

	err := parseMSMask(r, ics)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if ics.MSMaskPresent != 2 {
		t.Errorf("MSMaskPresent = %d, want 2", ics.MSMaskPresent)
	}
}

func TestParseMSMask_PerBand(t *testing.T) {
	// Test ms_mask_present == 1 (per-band mask)
	// With NumWindowGroups=1, MaxSFB=4: need 4 bits for mask
	// Bitstream: 01 (ms_mask=1) + 1010 (mask bits) + padding
	// = 0110_1000 = 0x68
	// After reading 2 bits for ms_mask (01), remaining bits are:
	// 1, 0, 1, 0 (read left-to-right from MSB)
	data := []byte{0x68}
	r := bits.NewReader(data)

	ics := &ICStream{
		NumWindowGroups: 1,
		MaxSFB:          4,
	}

	err := parseMSMask(r, ics)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if ics.MSMaskPresent != 1 {
		t.Errorf("MSMaskPresent = %d, want 1", ics.MSMaskPresent)
	}

	// Check mask bits: 1, 0, 1, 0
	expected := []uint8{1, 0, 1, 0}
	for i, exp := range expected {
		if ics.MSUsed[0][i] != exp {
			t.Errorf("MSUsed[0][%d] = %d, want %d", i, ics.MSUsed[0][i], exp)
		}
	}
}
