package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/huffman"
	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestMSDecode_MSMaskNotPresent(t *testing.T) {
	// When ms_mask_present = 0, M/S decoding is disabled
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		MSMaskPresent:   0, // No M/S
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffsetMax = 1024

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4

	// Input: Left = [1,2,3,4], Right = [5,6,7,8]
	lSpec := []float64{1.0, 2.0, 3.0, 4.0}
	rSpec := []float64{5.0, 6.0, 7.0, 8.0}

	cfg := &MSDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	MSDecode(lSpec, rSpec, cfg)

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

func TestMSDecode_MSMaskAll_LongBlock(t *testing.T) {
	// ms_mask_present = 2 means M/S applies to ALL bands
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		MSMaskPresent:   2, // All bands use M/S
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = 1 // Normal codebook (not intensity/noise)

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SFBCB[0][0] = 1

	// Input: M = [10, 20, 30, 40], S = [2, 4, 6, 8]
	// After M/S: L = M + S = [12, 24, 36, 48]
	//            R = M - S = [8, 16, 24, 32]
	lSpec := []float64{10.0, 20.0, 30.0, 40.0}
	rSpec := []float64{2.0, 4.0, 6.0, 8.0}

	cfg := &MSDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	MSDecode(lSpec, rSpec, cfg)

	expectedL := []float64{12.0, 24.0, 36.0, 48.0}
	expectedR := []float64{8.0, 16.0, 24.0, 32.0}

	for i := range lSpec {
		if lSpec[i] != expectedL[i] {
			t.Errorf("lSpec[%d] = %v, want %v", i, lSpec[i], expectedL[i])
		}
		if rSpec[i] != expectedR[i] {
			t.Errorf("rSpec[%d] = %v, want %v", i, rSpec[i], expectedR[i])
		}
	}
}

func TestMSDecode_MSMaskPerBand(t *testing.T) {
	// ms_mask_present = 1 means per-band M/S mask
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          3,
		NumSWB:          3,
		MSMaskPresent:   1, // Per-band mask
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffset[2] = 8
	icsL.SWBOffset[3] = 12
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = 1
	icsL.SFBCB[0][1] = 1
	icsL.SFBCB[0][2] = 1

	// SFB 0: M/S enabled
	// SFB 1: M/S disabled
	// SFB 2: M/S enabled
	icsL.MSUsed[0][0] = 1
	icsL.MSUsed[0][1] = 0
	icsL.MSUsed[0][2] = 1

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          3,
		NumSWB:          3,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SWBOffset[2] = 8
	icsR.SWBOffset[3] = 12
	icsR.SFBCB[0][0] = 1
	icsR.SFBCB[0][1] = 1
	icsR.SFBCB[0][2] = 1

	// Input: all M=10, S=2
	lSpec := make([]float64, 12)
	rSpec := make([]float64, 12)
	for i := 0; i < 12; i++ {
		lSpec[i] = 10.0
		rSpec[i] = 2.0
	}

	cfg := &MSDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	MSDecode(lSpec, rSpec, cfg)

	// SFB 0 (indices 0-3): M/S applied -> L=12, R=8
	for i := 0; i < 4; i++ {
		if lSpec[i] != 12.0 {
			t.Errorf("lSpec[%d] = %v, want 12.0 (M/S applied)", i, lSpec[i])
		}
		if rSpec[i] != 8.0 {
			t.Errorf("rSpec[%d] = %v, want 8.0 (M/S applied)", i, rSpec[i])
		}
	}

	// SFB 1 (indices 4-7): M/S NOT applied -> unchanged
	for i := 4; i < 8; i++ {
		if lSpec[i] != 10.0 {
			t.Errorf("lSpec[%d] = %v, want 10.0 (unchanged)", i, lSpec[i])
		}
		if rSpec[i] != 2.0 {
			t.Errorf("rSpec[%d] = %v, want 2.0 (unchanged)", i, rSpec[i])
		}
	}

	// SFB 2 (indices 8-11): M/S applied -> L=12, R=8
	for i := 8; i < 12; i++ {
		if lSpec[i] != 12.0 {
			t.Errorf("lSpec[%d] = %v, want 12.0 (M/S applied)", i, lSpec[i])
		}
		if rSpec[i] != 8.0 {
			t.Errorf("rSpec[%d] = %v, want 8.0 (M/S applied)", i, rSpec[i])
		}
	}
}

func TestMSDecode_SkipsIntensityStereo(t *testing.T) {
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          2,
		NumSWB:          2,
		MSMaskPresent:   2, // All bands use M/S
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffset[2] = 8
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = 1 // Normal
	icsL.SFBCB[0][1] = 1 // Normal (left)

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SWBOffset[2] = 8
	icsR.SFBCB[0][0] = 1                           // Normal
	icsR.SFBCB[0][1] = uint8(huffman.IntensityHCB) // Intensity stereo on right

	lSpec := make([]float64, 8)
	rSpec := make([]float64, 8)
	for i := 0; i < 8; i++ {
		lSpec[i] = 10.0
		rSpec[i] = 2.0
	}

	cfg := &MSDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	MSDecode(lSpec, rSpec, cfg)

	// SFB 0: M/S applied (no intensity) -> L=12, R=8
	for i := 0; i < 4; i++ {
		if lSpec[i] != 12.0 {
			t.Errorf("lSpec[%d] = %v, want 12.0", i, lSpec[i])
		}
		if rSpec[i] != 8.0 {
			t.Errorf("rSpec[%d] = %v, want 8.0", i, rSpec[i])
		}
	}

	// SFB 1: M/S skipped (intensity stereo) -> unchanged
	for i := 4; i < 8; i++ {
		if lSpec[i] != 10.0 {
			t.Errorf("lSpec[%d] = %v, want 10.0 (IS band, unchanged)", i, lSpec[i])
		}
		if rSpec[i] != 2.0 {
			t.Errorf("rSpec[%d] = %v, want 2.0 (IS band, unchanged)", i, rSpec[i])
		}
	}
}

func TestMSDecode_SkipsNoiseBands(t *testing.T) {
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          2,
		NumSWB:          2,
		MSMaskPresent:   2, // All bands use M/S
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffset[2] = 8
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = 1                       // Normal
	icsL.SFBCB[0][1] = uint8(huffman.NoiseHCB) // Noise band

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SWBOffset[2] = 8
	icsR.SFBCB[0][0] = 1 // Normal
	icsR.SFBCB[0][1] = 1 // Normal

	lSpec := make([]float64, 8)
	rSpec := make([]float64, 8)
	for i := 0; i < 8; i++ {
		lSpec[i] = 10.0
		rSpec[i] = 2.0
	}

	cfg := &MSDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	MSDecode(lSpec, rSpec, cfg)

	// SFB 0: M/S applied (no noise) -> L=12, R=8
	for i := 0; i < 4; i++ {
		if lSpec[i] != 12.0 {
			t.Errorf("lSpec[%d] = %v, want 12.0", i, lSpec[i])
		}
		if rSpec[i] != 8.0 {
			t.Errorf("rSpec[%d] = %v, want 8.0", i, rSpec[i])
		}
	}

	// SFB 1: M/S skipped (noise band) -> unchanged
	for i := 4; i < 8; i++ {
		if lSpec[i] != 10.0 {
			t.Errorf("lSpec[%d] = %v, want 10.0 (noise band, unchanged)", i, lSpec[i])
		}
		if rSpec[i] != 2.0 {
			t.Errorf("rSpec[%d] = %v, want 2.0 (noise band, unchanged)", i, rSpec[i])
		}
	}
}

func TestMSDecode_ShortBlocks(t *testing.T) {
	// Test 8 short windows grouped into 2 groups of 4
	icsL := &syntax.ICStream{
		NumWindowGroups: 2,
		NumWindows:      8,
		MaxSFB:          1,
		NumSWB:          1,
		MSMaskPresent:   2,
		WindowSequence:  syntax.EightShortSequence,
	}
	icsL.WindowGroupLength[0] = 4 // First group: 4 windows
	icsL.WindowGroupLength[1] = 4 // Second group: 4 windows
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4 // 4 coefficients per SFB per window
	icsL.SWBOffsetMax = 128
	icsL.SFBCB[0][0] = 1
	icsL.SFBCB[1][0] = 1

	icsR := &syntax.ICStream{
		NumWindowGroups: 2,
		NumWindows:      8,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.EightShortSequence,
	}
	icsR.WindowGroupLength[0] = 4
	icsR.WindowGroupLength[1] = 4
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SFBCB[0][0] = 1
	icsR.SFBCB[1][0] = 1

	// FrameLength=1024, nshort=128
	// Full spectrum: 8 windows * 128 coefficients = 1024 total
	// SFB covers only first 4 coefficients of each window
	lSpec := make([]float64, 1024)
	rSpec := make([]float64, 1024)
	for i := 0; i < 1024; i++ {
		lSpec[i] = 10.0
		rSpec[i] = 2.0
	}

	cfg := &MSDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	MSDecode(lSpec, rSpec, cfg)

	// Check coefficients for each of the 8 windows
	// Each window starts at offset group*128, and we process indices 0-3
	for win := 0; win < 8; win++ {
		base := win * 128
		// First 4 coefficients of each window should be transformed: L=12, R=8
		for i := 0; i < 4; i++ {
			idx := base + i
			if lSpec[idx] != 12.0 {
				t.Errorf("lSpec[%d] (win=%d, i=%d) = %v, want 12.0", idx, win, i, lSpec[idx])
			}
			if rSpec[idx] != 8.0 {
				t.Errorf("rSpec[%d] (win=%d, i=%d) = %v, want 8.0", idx, win, i, rSpec[idx])
			}
		}
		// Remaining coefficients (4-127) should be unchanged
		for i := 4; i < 128; i++ {
			idx := base + i
			if lSpec[idx] != 10.0 {
				t.Errorf("lSpec[%d] (win=%d, i=%d) = %v, want 10.0 (unchanged)", idx, win, i, lSpec[idx])
			}
			if rSpec[idx] != 2.0 {
				t.Errorf("rSpec[%d] (win=%d, i=%d) = %v, want 2.0 (unchanged)", idx, win, i, rSpec[idx])
			}
		}
	}
}
