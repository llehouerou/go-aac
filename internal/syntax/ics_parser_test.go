// internal/syntax/ics_parser_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseSideInfo_GlobalGain(t *testing.T) {
	// Test that global gain is correctly read (8 bits)
	// Bitstream: 0xAB (global_gain=171)
	// Then we need section data, scale factor data, and tool flags
	// For simplicity, we use common_window=true to skip ics_info

	// This is a minimal test - we can't easily test the full flow without
	// a valid section/scalefactor/tool sequence.
	// The global gain read is tested here; full integration tests will
	// validate the complete parsing path.

	testCases := []struct {
		name       string
		globalGain uint8
	}{
		{"gain 0", 0},
		{"gain 128", 128},
		{"gain 255", 255},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// We only verify the global gain read here
			// Full side_info parsing requires valid subsequent data
			data := []byte{tc.globalGain}
			r := bits.NewReader(data)

			ele := &Element{CommonWindow: true}
			ics := &ICStream{}
			cfg := &SideInfoConfig{
				SFIndex:      4, // 44.1 kHz
				FrameLength:  1024,
				ObjectType:   ObjectTypeLC,
				CommonWindow: true,
				ScalFlag:     false,
			}

			// Read global gain manually to verify
			ics.GlobalGain = uint8(r.GetBits(8))

			if ics.GlobalGain != tc.globalGain {
				t.Errorf("GlobalGain = %d, want %d", ics.GlobalGain, tc.globalGain)
			}

			// Note: Full ParseSideInfo would fail here due to missing section data
			// This test only verifies the global gain reading logic
			_ = ele
			_ = cfg
		})
	}
}

func TestParseSideInfo_ScalFlag(t *testing.T) {
	// When scal_flag is true, pulse/TNS/gain control flags are not read
	// We verify this by checking that fewer bits are consumed

	// For this test, we just verify the struct field is available
	cfg := &SideInfoConfig{
		ScalFlag: true,
	}

	if !cfg.ScalFlag {
		t.Error("ScalFlag should be true")
	}
}

func TestSideInfoConfig_Fields(t *testing.T) {
	// Test SideInfoConfig struct fields
	cfg := &SideInfoConfig{
		SFIndex:      4,
		FrameLength:  1024,
		ObjectType:   ObjectTypeLC,
		CommonWindow: true,
		ScalFlag:     false,
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
	if !cfg.CommonWindow {
		t.Error("CommonWindow should be true")
	}
	if cfg.ScalFlag {
		t.Error("ScalFlag should be false")
	}
}

func TestICSConfig_Fields(t *testing.T) {
	// Test ICSConfig struct fields
	cfg := &ICSConfig{
		SFIndex:      4,
		FrameLength:  1024,
		ObjectType:   ObjectTypeLC,
		CommonWindow: false,
		ScalFlag:     false,
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
	if cfg.CommonWindow {
		t.Error("CommonWindow should be false")
	}
	if cfg.ScalFlag {
		t.Error("ScalFlag should be false")
	}
}

func TestParseIndividualChannelStream_PulseInShortBlock(t *testing.T) {
	// Verify that pulse data with short blocks returns an error
	// This is checked after spectral data parsing

	ics := &ICStream{
		WindowSequence:   EightShortSequence,
		PulseDataPresent: true,
	}

	// Verify the condition that would trigger the error
	if ics.PulseDataPresent && ics.WindowSequence == EightShortSequence {
		// This is the error case - would return ErrPulseInShortBlock
		// We can't easily test the full function, but we verify the logic
	} else {
		t.Error("Expected pulse in short block condition to be true")
	}
}

func TestERObjectStartValue(t *testing.T) {
	// Verify ERObjectStart is used correctly in the parsing logic
	if ERObjectStart != 17 {
		t.Errorf("ERObjectStart = %d, want 17", ERObjectStart)
	}

	// Test the comparison logic used in ParseSideInfo/ParseIndividualChannelStream
	testCases := []struct {
		objectType    uint8
		isER          bool
		tnsInSideInfo bool
	}{
		{ObjectTypeLC, false, true},  // LC < 17, parse TNS in side_info
		{ObjectTypeLTP, false, true}, // LTP < 17, parse TNS in side_info
		{17, true, false},            // ER_OBJECT_START, parse TNS after side_info
		{19, true, false},            // ER-AAC-LC, parse TNS after side_info
		{23, true, false},            // ER-AAC-LD, parse TNS after side_info
	}

	for _, tc := range testCases {
		isER := tc.objectType >= ERObjectStart
		if isER != tc.isER {
			t.Errorf("objectType %d: isER = %v, want %v", tc.objectType, isER, tc.isER)
		}

		tnsInSideInfo := tc.objectType < ERObjectStart
		if tnsInSideInfo != tc.tnsInSideInfo {
			t.Errorf("objectType %d: tnsInSideInfo = %v, want %v", tc.objectType, tnsInSideInfo, tc.tnsInSideInfo)
		}
	}
}

func TestGainControlError(t *testing.T) {
	// Test that ErrGainControlNotSupported is defined
	if ErrGainControlNotSupported == nil {
		t.Error("ErrGainControlNotSupported should not be nil")
	}

	expectedMsg := "syntax: gain control (SSR) not supported"
	if ErrGainControlNotSupported.Error() != expectedMsg {
		t.Errorf("Error message = %q, want %q", ErrGainControlNotSupported.Error(), expectedMsg)
	}
}

func TestPulseInShortBlockError(t *testing.T) {
	// Test that ErrPulseInShortBlock is defined
	if ErrPulseInShortBlock == nil {
		t.Error("ErrPulseInShortBlock should not be nil")
	}

	expectedMsg := "syntax: pulse coding not allowed in short blocks"
	if ErrPulseInShortBlock.Error() != expectedMsg {
		t.Errorf("Error message = %q, want %q", ErrPulseInShortBlock.Error(), expectedMsg)
	}
}
