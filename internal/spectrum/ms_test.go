package spectrum

import (
	"testing"

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
