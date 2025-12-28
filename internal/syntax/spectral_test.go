// internal/syntax/spectral_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
	"github.com/llehouerou/go-aac/internal/huffman"
)

func TestParseSpectralData_ZeroCodebook(t *testing.T) {
	// All SFBs use zero codebook - no spectral data needed
	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		NumWindowGroups: 1,
		MaxSFB:          4,
	}
	ics.NumSec[0] = 1
	ics.SectCB[0][0] = 0 // Zero codebook
	ics.SectStart[0][0] = 0
	ics.SectEnd[0][0] = 4
	// Set up SFB offsets
	ics.SectSFBOffset[0][0] = 0
	ics.SectSFBOffset[0][1] = 32
	ics.SectSFBOffset[0][2] = 64
	ics.SectSFBOffset[0][3] = 96
	ics.SectSFBOffset[0][4] = 128
	ics.WindowGroupLength[0] = 1

	data := []byte{0x00}
	r := bits.NewReader(data)

	specData := make([]int16, 1024)
	err := ParseSpectralData(r, ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All values should remain 0
	for i := 0; i < 128; i++ {
		if specData[i] != 0 {
			t.Errorf("specData[%d]: got %d, want 0", i, specData[i])
		}
	}
}

func TestParseSpectralData_NoiseCodebook(t *testing.T) {
	// Noise codebook (13) should skip spectral data (PNS)
	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		NumWindowGroups: 1,
		MaxSFB:          2,
	}
	ics.NumSec[0] = 1
	ics.SectCB[0][0] = uint8(huffman.NoiseHCB) // Noise codebook
	ics.SectStart[0][0] = 0
	ics.SectEnd[0][0] = 2
	ics.SectSFBOffset[0][0] = 0
	ics.SectSFBOffset[0][1] = 32
	ics.SectSFBOffset[0][2] = 64
	ics.WindowGroupLength[0] = 1

	data := []byte{0x00}
	r := bits.NewReader(data)

	specData := make([]int16, 1024)
	err := ParseSpectralData(r, ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All values should remain 0 (no spectral data for noise)
	for i := 0; i < 64; i++ {
		if specData[i] != 0 {
			t.Errorf("specData[%d]: got %d, want 0", i, specData[i])
		}
	}
}

func TestParseSpectralData_IntensityCodebook(t *testing.T) {
	// Intensity codebooks (14, 15) should skip spectral data
	testCases := []struct {
		name   string
		sectCB uint8
	}{
		{"IntensityHCB2", uint8(huffman.IntensityHCB2)},
		{"IntensityHCB", uint8(huffman.IntensityHCB)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ics := &ICStream{
				WindowSequence:  OnlyLongSequence,
				NumWindowGroups: 1,
				MaxSFB:          2,
			}
			ics.NumSec[0] = 1
			ics.SectCB[0][0] = tc.sectCB
			ics.SectStart[0][0] = 0
			ics.SectEnd[0][0] = 2
			ics.SectSFBOffset[0][0] = 0
			ics.SectSFBOffset[0][1] = 32
			ics.SectSFBOffset[0][2] = 64
			ics.WindowGroupLength[0] = 1

			data := []byte{0x00}
			r := bits.NewReader(data)

			specData := make([]int16, 1024)
			err := ParseSpectralData(r, ics, specData, 1024)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// All values should remain 0
			for i := 0; i < 64; i++ {
				if specData[i] != 0 {
					t.Errorf("specData[%d]: got %d, want 0", i, specData[i])
				}
			}
		})
	}
}

func TestParseSpectralData_MultipleSections(t *testing.T) {
	// Test with multiple sections, some zero and some spectral
	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		NumWindowGroups: 1,
		MaxSFB:          4,
	}
	ics.NumSec[0] = 2
	// First section: zero codebook
	ics.SectCB[0][0] = 0
	ics.SectStart[0][0] = 0
	ics.SectEnd[0][0] = 2
	// Second section: zero codebook
	ics.SectCB[0][1] = 0
	ics.SectStart[0][1] = 2
	ics.SectEnd[0][1] = 4
	// SFB offsets
	ics.SectSFBOffset[0][0] = 0
	ics.SectSFBOffset[0][1] = 32
	ics.SectSFBOffset[0][2] = 64
	ics.SectSFBOffset[0][3] = 96
	ics.SectSFBOffset[0][4] = 128
	ics.WindowGroupLength[0] = 1

	data := []byte{0x00}
	r := bits.NewReader(data)

	specData := make([]int16, 1024)
	err := ParseSpectralData(r, ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All values should remain 0
	for i := 0; i < 128; i++ {
		if specData[i] != 0 {
			t.Errorf("specData[%d]: got %d, want 0", i, specData[i])
		}
	}
}

func TestParseSpectralData_MultipleWindowGroups(t *testing.T) {
	// Test with multiple window groups (short windows)
	ics := &ICStream{
		WindowSequence:  EightShortSequence,
		NumWindowGroups: 2,
		MaxSFB:          2,
	}
	ics.NumSec[0] = 1
	ics.NumSec[1] = 1
	// First group: zero codebook
	ics.SectCB[0][0] = 0
	ics.SectStart[0][0] = 0
	ics.SectEnd[0][0] = 2
	// Second group: zero codebook
	ics.SectCB[1][0] = 0
	ics.SectStart[1][0] = 0
	ics.SectEnd[1][0] = 2
	// SFB offsets for group 0
	ics.SectSFBOffset[0][0] = 0
	ics.SectSFBOffset[0][1] = 8
	ics.SectSFBOffset[0][2] = 16
	// SFB offsets for group 1
	ics.SectSFBOffset[1][0] = 0
	ics.SectSFBOffset[1][1] = 8
	ics.SectSFBOffset[1][2] = 16
	// Window group lengths
	ics.WindowGroupLength[0] = 4
	ics.WindowGroupLength[1] = 4

	data := []byte{0x00}
	r := bits.NewReader(data)

	specData := make([]int16, 1024)
	err := ParseSpectralData(r, ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All values should remain 0
	for i := 0; i < 1024; i++ {
		if specData[i] != 0 {
			t.Errorf("specData[%d]: got %d, want 0", i, specData[i])
		}
	}
}

func TestParseSpectralData_EmptySections(t *testing.T) {
	// Test with no sections (MaxSFB = 0)
	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		NumWindowGroups: 1,
		MaxSFB:          0,
	}
	ics.NumSec[0] = 0
	ics.WindowGroupLength[0] = 1

	data := []byte{0x00}
	r := bits.NewReader(data)

	specData := make([]int16, 1024)
	err := ParseSpectralData(r, ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All values should remain 0
	for i := 0; i < 1024; i++ {
		if specData[i] != 0 {
			t.Errorf("specData[%d]: got %d, want 0", i, specData[i])
		}
	}
}
