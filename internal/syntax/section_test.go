// internal/syntax/section_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseSectionData_SingleSection(t *testing.T) {
	// Long window, max_sfb = 4, single section using codebook 1
	// sect_cb: 4 bits = 0001 (codebook 1)
	// sect_len_incr: 5 bits = 00100 (4) - covers all 4 SFBs
	// Total: 9 bits
	// Bits: 0001 00100 = 0b0001_0010_0 padded = 0x12
	data := []byte{0x12, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		MaxSFB:          4,
		NumWindowGroups: 1,
	}

	err := ParseSectionData(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 1 section covering SFBs 0-3
	if ics.NumSec[0] != 1 {
		t.Errorf("NumSec[0]: got %d, want 1", ics.NumSec[0])
	}
	if ics.SectCB[0][0] != 1 {
		t.Errorf("SectCB[0][0]: got %d, want 1", ics.SectCB[0][0])
	}
	if ics.SectStart[0][0] != 0 {
		t.Errorf("SectStart[0][0]: got %d, want 0", ics.SectStart[0][0])
	}
	if ics.SectEnd[0][0] != 4 {
		t.Errorf("SectEnd[0][0]: got %d, want 4", ics.SectEnd[0][0])
	}

	// Check SFBCB (codebook per SFB)
	for sfb := uint8(0); sfb < 4; sfb++ {
		if ics.SFBCB[0][sfb] != 1 {
			t.Errorf("SFBCB[0][%d]: got %d, want 1", sfb, ics.SFBCB[0][sfb])
		}
	}
}

