// internal/syntax/adts_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestADTSHeader_Fields(t *testing.T) {
	var h ADTSHeader

	// Fixed header (28 bits)
	h.Syncword = 0x0FFF
	h.ID = 0 // MPEG-4
	h.Layer = 0
	h.ProtectionAbsent = true
	h.Profile = 1 // AAC LC
	h.SFIndex = 4 // 44100 Hz
	h.PrivateBit = false
	h.ChannelConfiguration = 2 // Stereo

	// Variable header
	h.Original = false
	h.Home = false
	h.CopyrightIDBit = false
	h.CopyrightIDStart = false
	h.AACFrameLength = 0
	h.ADTSBufferFullness = 0
	h.CRCCheck = 0
	h.NoRawDataBlocksInFrame = 0

	// Control
	h.OldFormat = false
}

func TestADTSHeader_Syncword(t *testing.T) {
	var h ADTSHeader
	h.Syncword = 0x0FFF

	if h.Syncword != 0x0FFF {
		t.Errorf("Syncword should be 0x0FFF, got 0x%X", h.Syncword)
	}
}

func TestADTSHeader_FrameLength(t *testing.T) {
	var h ADTSHeader

	// Frame length is 13 bits, max 8191
	h.AACFrameLength = 8191
	if h.AACFrameLength != 8191 {
		t.Errorf("AACFrameLength max should be 8191")
	}
}

