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
