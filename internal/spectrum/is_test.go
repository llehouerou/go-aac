package spectrum

import (
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
