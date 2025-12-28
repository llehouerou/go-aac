// internal/syntax/fill_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseExcludedChannels_SevenChannels(t *testing.T) {
	// excluded_channels format (Table 4.4.32):
	// - exclude_mask[0-6]: 7 x 1 bit
	// - additional_excluded_chns: 1 bit (0 = no more)
	//
	// Test case: 7 channels with mask 1010101, no additional
	// Binary: 1010101 0 = 0xAA

	data := []byte{0xAA}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseExcludedChannels(r, drc)

	if bytesRead != 1 {
		t.Errorf("bytesRead = %d, want 1", bytesRead)
	}

	// Check exclude mask
	expected := []uint8{1, 0, 1, 0, 1, 0, 1}
	for i := 0; i < 7; i++ {
		if drc.ExcludeMask[i] != expected[i] {
			t.Errorf("ExcludeMask[%d] = %d, want %d", i, drc.ExcludeMask[i], expected[i])
		}
	}
}

func TestParseExcludedChannels_Extended(t *testing.T) {
	// Test with additional excluded channels
	// - exclude_mask[0-6]: 1111111 (7 bits)
	// - additional_excluded_chns: 1 (continue)
	// - exclude_mask[7-13]: 0000000 (7 bits)
	// - additional_excluded_chns: 0 (stop)
	//
	// Binary: 1111111 1 0000000 0 = 0xFF 0x00

	data := []byte{0xFF, 0x00}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseExcludedChannels(r, drc)

	if bytesRead != 2 {
		t.Errorf("bytesRead = %d, want 2", bytesRead)
	}

	// First 7 channels excluded
	for i := 0; i < 7; i++ {
		if drc.ExcludeMask[i] != 1 {
			t.Errorf("ExcludeMask[%d] = %d, want 1", i, drc.ExcludeMask[i])
		}
	}

	// Next 7 channels not excluded
	for i := 7; i < 14; i++ {
		if drc.ExcludeMask[i] != 0 {
			t.Errorf("ExcludeMask[%d] = %d, want 0", i, drc.ExcludeMask[i])
		}
	}

	// Additional excluded flags
	if drc.AdditionalExcludedChns[0] != 1 {
		t.Errorf("AdditionalExcludedChns[0] = %d, want 1", drc.AdditionalExcludedChns[0])
	}
	if drc.AdditionalExcludedChns[1] != 0 {
		t.Errorf("AdditionalExcludedChns[1] = %d, want 0", drc.AdditionalExcludedChns[1])
	}
}

func TestParseDynamicRangeInfo_Minimal(t *testing.T) {
	// Minimal DRC info:
	// - has_instance_tag: 1 bit = 0 (no instance tag)
	// - excluded_chns_present: 1 bit = 0 (no excluded channels)
	// - has_bands_data: 1 bit = 0 (no band data, single band)
	// - has_prog_ref_level: 1 bit = 0 (no program reference level)
	// - dyn_rng_sgn[0]: 1 bit = 0
	// - dyn_rng_ctl[0]: 7 bits = 0x55 (85)
	//
	// Binary: 0 0 0 0 0 1010101
	// Byte 0: 0000_0101 = 0x05
	// Byte 1: 0101_xxxx = 0x50
	//
	// n starts at 1, then +1 for single band dyn_rng = 2

	data := []byte{0x05, 0x50}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseDynamicRangeInfo(r, drc)

	if bytesRead != 2 {
		t.Errorf("bytesRead = %d, want 2", bytesRead)
	}

	if drc.NumBands != 1 {
		t.Errorf("NumBands = %d, want 1", drc.NumBands)
	}

	if drc.DynRngSgn[0] != 0 {
		t.Errorf("DynRngSgn[0] = %d, want 0", drc.DynRngSgn[0])
	}

	if drc.DynRngCtl[0] != 0x55 {
		t.Errorf("DynRngCtl[0] = %d, want 85", drc.DynRngCtl[0])
	}
}

func TestParseDynamicRangeInfo_WithInstanceTag(t *testing.T) {
	// DRC with instance tag:
	// - has_instance_tag: 1 bit = 1
	// - pce_instance_tag: 4 bits = 0x5
	// - reserved: 4 bits = 0
	// - excluded_chns_present: 1 bit = 0
	// - has_bands_data: 1 bit = 0
	// - has_prog_ref_level: 1 bit = 0
	// - dyn_rng_sgn[0]: 1 bit = 1
	// - dyn_rng_ctl[0]: 7 bits = 0x7F (127)
	//
	// Binary: 1 0101 0000 0 0 0 1 1111111
	// Byte 0: 1010_1000 = 0xA8
	// Byte 1: 0000_1111 = 0x0F
	// Byte 2: 1111_xxxx = 0xF0
	//
	// n starts at 1, +1 for instance_tag = 2, +1 for dyn_rng = 3

	data := []byte{0xA8, 0x0F, 0xF0}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseDynamicRangeInfo(r, drc)

	if bytesRead != 3 {
		t.Errorf("bytesRead = %d, want 3", bytesRead)
	}

	if drc.PCEInstanceTag != 5 {
		t.Errorf("PCEInstanceTag = %d, want 5", drc.PCEInstanceTag)
	}

	if drc.DynRngSgn[0] != 1 {
		t.Errorf("DynRngSgn[0] = %d, want 1", drc.DynRngSgn[0])
	}

	if drc.DynRngCtl[0] != 0x7F {
		t.Errorf("DynRngCtl[0] = %d, want 127", drc.DynRngCtl[0])
	}
}

