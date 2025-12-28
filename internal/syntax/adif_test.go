package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseADIF_EmptyData(t *testing.T) {
	r := bits.NewReader([]byte{})
	_, err := ParseADIF(r)
	if err == nil {
		t.Error("ParseADIF should return error for empty data")
	}
}

func TestParseADIF_InsufficientData(t *testing.T) {
	// Only 2 bytes - not enough for minimum ADIF header.
	// The Reader pads with zeros and doesn't error until buffer is exhausted.
	// With insufficient data, parsing may succeed but will read garbage (zeros).
	// This test verifies that behavior - the parser reads what it can.
	r := bits.NewReader([]byte{0x00, 0x00})
	h, err := ParseADIF(r)

	// With only 2 bytes (16 bits) of zeros:
	// - copyright_id_present: 0 (1 bit) - no copyright
	// - original_copy: 0 (1 bit)
	// - home: 0 (1 bit)
	// - bitstream_type: 0 (1 bit) - constant rate
	// - bitrate: 0 (23 bits) - zeros from padding
	// - num_pce: 0 (4 bits) - zeros, meaning 1 PCE
	// - Then PCE parsing with zeros...
	// Since we're reading mostly zeros due to padding, it will parse successfully
	// with default values. This is expected behavior.
	if err != nil {
		// If it errors (e.g., from PCE validation), that's also acceptable
		return
	}

	// If it succeeds, verify we got zeros/defaults
	if h.CopyrightIDPresent {
		t.Error("expected no copyright with zero input")
	}
	if h.Bitrate != 0 {
		t.Errorf("expected zero bitrate, got %d", h.Bitrate)
	}
}

func TestParseADIF_NoCopyright_ConstantRate(t *testing.T) {
	// Construct minimal ADIF header:
	// - copyright_id_present: 0 (1 bit)
	// - original_copy: 0 (1 bit)
	// - home: 0 (1 bit)
	// - bitstream_type: 0 (1 bit) = constant rate
	// - bitrate: 128000 = 0x1F400 (23 bits)
	// - num_pce: 0 (4 bits) => 1 PCE
	// - adif_buffer_fullness: 0 (20 bits) - only for constant rate
	// - PCE data follows

	// We need to construct a valid PCE as well
	// For simplicity, build a minimal PCE with 0 elements

	// Bit layout after "ADIF" magic:
	// Bit 0: copyright_id_present = 0
	// Bit 1: original_copy = 0
	// Bit 2: home = 0
	// Bit 3: bitstream_type = 0 (constant rate)
	// Bits 4-26: bitrate = 128000 (0x01F400) - 23 bits
	// Bits 27-30: num_pce = 0 (meaning 1 PCE)
	// Bits 31-50: adif_buffer_fullness = 0 - 20 bits
	// Bits 51+: PCE data

	// Build bytes:
	// Byte 0: 0000 (copyright, orig, home, bstype) + 0000 (bitrate high 4 bits) = 0x00
	// But let's calculate properly...

	// bitrate 128000 = 0x01F400
	// After the first 4 flag bits, we have 23 bits of bitrate
	// So:
	// Bits 0-3: flags (all 0) = 0000
	// Bits 4-26: bitrate = 0000 0001 1111 0100 0000 0000 (23 bits)
	// Bits 27-30: num_pce = 0000 (4 bits)
	// Bits 31-50: buffer_fullness = 0 (20 bits)

	// Let's build this more carefully:
	// Bit positions (0-indexed from MSB):
	// 0: copyright_id_present = 0
	// 1: original_copy = 0
	// 2: home = 0
	// 3: bitstream_type = 0
	// 4-26: bitrate (23 bits) = 128000 = 0x1F400
	// 27-30: num_pce (4 bits) = 0
	// 31-50: adif_buffer_fullness (20 bits) = 0
	// 51+: PCE

	// Construct PCE bits for minimal stereo config:
	// element_instance_tag: 4 bits = 0
	// object_type: 2 bits = 1 (LC)
	// sf_index: 4 bits = 4 (44100 Hz)
	// num_front_channel_elements: 4 bits = 1
	// num_side_channel_elements: 4 bits = 0
	// num_back_channel_elements: 4 bits = 0
	// num_lfe_channel_elements: 2 bits = 0
	// num_assoc_data_elements: 3 bits = 0
	// num_valid_cc_elements: 4 bits = 0
	// mono_mixdown_present: 1 bit = 0
	// stereo_mixdown_present: 1 bit = 0
	// matrix_mixdown_idx_present: 1 bit = 0
	// front_element[0]: is_cpe=1 (1 bit), tag_select=0 (4 bits) = stereo pair
	// Then byte align + comment_field_bytes=0 (8 bits)

	// This is complex - let's build it byte by byte
	data := buildMinimalADIFHeader(t, false, false, false, 0, 128000, 0)

	r := bits.NewReader(data)
	h, err := ParseADIF(r)
	if err != nil {
		t.Fatalf("ParseADIF failed: %v", err)
	}

	if h.CopyrightIDPresent {
		t.Error("CopyrightIDPresent should be false")
	}
	if h.OriginalCopy {
		t.Error("OriginalCopy should be false")
	}
	if h.Home {
		t.Error("Home should be false")
	}
	if h.BitstreamType != 0 {
		t.Errorf("BitstreamType = %d, want 0 (constant rate)", h.BitstreamType)
	}
	if h.Bitrate != 128000 {
		t.Errorf("Bitrate = %d, want 128000", h.Bitrate)
	}
	if h.NumProgramConfigElements != 0 {
		t.Errorf("NumProgramConfigElements = %d, want 0", h.NumProgramConfigElements)
	}
}

