package spectrum

import (
	"math"
	"testing"

	"github.com/llehouerou/go-aac/internal/huffman"
	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestISDecode_NoIntensityBands(t *testing.T) {
	// When no intensity stereo bands exist, spectra should be unchanged
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = 1 // Normal codebook, not intensity

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SFBCB[0][0] = 1 // Normal codebook, not intensity

	lSpec := []float64{1.0, 2.0, 3.0, 4.0}
	rSpec := []float64{5.0, 6.0, 7.0, 8.0}

	cfg := &ISDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	ISDecode(lSpec, rSpec, cfg)

	// Should be unchanged
	expectedL := []float64{1.0, 2.0, 3.0, 4.0}
	expectedR := []float64{5.0, 6.0, 7.0, 8.0}

	for i := range lSpec {
		if lSpec[i] != expectedL[i] {
			t.Errorf("lSpec[%d] = %v, want %v", i, lSpec[i], expectedL[i])
		}
		if rSpec[i] != expectedR[i] {
			t.Errorf("rSpec[%d] = %v, want %v", i, rSpec[i], expectedR[i])
		}
	}
}

func TestISDecode_BasicIntensityStereo(t *testing.T) {
	// Basic intensity stereo: scale factor = 0 means scale = 1.0 (0.5^0)
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = 1 // Normal codebook in left

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SWBOffsetMax = 1024
	icsR.SFBCB[0][0] = uint8(huffman.IntensityHCB) // Intensity stereo
	icsR.ScaleFactors[0][0] = 0                    // scale = 0.5^(0/4) = 1.0

	// Left channel: [10, 20, 30, 40]
	// Right should become: [10, 20, 30, 40] (scale=1.0, same sign)
	lSpec := []float64{10.0, 20.0, 30.0, 40.0}
	rSpec := make([]float64, 4)

	cfg := &ISDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	ISDecode(lSpec, rSpec, cfg)

	expectedR := []float64{10.0, 20.0, 30.0, 40.0}
	for i := range rSpec {
		if rSpec[i] != expectedR[i] {
			t.Errorf("rSpec[%d] = %v, want %v", i, rSpec[i], expectedR[i])
		}
	}

	// Left should be unchanged
	expectedL := []float64{10.0, 20.0, 30.0, 40.0}
	for i := range lSpec {
		if lSpec[i] != expectedL[i] {
			t.Errorf("lSpec[%d] modified: got %v, want %v", i, lSpec[i], expectedL[i])
		}
	}
}

func TestISDecode_ScaleFactorScaling(t *testing.T) {
	// Test that scale factor correctly scales the output
	// scale = 0.5^(sf/4)
	// sf=4 -> scale = 0.5^1 = 0.5
	// sf=-4 -> scale = 0.5^(-1) = 2.0
	tests := []struct {
		name        string
		scaleFactor int16
		expected    float64
	}{
		{"sf=0 -> scale=1.0", 0, 1.0},
		{"sf=4 -> scale=0.5", 4, 0.5},
		{"sf=-4 -> scale=2.0", -4, 2.0},
		{"sf=8 -> scale=0.25", 8, 0.25},
		{"sf=2 -> scale~0.707", 2, 0.7071067811865476}, // 0.5^0.5
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			icsL := &syntax.ICStream{
				NumWindowGroups: 1,
				MaxSFB:          1,
				NumSWB:          1,
				WindowSequence:  syntax.OnlyLongSequence,
			}
			icsL.WindowGroupLength[0] = 1
			icsL.SWBOffset[0] = 0
			icsL.SWBOffset[1] = 1
			icsL.SWBOffsetMax = 1024
			icsL.SFBCB[0][0] = 1

			icsR := &syntax.ICStream{
				NumWindowGroups: 1,
				MaxSFB:          1,
				NumSWB:          1,
				WindowSequence:  syntax.OnlyLongSequence,
			}
			icsR.WindowGroupLength[0] = 1
			icsR.SWBOffset[0] = 0
			icsR.SWBOffset[1] = 1
			icsR.SWBOffsetMax = 1024
			icsR.SFBCB[0][0] = uint8(huffman.IntensityHCB)
			icsR.ScaleFactors[0][0] = tc.scaleFactor

			lSpec := []float64{1.0} // Use 1.0 so result equals scale
			rSpec := make([]float64, 1)

			cfg := &ISDecodeConfig{
				ICSL:        icsL,
				ICSR:        icsR,
				FrameLength: 1024,
			}

			ISDecode(lSpec, rSpec, cfg)

			const epsilon = 1e-10
			if diff := math.Abs(rSpec[0] - tc.expected); diff > epsilon {
				t.Errorf("rSpec[0] = %v, want %v (diff=%v)", rSpec[0], tc.expected, diff)
			}
		})
	}
}