func TestParseDynamicRangeInfo_WithProgRefLevel(t *testing.T) {
	// DRC with program reference level:
	// - has_instance_tag: 1 bit = 0
	// - excluded_chns_present: 1 bit = 0
	// - has_bands_data: 1 bit = 0
	// - has_prog_ref_level: 1 bit = 1
	// - prog_ref_level: 7 bits = 0x40 (64)
	// - reserved: 1 bit = 0
	// - dyn_rng_sgn[0]: 1 bit = 0
	// - dyn_rng_ctl[0]: 7 bits = 0x20 (32)
	//
	// Binary: 0 0 0 1 1000000 0 0 0100000
	// Byte 0: 0001_1000 = 0x18
	// Byte 1: 0000_0010 = 0x02
	// Byte 2: 0000_0xxx = 0x00
	//
	// n starts at 1, +1 for prog_ref_level = 2, +1 for dyn_rng = 3

	data := []byte{0x18, 0x02, 0x00}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseDynamicRangeInfo(r, drc)

	if bytesRead != 3 {
		t.Errorf("bytesRead = %d, want 3", bytesRead)
	}

	if drc.ProgRefLevel != 64 {
		t.Errorf("ProgRefLevel = %d, want 64", drc.ProgRefLevel)
	}

	if drc.DynRngCtl[0] != 32 {
		t.Errorf("DynRngCtl[0] = %d, want 32", drc.DynRngCtl[0])
	}
}

func TestParseDynamicRangeInfo_MultiBand(t *testing.T) {
	// DRC with multiple bands:
	// - has_instance_tag: 1 bit = 0
	// - excluded_chns_present: 1 bit = 0
	// - has_bands_data: 1 bit = 1
	// - band_incr: 4 bits = 2 (num_bands = 1 + 2 = 3)
	// - drc_bands_reserved_bits: 4 bits = 0
	// - band_top[0]: 8 bits = 10 (0x0A)
	// - band_top[1]: 8 bits = 20 (0x14)
	// - band_top[2]: 8 bits = 30 (0x1E)
	// - has_prog_ref_level: 1 bit = 0
	// - dyn_rng_sgn[0]: 1 bit = 0, dyn_rng_ctl[0]: 7 bits = 10
	// - dyn_rng_sgn[1]: 1 bit = 0, dyn_rng_ctl[1]: 7 bits = 20
	// - dyn_rng_sgn[2]: 1 bit = 0, dyn_rng_ctl[2]: 7 bits = 30
	//
	// Bit layout:
	// 0 0 1 0010 0000 00001010 00010100 00011110 0 00001010 00010100 00011110
	//
	// Byte 0: 0 0 1 0010 0 = 0010_0100 = 0x24
	// Byte 1: 000 00001 = 0000_0001 = 0x01
	// Byte 2: 010 00010 = 0100_0010 = 0x42
	// Byte 3: 100 00011 = 1000_0011 = 0x83
	// Byte 4: 110 0 0000 = 1100_0000 = 0xC0
	// Byte 5: 1010 0001 = 1010_0001 = 0xA1
	// Byte 6: 0100 0001 = 0100_0001 = 0x41
	// Byte 7: 1110 xxxx = 1110_0000 = 0xE0
	//
	// n count: start=1, +1 bands_data, +3 band_top, +3 dyn_rng = 8

	data := []byte{0x24, 0x01, 0x42, 0x83, 0xC0, 0xA1, 0x41, 0xE0}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseDynamicRangeInfo(r, drc)

	if bytesRead != 8 {
		t.Errorf("bytesRead = %d, want 8", bytesRead)
	}

	if drc.NumBands != 3 {
		t.Errorf("NumBands = %d, want 3", drc.NumBands)
	}

	if drc.BandTop[0] != 10 || drc.BandTop[1] != 20 || drc.BandTop[2] != 30 {
		t.Errorf("BandTop = %v, want [10, 20, 30]", drc.BandTop[:3])
	}

	if drc.DynRngCtl[0] != 10 || drc.DynRngCtl[1] != 20 || drc.DynRngCtl[2] != 30 {
		t.Errorf("DynRngCtl = %v, want [10, 20, 30]", drc.DynRngCtl[:3])
	}
}