func TestParseADIF_WithCopyright(t *testing.T) {
	// Test ADIF header with copyright ID present
	data := buildMinimalADIFHeader(t, true, false, false, 0, 128000, 0)

	r := bits.NewReader(data)
	h, err := ParseADIF(r)
	if err != nil {
		t.Fatalf("ParseADIF failed: %v", err)
	}

	if !h.CopyrightIDPresent {
		t.Error("CopyrightIDPresent should be true")
	}
}

func TestParseADIF_VariableRate(t *testing.T) {
	// Test ADIF header with variable bitrate
	data := buildMinimalADIFHeader(t, false, false, false, 1, 0, 0)

	r := bits.NewReader(data)
	h, err := ParseADIF(r)
	if err != nil {
		t.Fatalf("ParseADIF failed: %v", err)
	}

	if h.BitstreamType != 1 {
		t.Errorf("BitstreamType = %d, want 1 (variable rate)", h.BitstreamType)
	}
	// For variable rate, adif_buffer_fullness is not read
	if !h.IsConstantRate() {
		// Good - variable rate
	} else {
		t.Error("IsConstantRate() should return false for variable rate")
	}
}

func TestADIFHeader_IsConstantRate(t *testing.T) {
	tests := []struct {
		name          string
		bitstreamType uint8
		want          bool
	}{
		{"constant rate", 0, true},
		{"variable rate", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ADIFHeader{BitstreamType: tt.bitstreamType}
			if got := h.IsConstantRate(); got != tt.want {
				t.Errorf("IsConstantRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestADIFHeader_HasCopyrightID(t *testing.T) {
	t.Run("no copyright", func(t *testing.T) {
		h := &ADIFHeader{CopyrightIDPresent: false}
		id, present := h.HasCopyrightID()
		if present {
			t.Error("HasCopyrightID should return false when not present")
		}
		if id != "" {
			t.Errorf("HasCopyrightID should return empty string, got %q", id)
		}
	})

	t.Run("with copyright", func(t *testing.T) {
		h := &ADIFHeader{
			CopyrightIDPresent: true,
			CopyrightID:        [10]int8{'T', 'E', 'S', 'T', 0, 0, 0, 0, 0, 0},
		}
		id, present := h.HasCopyrightID()
		if !present {
			t.Error("HasCopyrightID should return true when present")
		}
		if id != "TEST" {
			t.Errorf("HasCopyrightID = %q, want %q", id, "TEST")
		}
	})
}

func TestADIFHeader_GetPCEs(t *testing.T) {
	h := &ADIFHeader{
		NumProgramConfigElements: 1, // means 2 PCEs
		PCE: [16]ProgramConfig{
			{Channels: 2, SFIndex: 4},
			{Channels: 6, SFIndex: 3},
		},
	}

	pces := h.GetPCEs()
	if len(pces) != 2 {
		t.Errorf("GetPCEs() returned %d elements, want 2", len(pces))
	}
	if pces[0].Channels != 2 {
		t.Errorf("pces[0].Channels = %d, want 2", pces[0].Channels)
	}
	if pces[1].Channels != 6 {
		t.Errorf("pces[1].Channels = %d, want 6", pces[1].Channels)
	}
}

// buildMinimalADIFHeader constructs a minimal valid ADIF header for testing.
// It creates a header with a single stereo PCE.
func buildMinimalADIFHeader(t *testing.T, copyrightPresent, originalCopy, home bool, bitstreamType uint8, bitrate uint32, numPCE uint8) []byte {
	t.Helper()

	// Use a simple bit buffer to construct the header
	buf := make([]byte, 256)
	bitPos := 0

	writeBit := func(b bool) {
		byteIdx := bitPos / 8
		bitIdx := 7 - (bitPos % 8)
		if b {
			buf[byteIdx] |= 1 << bitIdx
		}
		bitPos++
	}

	writeBits := func(val uint32, n int) {
		for i := n - 1; i >= 0; i-- {
			writeBit((val>>i)&1 == 1)
		}
	}

	// ADIF header fields (after "ADIF" magic which is already consumed)
	writeBit(copyrightPresent)

	if copyrightPresent {
		// Write 9 bytes (72 bits) of copyright ID - use "COPYRIGHT" padded
		copyright := []byte("COPYRIGHT")
		for i := 0; i < 9; i++ {
			if i < len(copyright) {
				writeBits(uint32(copyright[i]), 8)
			} else {
				writeBits(0, 8)
			}
		}
	}

	writeBit(originalCopy)
	writeBit(home)
	writeBit(bitstreamType == 1)
	writeBits(bitrate, 23)
	writeBits(uint32(numPCE), 4)

	// For each PCE
	for i := uint8(0); i <= numPCE; i++ {
		// adif_buffer_fullness (only for constant rate)
		if bitstreamType == 0 {
			writeBits(0, 20)
		}

		// Minimal PCE for stereo:
		writeBits(0, 4) // element_instance_tag
		writeBits(1, 2) // object_type (LC)
		writeBits(4, 4) // sf_index (44100)
		writeBits(1, 4) // num_front_channel_elements = 1
		writeBits(0, 4) // num_side_channel_elements = 0
		writeBits(0, 4) // num_back_channel_elements = 0
		writeBits(0, 2) // num_lfe_channel_elements = 0
		writeBits(0, 3) // num_assoc_data_elements = 0
		writeBits(0, 4) // num_valid_cc_elements = 0
		writeBit(false) // mono_mixdown_present
		writeBit(false) // stereo_mixdown_present
		writeBit(false) // matrix_mixdown_idx_present

		// Front element: CPE + tag
		writeBit(true)  // is_cpe = 1 (stereo pair)
		writeBits(0, 4) // tag_select = 0

		// Byte align
		remainder := bitPos % 8
		if remainder != 0 {
			for j := 0; j < 8-remainder; j++ {
				writeBit(false)
			}
		}

		// Comment field
		writeBits(0, 8) // comment_field_bytes = 0
	}

	// Return only the used bytes
	numBytes := (bitPos + 7) / 8
	return buf[:numBytes]
}

func TestADIFHeader_Fields(t *testing.T) {
	var h ADIFHeader

	h.CopyrightIDPresent = false
	h.OriginalCopy = false
	h.Bitrate = 0
	h.ADIFBufferFullness = 0
	h.NumProgramConfigElements = 0
	h.Home = false
	h.BitstreamType = 0
}

func TestADIFHeader_CopyrightID(t *testing.T) {
	var h ADIFHeader

	// Copyright ID is 10 bytes (per FAAD2 structs.h:173)
	if len(h.CopyrightID) != 10 {
		t.Errorf("CopyrightID should have 10 bytes, got %d", len(h.CopyrightID))
	}
}

func TestADIFHeader_PCEs(t *testing.T) {
	var h ADIFHeader

	// Up to 16 PCEs
	if len(h.PCE) != 16 {
		t.Errorf("PCE should have 16 elements, got %d", len(h.PCE))
	}

	// Each PCE should be a ProgramConfig
	h.PCE[0].Channels = 2
}

func TestCheckADIFMagic_Valid(t *testing.T) {
	data := []byte{'A', 'D', 'I', 'F', 0x00, 0x00, 0x00, 0x00}
	r := bits.NewReader(data)
	if !CheckADIFMagic(r) {
		t.Error("CheckADIFMagic should return true for valid ADIF magic")
	}
}

func TestCheckADIFMagic_Invalid(t *testing.T) {
	data := []byte{0xFF, 0xF1, 0x50, 0x80} // ADTS syncword
	r := bits.NewReader(data)
	if CheckADIFMagic(r) {
		t.Error("CheckADIFMagic should return false for non-ADIF data")
	}
}

func TestCheckADIFMagic_Short(t *testing.T) {
	data := []byte{'A', 'D', 'I'} // Too short
	r := bits.NewReader(data)
	if CheckADIFMagic(r) {
		t.Error("CheckADIFMagic should return false for short data")
	}
}

func TestCheckADIFMagic_ConsumesBytes(t *testing.T) {
	// When magic is found, bytes should be consumed
	data := []byte{'A', 'D', 'I', 'F', 0xAB, 0xCD}
	r := bits.NewReader(data)
	if !CheckADIFMagic(r) {
		t.Fatal("CheckADIFMagic should return true for valid ADIF magic")
	}
	// After consuming "ADIF", next byte should be 0xAB
	nextByte := r.GetBits(8)
	if nextByte != 0xAB {
		t.Errorf("After consuming magic, expected 0xAB, got 0x%02X", nextByte)
	}
}

func TestCheckADIFMagic_DoesNotConsumeOnMismatch(t *testing.T) {
	// When magic is NOT found, bytes should NOT be consumed
	data := []byte{0xFF, 0xF1, 0x50, 0x80} // ADTS syncword
	r := bits.NewReader(data)
	if CheckADIFMagic(r) {
		t.Fatal("CheckADIFMagic should return false for non-ADIF data")
	}
	// Bytes should not be consumed; next read should still get 0xFF
	nextByte := r.GetBits(8)
	if nextByte != 0xFF {
		t.Errorf("After failed check, expected 0xFF (not consumed), got 0x%02X", nextByte)
	}
}
