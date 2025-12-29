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