func TestParseExtensionPayload_DynamicRange(t *testing.T) {
	// Extension type: EXT_DYNAMIC_RANGE (11)
	// Followed by minimal DRC data
	//
	// extension_type: 4 bits = 1011 (11)
	// Then dynamic_range_info:
	// - has_instance_tag: 0
	// - excluded_chns_present: 0
	// - has_bands_data: 0
	// - has_prog_ref_level: 0
	// - dyn_rng_sgn: 0
	// - dyn_rng_ctl: 7 bits = 0x55 (85)

	data := []byte{0xB0, 0x55}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseExtensionPayload(r, drc, 2)

	// dynamic_range_info returns 2 (n starts at 1, plus 1 for dyn_rng)
	if bytesRead != 2 {
		t.Errorf("bytesRead = %d, want 2", bytesRead)
	}

	if !drc.Present {
		t.Error("drc.Present should be true")
	}

	if drc.DynRngCtl[0] != 0x55 {
		t.Errorf("DynRngCtl[0] = %d, want 85", drc.DynRngCtl[0])
	}
}

func TestParseExtensionPayload_FillData(t *testing.T) {
	// Extension type: EXT_FILL_DATA (1)
	// Followed by fill_nibble (4 bits = 0000) + fill_bytes
	//
	// extension_type: 4 bits = 0001
	// fill_nibble: 4 bits = 0000
	// fill_byte[0]: 8 bits = 0xA5
	// fill_byte[1]: 8 bits = 0xA5

	data := []byte{0x10, 0xA5, 0xA5}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseExtensionPayload(r, drc, 3)

	// EXT_FILL_DATA returns count
	if bytesRead != 3 {
		t.Errorf("bytesRead = %d, want 3", bytesRead)
	}
}

func TestParseExtensionPayload_Filler(t *testing.T) {
	// Extension type: EXT_FIL (0)
	// Just reads fill_nibble (4 bits) + remaining bytes

	data := []byte{0x00, 0xFF}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseExtensionPayload(r, drc, 2)

	if bytesRead != 2 {
		t.Errorf("bytesRead = %d, want 2", bytesRead)
	}
}

func TestParseExtensionPayload_DataElement(t *testing.T) {
	// Extension type: EXT_DATA_ELEMENT (2)
	// data_element_version: 4 bits = 0 (ANC_DATA)
	// dataElementLengthPart: 8 bits = 5 (length)
	// data_element_byte[0-4]: 5 bytes

	data := []byte{0x20, 0x05, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseExtensionPayload(r, drc, 10)

	// dataElementLength=5, loopCounter=1, +1 = 7
	if bytesRead != 7 {
		t.Errorf("bytesRead = %d, want 7", bytesRead)
	}
}

func TestParseFillElement_Empty(t *testing.T) {
	// count = 0, no payload
	// count: 4 bits = 0000

	data := []byte{0x00}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	err := ParseFillElement(r, drc)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseFillElement_SmallCount(t *testing.T) {
	// count = 3 (small, no extension)
	// count: 4 bits = 0011
	// Followed by 3 bytes of extension_payload (EXT_FIL with fill data)

	data := []byte{0x30, 0x0F, 0xFF}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	err := ParseFillElement(r, drc)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseFillElement_ExtendedCount(t *testing.T) {
	// count = 15 (triggers extended count)
	// count: 4 bits = 1111 (15)
	// extra_count: 8 bits = 5 (total = 15 + 5 - 1 = 19 bytes)
	// Followed by 19 bytes of extension_payload

	data := make([]byte, 25)
	data[0] = 0xF0 // count=15
	data[1] = 0x50 // extra=5, ext_type=0 start
	data[2] = 0x00 // ext_type=0 end, fill_nibble
	// Rest are fill bytes

	r := bits.NewReader(data)
	drc := &DRCInfo{}

	err := ParseFillElement(r, drc)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseFillElement_WithDRC(t *testing.T) {
	// Fill element containing DRC extension
	// count: 4 bits = 3
	// ext_type: 4 bits = 1011 (EXT_DYNAMIC_RANGE = 11)
	// dynamic_range_info (minimal: 2 bytes consumed)

	data := []byte{0x3B, 0x05, 0x50}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	err := ParseFillElement(r, drc)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !drc.Present {
		t.Error("drc.Present should be true")
	}

	if drc.DynRngCtl[0] != 0x55 {
		t.Errorf("DynRngCtl[0] = %d, want 85", drc.DynRngCtl[0])
	}
}
