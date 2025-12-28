// internal/syntax/scalefactor_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
	"github.com/llehouerou/go-aac/internal/huffman"
)

func TestDecodeScaleFactors_AllZero(t *testing.T) {
	// Global gain = 100, single window group, max_sfb = 2
	// Both SFBs use zero codebook -> scale factors should be 0
	ics := &ICStream{
		GlobalGain:      100,
		NumWindowGroups: 1,
		MaxSFB:          2,
	}
	ics.SFBCB[0][0] = 0 // Zero codebook
	ics.SFBCB[0][1] = 0 // Zero codebook

	// No bits needed for zero codebook
	data := []byte{0x00}
	r := bits.NewReader(data)

	err := DecodeScaleFactors(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.ScaleFactors[0][0] != 0 {
		t.Errorf("ScaleFactors[0][0]: got %d, want 0", ics.ScaleFactors[0][0])
	}
	if ics.ScaleFactors[0][1] != 0 {
		t.Errorf("ScaleFactors[0][1]: got %d, want 0", ics.ScaleFactors[0][1])
	}
}

func TestDecodeScaleFactors_Spectral(t *testing.T) {
	// Test spectral scale factors with known Huffman patterns.
	// The scale factor Huffman codebook has specific bit patterns.
	// Delta 0 (index 60) encodes to a specific codeword.
	//
	// From FAAD2's hcb_sf table, we can trace the tree:
	// - Start at offset 0
	// - The "zero delta" path leads to value 60
	//
	// For this test, we use a pattern that produces valid Huffman codes
	// and verify the differential decoding logic works correctly.

	ics := &ICStream{
		GlobalGain:      100,
		NumWindowGroups: 1,
		MaxSFB:          2,
	}
	ics.SFBCB[0][0] = 1 // Spectral codebook
	ics.SFBCB[0][1] = 1 // Spectral codebook

	// Use a bit pattern that produces valid scale factor Huffman codes.
	// We'll provide enough data and verify the result is in valid range.
	// Detailed validation is done via FAAD2 reference comparison tests.
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	r := bits.NewReader(data)

	err := DecodeScaleFactors(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify scale factors are in valid range
	for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
		sf := ics.ScaleFactors[0][sfb]
		if sf < 0 || sf > 255 {
			t.Errorf("ScaleFactors[0][%d] out of range: %d", sfb, sf)
		}
	}
}

func TestDecodeScaleFactors_IntensityStereo(t *testing.T) {
	// Test intensity stereo scale factors.
	// Intensity stereo uses codebooks 14 and 15.

	ics := &ICStream{
		GlobalGain:      100,
		NumWindowGroups: 1,
		MaxSFB:          2,
	}
	ics.SFBCB[0][0] = uint8(huffman.IntensityHCB)  // 15
	ics.SFBCB[0][1] = uint8(huffman.IntensityHCB2) // 14

	// Provide data for Huffman decoding
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	r := bits.NewReader(data)

	err := DecodeScaleFactors(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// IS position values can be negative (no 0-255 constraint)
	// Just verify no error occurred - detailed testing via FAAD2 comparison
}

func TestDecodeScaleFactors_Noise(t *testing.T) {
	// Test PNS (noise) scale factors.
	// First noise uses 9-bit PCM, subsequent use Huffman delta.

	ics := &ICStream{
		GlobalGain:      100, // noiseEnergy starts at 100 - 90 = 10
		NumWindowGroups: 1,
		MaxSFB:          3,
	}
	ics.SFBCB[0][0] = uint8(huffman.NoiseHCB) // 13 - first noise (PCM)
	ics.SFBCB[0][1] = uint8(huffman.NoiseHCB) // 13 - second noise (Huffman)
	ics.SFBCB[0][2] = uint8(huffman.NoiseHCB) // 13 - third noise (Huffman)

	// First noise: 9 bits PCM
	// Data: 0x80, 0x00 = 0b10000000_00000000
	// First 9 bits = 0b100000000 = 256
	// t = 256 - 256 = 0
	// noiseEnergy = 10 + 0 = 10
	// Remaining bits are for Huffman deltas
	data := []byte{0x80, 0x00, 0xFF, 0xFF} // First 9 bits = 256
	r := bits.NewReader(data)

	err := DecodeScaleFactors(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First noise should be 10 (global_gain - 90 + (256 - 256))
	if ics.ScaleFactors[0][0] != 10 {
		t.Errorf("ScaleFactors[0][0]: got %d, want 10", ics.ScaleFactors[0][0])
	}
}

func TestDecodeScaleFactors_MultipleWindowGroups(t *testing.T) {
	// Test with multiple window groups (short windows).

	ics := &ICStream{
		GlobalGain:      100,
		NumWindowGroups: 2,
		MaxSFB:          2,
	}
	// All zero codebooks
	ics.SFBCB[0][0] = 0
	ics.SFBCB[0][1] = 0
	ics.SFBCB[1][0] = 0
	ics.SFBCB[1][1] = 0

	data := []byte{0x00}
	r := bits.NewReader(data)

	err := DecodeScaleFactors(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All should be 0
	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
			if ics.ScaleFactors[g][sfb] != 0 {
				t.Errorf("ScaleFactors[%d][%d]: got %d, want 0", g, sfb, ics.ScaleFactors[g][sfb])
			}
		}
	}
}

func TestDecodeScaleFactors_MixedCodebooks(t *testing.T) {
	// Test with mixed codebook types in the same frame.

	ics := &ICStream{
		GlobalGain:      100,
		NumWindowGroups: 1,
		MaxSFB:          4,
	}
	ics.SFBCB[0][0] = 0 // Zero
	ics.SFBCB[0][1] = 1 // Spectral
	ics.SFBCB[0][2] = 0 // Zero
	ics.SFBCB[0][3] = 1 // Spectral

	// Provide data for Huffman decoding
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	r := bits.NewReader(data)

	err := DecodeScaleFactors(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Zero codebook SFBs should be 0
	if ics.ScaleFactors[0][0] != 0 {
		t.Errorf("ScaleFactors[0][0]: got %d, want 0", ics.ScaleFactors[0][0])
	}
	if ics.ScaleFactors[0][2] != 0 {
		t.Errorf("ScaleFactors[0][2]: got %d, want 0", ics.ScaleFactors[0][2])
	}

	// Spectral SFBs should be in valid range
	if ics.ScaleFactors[0][1] < 0 || ics.ScaleFactors[0][1] > 255 {
		t.Errorf("ScaleFactors[0][1] out of range: %d", ics.ScaleFactors[0][1])
	}
	if ics.ScaleFactors[0][3] < 0 || ics.ScaleFactors[0][3] > 255 {
		t.Errorf("ScaleFactors[0][3] out of range: %d", ics.ScaleFactors[0][3])
	}
}

func TestParseScaleFactorData(t *testing.T) {
	// Verify the wrapper function works correctly.

	ics := &ICStream{
		GlobalGain:      100,
		NumWindowGroups: 1,
		MaxSFB:          1,
	}
	ics.SFBCB[0][0] = 0 // Zero codebook

	data := []byte{0x00}
	r := bits.NewReader(data)

	err := ParseScaleFactorData(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.ScaleFactors[0][0] != 0 {
		t.Errorf("ScaleFactors[0][0]: got %d, want 0", ics.ScaleFactors[0][0])
	}
}