func TestISDecode_IntensityHCB2_InvertsSign(t *testing.T) {
	// INTENSITY_HCB2 with ms_mask_present=0 should invert the sign
	// is_intensity() returns -1, invert_intensity() returns 1
	// Since -1 != 1, sign is inverted
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		MSMaskPresent:   0, // No M/S -> invert_intensity returns 1
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = 1

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SWBOffsetMax = 1024
	icsR.SFBCB[0][0] = uint8(huffman.IntensityHCB2) // Out-of-phase: is_intensity=-1
	icsR.ScaleFactors[0][0] = 0                     // scale = 1.0

	lSpec := []float64{10.0, -20.0, 30.0, -40.0}
	rSpec := make([]float64, 4)

	cfg := &ISDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	ISDecode(lSpec, rSpec, cfg)

	// is_intensity=-1, invert_intensity=1, so signs are inverted
	expectedR := []float64{-10.0, 20.0, -30.0, 40.0}
	for i := range rSpec {
		if rSpec[i] != expectedR[i] {
			t.Errorf("rSpec[%d] = %v, want %v", i, rSpec[i], expectedR[i])
		}
	}
}

func TestISDecode_MSMaskInteraction(t *testing.T) {
	// When ms_mask_present=1 and ms_used=1, invert_intensity returns -1
	// Combined with INTENSITY_HCB (is_intensity=1), sign is inverted
	// Combined with INTENSITY_HCB2 (is_intensity=-1), sign is NOT inverted
	tests := []struct {
		name       string
		codebook   uint8
		msUsed     uint8
		expectSign float64 // Expected sign: 1.0 or -1.0
	}{
		// INTENSITY_HCB (is=1) with ms_used=0 (inv=1): 1 != 1? No -> no invert
		{"HCB, ms_used=0", uint8(huffman.IntensityHCB), 0, 1.0},
		// INTENSITY_HCB (is=1) with ms_used=1 (inv=-1): 1 != -1? Yes -> invert
		{"HCB, ms_used=1", uint8(huffman.IntensityHCB), 1, -1.0},
		// INTENSITY_HCB2 (is=-1) with ms_used=0 (inv=1): -1 != 1? Yes -> invert
		{"HCB2, ms_used=0", uint8(huffman.IntensityHCB2), 0, -1.0},
		// INTENSITY_HCB2 (is=-1) with ms_used=1 (inv=-1): -1 != -1? No -> no invert
		{"HCB2, ms_used=1", uint8(huffman.IntensityHCB2), 1, 1.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			icsL := &syntax.ICStream{
				NumWindowGroups: 1,
				MaxSFB:          1,
				NumSWB:          1,
				MSMaskPresent:   1, // Per-band M/S
				WindowSequence:  syntax.OnlyLongSequence,
			}
			icsL.WindowGroupLength[0] = 1
			icsL.SWBOffset[0] = 0
			icsL.SWBOffset[1] = 1
			icsL.SWBOffsetMax = 1024
			icsL.SFBCB[0][0] = 1
			icsL.MSUsed[0][0] = tc.msUsed

			icsR := &syntax.ICStream{
				NumWindowGroups: 1,
				MaxSFB:          1,
				NumSWB:          1,
				WindowSequence:  syntax.OnlyLongSequence,
			}
			icsR.WindowGroupLength[0] = 1
			icsR.SWBOffset[0] = 0
			icsR.SWBOffset[1] = 1
			icsR.SWBOffsetMax = 1024
			icsR.SFBCB[0][0] = tc.codebook
			icsR.ScaleFactors[0][0] = 0 // scale = 1.0

			lSpec := []float64{10.0}
			rSpec := make([]float64, 1)

			cfg := &ISDecodeConfig{
				ICSL:        icsL,
				ICSR:        icsR,
				FrameLength: 1024,
			}

			ISDecode(lSpec, rSpec, cfg)

			expected := 10.0 * tc.expectSign
			if rSpec[0] != expected {
				t.Errorf("rSpec[0] = %v, want %v", rSpec[0], expected)
			}
		})
	}
}