func TestADTSHeader_HeaderSize(t *testing.T) {
	tests := []struct {
		name             string
		protectionAbsent bool
		want             int
	}{
		{"no CRC", true, 7},
		{"with CRC", false, 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := ADTSHeader{ProtectionAbsent: tt.protectionAbsent}
			if got := h.HeaderSize(); got != tt.want {
				t.Errorf("HeaderSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestADTSHeader_DataSize(t *testing.T) {
	tests := []struct {
		name             string
		protectionAbsent bool
		frameLength      uint16
		want             int
	}{
		{"no CRC, 512 bytes", true, 512, 505},    // 512 - 7 = 505
		{"with CRC, 512 bytes", false, 512, 503}, // 512 - 9 = 503
		{"no CRC, minimum header", true, 7, 0},
		{"with CRC, minimum header", false, 9, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := ADTSHeader{
				ProtectionAbsent: tt.protectionAbsent,
				AACFrameLength:   tt.frameLength,
			}
			if got := h.DataSize(); got != tt.want {
				t.Errorf("DataSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestADTSSyncword(t *testing.T) {
	if ADTSSyncword != 0x0FFF {
		t.Errorf("ADTSSyncword should be 0x0FFF, got 0x%X", ADTSSyncword)
	}
}

func TestFindSyncword_AtStart(t *testing.T) {
	// Valid ADTS frame starts with 0xFFF
	data := []byte{0xFF, 0xF1, 0x4C, 0x80, 0x00, 0x00, 0x00}
	r := bits.NewReader(data)

	err := FindSyncword(r)
	if err != nil {
		t.Fatalf("FindSyncword failed: %v", err)
	}

	// Should have consumed the 12-bit syncword
	consumed := r.GetProcessedBits()
	if consumed != 12 {
		t.Errorf("consumed %d bits, want 12", consumed)
	}
}

func TestFindSyncword_WithGarbage(t *testing.T) {
	// 3 bytes of garbage, then valid ADTS sync
	data := []byte{0x00, 0xAA, 0xBB, 0xFF, 0xF1, 0x4C, 0x80, 0x00}
	r := bits.NewReader(data)

	err := FindSyncword(r)
	if err != nil {
		t.Fatalf("FindSyncword failed: %v", err)
	}

	// Should have skipped 3 bytes (24 bits) + consumed 12-bit syncword = 36 bits
	consumed := r.GetProcessedBits()
	if consumed != 36 {
		t.Errorf("consumed %d bits, want 36", consumed)
	}
}

func TestFindSyncword_NotFound(t *testing.T) {
	// No syncword in data
	data := make([]byte, 800)
	for i := range data {
		data[i] = 0xAA
	}
	r := bits.NewReader(data)

	err := FindSyncword(r)
	if err == nil {
		t.Fatal("expected error for missing syncword")
	}
}

func TestParseFixedHeader(t *testing.T) {
	// Manually construct ADTS fixed header (16 bits after syncword):
	// syncword=0xFFF (12 bits) - already consumed by FindSyncword
	// id=0 (1 bit) - MPEG-4
	// layer=00 (2 bits)
	// protection_absent=1 (1 bit) - no CRC
	// profile=01 (2 bits) - LC
	// sf_index=0011 (4 bits) - 48000 Hz
	// private_bit=0 (1 bit)
	// channel_config=010 (3 bits) - stereo
	// original=0 (1 bit)
	// home=0 (1 bit)
	//
	// Bytes: FF F1 4C 80
	// Binary: 11111111 11110001 01001100 10000000
	// Syncword: 111111111111 (0xFFF)
	// After syncword: 0001 01001100 10000000
	//   ID=0, Layer=00, ProtAbsent=1, Profile=01, SFIndex=0011,
	//   PrivateBit=0, ChannelConfig=010, Original=0, Home=0

	data := []byte{0xFF, 0xF1, 0x4C, 0x80, 0x00, 0x1F, 0xFC}
	r := bits.NewReader(data)

	// Skip syncword (would be done by FindSyncword)
	r.FlushBits(12)

	h := &ADTSHeader{Syncword: ADTSSyncword}
	err := parseFixedHeader(r, h)
	if err != nil {
		t.Fatalf("parseFixedHeader failed: %v", err)
	}

	// Verify parsed values
	if h.ID != 0 {
		t.Errorf("ID = %d, want 0 (MPEG-4)", h.ID)
	}
	if h.Layer != 0 {
		t.Errorf("Layer = %d, want 0", h.Layer)
	}
	if !h.ProtectionAbsent {
		t.Error("ProtectionAbsent = false, want true")
	}
	if h.Profile != 1 {
		t.Errorf("Profile = %d, want 1 (LC)", h.Profile)
	}
	if h.SFIndex != 3 {
		t.Errorf("SFIndex = %d, want 3 (48000Hz)", h.SFIndex)
	}
	if h.ChannelConfiguration != 2 {
		t.Errorf("ChannelConfiguration = %d, want 2 (stereo)", h.ChannelConfiguration)
	}
}

func TestParseVariableHeader(t *testing.T) {
	// Variable header is 28 bits:
	// copyright_id_bit=0 (1 bit)
	// copyright_id_start=0 (1 bit)
	// frame_length=0x0180 (13 bits) = 384 bytes
	// buffer_fullness=0x7FF (11 bits) = VBR marker
	// num_raw_blocks=0 (2 bits) = 1 raw block

	// Bit layout (28 bits total, MSB first):
	// Pos  0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27
	//      c  c  |--------- frame_length (13 bits) ---------|  |-- buffer_fullness (11) --|  nb
	//      0  0  0  0  0  0  1  1  0  0  0  0  0  0  0  1  1  1  1  1  1  1  1  1  1  1  0  0
	//
	// Byte 0 (bits 0-7):   00000011 = 0x03
	// Byte 1 (bits 8-15):  00000001 = 0x01
	// Byte 2 (bits 16-23): 11111111 = 0xFF
	// Byte 3 (bits 24-27): 1100xxxx = 0xC0
	data := []byte{0x03, 0x01, 0xFF, 0xC0}
	r := bits.NewReader(data)

	h := &ADTSHeader{}
	parseVariableHeader(r, h)

	if h.CopyrightIDBit {
		t.Error("CopyrightIDBit = true, want false")
	}
	if h.CopyrightIDStart {
		t.Error("CopyrightIDStart = true, want false")
	}
	if h.AACFrameLength != 384 {
		t.Errorf("AACFrameLength = %d, want 384", h.AACFrameLength)
	}
	if h.ADTSBufferFullness != 0x7FF {
		t.Errorf("ADTSBufferFullness = 0x%X, want 0x7FF", h.ADTSBufferFullness)
	}
	if h.NoRawDataBlocksInFrame != 0 {
		t.Errorf("NoRawDataBlocksInFrame = %d, want 0", h.NoRawDataBlocksInFrame)
	}
}

func TestParseErrorCheck_WithCRC(t *testing.T) {
	// CRC is 16 bits, read when protection_absent=0
	data := []byte{0xAB, 0xCD, 0x00, 0x00}
	r := bits.NewReader(data)

	h := &ADTSHeader{ProtectionAbsent: false}
	parseErrorCheck(r, h)

	if h.CRCCheck != 0xABCD {
		t.Errorf("CRCCheck = 0x%X, want 0xABCD", h.CRCCheck)
	}

	consumed := r.GetProcessedBits()
	if consumed != 16 {
		t.Errorf("consumed %d bits, want 16", consumed)
	}
}

func TestParseErrorCheck_NoCRC(t *testing.T) {
	data := []byte{0xAB, 0xCD, 0x00, 0x00}
	r := bits.NewReader(data)

	h := &ADTSHeader{ProtectionAbsent: true}
	parseErrorCheck(r, h)

	// Should not consume any bits when protection_absent=true
	consumed := r.GetProcessedBits()
	if consumed != 0 {
		t.Errorf("consumed %d bits, want 0 (no CRC)", consumed)
	}
}