func TestParseSectionData_MultipleSections(t *testing.T) {
	// Long window, max_sfb = 6, two sections:
	// Section 1: codebook 0 (zero), length 2
	// Section 2: codebook 1, length 4
	// sect_cb: 0000 (codebook 0)
	// sect_len_incr: 00010 (2)
	// sect_cb: 0001 (codebook 1)
	// sect_len_incr: 00100 (4)
	// Total: 4+5+4+5 = 18 bits
	// Bits: 0000 00010 0001 00100 = 0b0000_0001_0000_1001_00
	data := []byte{0x01, 0x09, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		MaxSFB:          6,
		NumWindowGroups: 1,
	}

	err := ParseSectionData(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumSec[0] != 2 {
		t.Errorf("NumSec[0]: got %d, want 2", ics.NumSec[0])
	}

	// Section 0: codebook 0, SFBs 0-1
	if ics.SectCB[0][0] != 0 {
		t.Errorf("SectCB[0][0]: got %d, want 0", ics.SectCB[0][0])
	}
	if ics.SectEnd[0][0] != 2 {
		t.Errorf("SectEnd[0][0]: got %d, want 2", ics.SectEnd[0][0])
	}

	// Section 1: codebook 1, SFBs 2-5
	if ics.SectCB[0][1] != 1 {
		t.Errorf("SectCB[0][1]: got %d, want 1", ics.SectCB[0][1])
	}
	if ics.SectStart[0][1] != 2 {
		t.Errorf("SectStart[0][1]: got %d, want 2", ics.SectStart[0][1])
	}
	if ics.SectEnd[0][1] != 6 {
		t.Errorf("SectEnd[0][1]: got %d, want 6", ics.SectEnd[0][1])
	}

	// Check SFBCB assignments
	for sfb := uint8(0); sfb < 2; sfb++ {
		if ics.SFBCB[0][sfb] != 0 {
			t.Errorf("SFBCB[0][%d]: got %d, want 0", sfb, ics.SFBCB[0][sfb])
		}
	}
	for sfb := uint8(2); sfb < 6; sfb++ {
		if ics.SFBCB[0][sfb] != 1 {
			t.Errorf("SFBCB[0][%d]: got %d, want 1", sfb, ics.SFBCB[0][sfb])
		}
	}
}

func TestParseSectionData_ShortWindow(t *testing.T) {
	// Short window sequence, max_sfb = 4, single section using codebook 2
	// For short windows: sect_bits = 3, sect_esc_val = 7
	// sect_cb: 4 bits = 0010 (codebook 2)
	// sect_len_incr: 3 bits = 100 (4) - covers all 4 SFBs
	// Total: 7 bits
	// Bits: 0010 100 = 0b0010_1000 = 0x28
	data := []byte{0x28, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence:  EightShortSequence,
		MaxSFB:          4,
		NumWindowGroups: 1,
	}

	err := ParseSectionData(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumSec[0] != 1 {
		t.Errorf("NumSec[0]: got %d, want 1", ics.NumSec[0])
	}
	if ics.SectCB[0][0] != 2 {
		t.Errorf("SectCB[0][0]: got %d, want 2", ics.SectCB[0][0])
	}
	if ics.SectEnd[0][0] != 4 {
		t.Errorf("SectEnd[0][0]: got %d, want 4", ics.SectEnd[0][0])
	}
}

func TestParseSectionData_NoiseCodebook(t *testing.T) {
	// Long window, max_sfb = 2, single section using noise codebook (13)
	// sect_cb: 4 bits = 1101 (codebook 13 = NoiseHCB)
	// sect_len_incr: 5 bits = 00010 (2)
	// Bits: 1101 00010 = 0b1101_0001_0 = 0xD1
	data := []byte{0xD1, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		MaxSFB:          2,
		NumWindowGroups: 1,
	}

	err := ParseSectionData(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ics.NoiseUsed {
		t.Error("NoiseUsed should be true")
	}
	if ics.IsUsed {
		t.Error("IsUsed should be false")
	}
}

func TestParseSectionData_IntensityCodebook(t *testing.T) {
	// Long window, max_sfb = 2, single section using intensity codebook (15)
	// sect_cb: 4 bits = 1111 (codebook 15 = IntensityHCB)
	// sect_len_incr: 5 bits = 00010 (2)
	// Bits: 1111 00010 = 0b1111_0001_0 = 0xF1
	data := []byte{0xF1, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		MaxSFB:          2,
		NumWindowGroups: 1,
	}

	err := ParseSectionData(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NoiseUsed {
		t.Error("NoiseUsed should be false")
	}
	if !ics.IsUsed {
		t.Error("IsUsed should be true")
	}
}

func TestParseSectionData_IntensityCodebook2(t *testing.T) {
	// Long window, max_sfb = 2, single section using intensity codebook 2 (14)
	// sect_cb: 4 bits = 1110 (codebook 14 = IntensityHCB2)
	// sect_len_incr: 5 bits = 00010 (2)
	// Bits: 1110 00010 = 0b1110_0001_0 = 0xE1
	data := []byte{0xE1, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		MaxSFB:          2,
		NumWindowGroups: 1,
	}

	err := ParseSectionData(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ics.IsUsed {
		t.Error("IsUsed should be true for IntensityHCB2")
	}
}

func TestParseSectionData_ReservedCodebook(t *testing.T) {
	// Long window, max_sfb = 2, using reserved codebook 12
	// sect_cb: 4 bits = 1100 (codebook 12 = reserved)
	// Bits: 1100 xxxxx = 0xC0
	data := []byte{0xC0, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		MaxSFB:          2,
		NumWindowGroups: 1,
	}

	err := ParseSectionData(r, ics)
	if err != ErrReservedCodebook {
		t.Errorf("expected ErrReservedCodebook, got %v", err)
	}
}

func TestParseSectionData_EscapedLength(t *testing.T) {
	// Long window, max_sfb = 36, single section with escaped length
	// For long windows: sect_bits = 5, sect_esc_val = 31
	// sect_cb: 4 bits = 0011 (codebook 3)
	// sect_len_incr: 5 bits = 11111 (31 = escape, keep reading)
	// sect_len_incr: 5 bits = 00101 (5)
	// Total length = 31 + 5 = 36
	// Bits: 0011 11111 00101 = 0b0011_1111_1001_01 = 0x3F 0x94
	data := []byte{0x3F, 0x94}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		MaxSFB:          36,
		NumWindowGroups: 1,
	}

	err := ParseSectionData(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumSec[0] != 1 {
		t.Errorf("NumSec[0]: got %d, want 1", ics.NumSec[0])
	}
	if ics.SectEnd[0][0] != 36 {
		t.Errorf("SectEnd[0][0]: got %d, want 36", ics.SectEnd[0][0])
	}
}

func TestParseSectionData_MultipleWindowGroups(t *testing.T) {
	// Short window with 2 window groups, max_sfb = 2 each
	// Group 0: codebook 1, length 2
	// Group 1: codebook 2, length 2
	// sect_cb: 4 bits = 0001 (codebook 1)
	// sect_len_incr: 3 bits = 010 (2)
	// sect_cb: 4 bits = 0010 (codebook 2)
	// sect_len_incr: 3 bits = 010 (2)
	// Total: 14 bits
	// Bits: 0001 010 0010 010 = 0b0001_0100_0100_10 = 0x14 0x48
	data := []byte{0x14, 0x48}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence:  EightShortSequence,
		MaxSFB:          2,
		NumWindowGroups: 2,
	}

	err := ParseSectionData(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Group 0
	if ics.NumSec[0] != 1 {
		t.Errorf("NumSec[0]: got %d, want 1", ics.NumSec[0])
	}
	if ics.SectCB[0][0] != 1 {
		t.Errorf("SectCB[0][0]: got %d, want 1", ics.SectCB[0][0])
	}

	// Group 1
	if ics.NumSec[1] != 1 {
		t.Errorf("NumSec[1]: got %d, want 1", ics.NumSec[1])
	}
	if ics.SectCB[1][0] != 2 {
		t.Errorf("SectCB[1][0]: got %d, want 2", ics.SectCB[1][0])
	}
}

func TestParseSectionData_ZeroMaxSFB(t *testing.T) {
	// Edge case: max_sfb = 0 means no sections needed
	data := []byte{0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		MaxSFB:          0,
		NumWindowGroups: 1,
	}

	err := ParseSectionData(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumSec[0] != 0 {
		t.Errorf("NumSec[0]: got %d, want 0", ics.NumSec[0])
	}
}
