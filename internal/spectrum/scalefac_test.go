package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/huffman"
	"github.com/llehouerou/go-aac/internal/syntax"
	"github.com/llehouerou/go-aac/internal/tables"
)

func TestApplyScaleFactors_LongBlock_SingleSFB(t *testing.T) {
	// Setup: single window group, single SFB covering 4 coefficients
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4

	// Set codebook to a spectral codebook (not noise/intensity)
	ics.SFBCB[0][0] = uint8(huffman.EscHCB)

	// Scale factor = 100 means multiplier = 2^((100-100)/4) = 2^0 = 1.0
	ics.ScaleFactors[0][0] = 100

	// Input: 4 coefficients with value 1.0
	specData := []float64{1.0, 1.0, 1.0, 1.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// With sf=100, multiplier=1.0, output should equal input
	for i, v := range specData {
		if v != 1.0 {
			t.Errorf("specData[%d] = %v, want 1.0", i, v)
		}
	}
}

func TestApplyScaleFactors_LongBlock_ScaleFactor104(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.EscHCB)

	// Scale factor = 104 means multiplier = 2^((104-100)/4) = 2^1 = 2.0
	ics.ScaleFactors[0][0] = 104

	specData := []float64{1.0, 2.0, 3.0, 4.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	expected := []float64{2.0, 4.0, 6.0, 8.0}
	for i, v := range specData {
		if v != expected[i] {
			t.Errorf("specData[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

func TestApplyScaleFactors_LongBlock_ScaleFactor101(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.EscHCB)

	// Scale factor = 101 means multiplier = 2^((101-100)/4) = 2^0.25 ~ 1.189
	ics.ScaleFactors[0][0] = 101

	specData := []float64{1.0, 1.0, 1.0, 1.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// Pow2FracTable[1] = 2^0.25
	expected := tables.Pow2FracTable[1]
	for i, v := range specData {
		if v != expected {
			t.Errorf("specData[%d] = %v, want %v", i, v, expected)
		}
	}
}

func TestApplyScaleFactors_NoiseCodebook_ZerosOutput(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	ics.ScaleFactors[0][0] = 120 // Should be ignored

	specData := []float64{1.0, 2.0, 3.0, 4.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// Noise bands should be zeroed (filled later by PNS)
	for i, v := range specData {
		if v != 0.0 {
			t.Errorf("specData[%d] = %v, want 0.0 (noise band)", i, v)
		}
	}
}

func TestApplyScaleFactors_IntensityCodebook_ZerosOutput(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.IntensityHCB)
	ics.ScaleFactors[0][0] = 120 // Should be ignored

	specData := []float64{1.0, 2.0, 3.0, 4.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// Intensity bands should be zeroed (filled later by IS)
	for i, v := range specData {
		if v != 0.0 {
			t.Errorf("specData[%d] = %v, want 0.0 (intensity band)", i, v)
		}
	}
}

func TestApplyScaleFactors_IntensityHCB2_ZerosOutput(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.IntensityHCB2)
	ics.ScaleFactors[0][0] = 120 // Should be ignored

	specData := []float64{1.0, 2.0, 3.0, 4.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// Intensity bands should be zeroed (filled later by IS)
	for i, v := range specData {
		if v != 0.0 {
			t.Errorf("specData[%d] = %v, want 0.0 (intensity band)", i, v)
		}
	}
}

func TestApplyScaleFactors_MultipleSFBs(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          3,
		NumSWB:          3,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	// SFB 0: indices 0-3 (width 4)
	// SFB 1: indices 4-7 (width 4)
	// SFB 2: indices 8-11 (width 4)
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffset[3] = 12

	ics.SFBCB[0][0] = uint8(huffman.EscHCB) // normal band
	ics.SFBCB[0][1] = uint8(huffman.EscHCB) // normal band
	ics.SFBCB[0][2] = uint8(huffman.EscHCB) // normal band

	// sf=100 -> mult=1.0
	// sf=104 -> mult=2.0
	// sf=108 -> mult=4.0
	ics.ScaleFactors[0][0] = 100
	ics.ScaleFactors[0][1] = 104
	ics.ScaleFactors[0][2] = 108

	specData := make([]float64, 12)
	for i := range specData {
		specData[i] = 1.0
	}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// SFB 0: 1.0 * 1.0 = 1.0
	for i := 0; i < 4; i++ {
		if specData[i] != 1.0 {
			t.Errorf("specData[%d] = %v, want 1.0", i, specData[i])
		}
	}
	// SFB 1: 1.0 * 2.0 = 2.0
	for i := 4; i < 8; i++ {
		if specData[i] != 2.0 {
			t.Errorf("specData[%d] = %v, want 2.0", i, specData[i])
		}
	}
	// SFB 2: 1.0 * 4.0 = 4.0
	for i := 8; i < 12; i++ {
		if specData[i] != 4.0 {
			t.Errorf("specData[%d] = %v, want 4.0", i, specData[i])
		}
	}
}

func TestApplyScaleFactors_ZeroCodebook_PreservesData(t *testing.T) {
	// Zero codebook means no spectral data - coefficients should remain zero
	// (In practice, they start at zero and stay zero)
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.ZeroHCB)
	ics.ScaleFactors[0][0] = 100

	specData := []float64{0.0, 0.0, 0.0, 0.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// Zero codebook should still apply scale factor (0 * anything = 0)
	for i, v := range specData {
		if v != 0.0 {
			t.Errorf("specData[%d] = %v, want 0.0", i, v)
		}
	}
}

func TestApplyScaleFactors_NegativeScaleFactor(t *testing.T) {
	// Test scale factors below 100 (negative exponent)
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.EscHCB)

	// Scale factor = 96 means multiplier = 2^((96-100)/4) = 2^-1 = 0.5
	ics.ScaleFactors[0][0] = 96

	specData := []float64{2.0, 4.0, 6.0, 8.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	expected := []float64{1.0, 2.0, 3.0, 4.0}
	for i, v := range specData {
		if v != expected[i] {
			t.Errorf("specData[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

func TestApplyScaleFactors_ShortBlock(t *testing.T) {
	// Setup: 8 short windows in 1 group
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      8,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.EightShortSequence,
	}
	ics.WindowGroupLength[0] = 8

	// Short block: each window is 128 samples, SFB covers first 4 of each
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4

	ics.SFBCB[0][0] = uint8(huffman.EscHCB)
	ics.ScaleFactors[0][0] = 104 // multiplier = 2.0

	// 8 windows * 4 coefficients = 32 values
	// Interleaved: win0[0-3], win1[0-3], ..., win7[0-3] = 32 total
	// With winInc = 4 (SWBOffset[NumSWB]), data layout is sequential
	specData := make([]float64, 32)
	for i := range specData {
		specData[i] = 1.0
	}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// All values should be multiplied by 2.0
	for i, v := range specData {
		if v != 2.0 {
			t.Errorf("specData[%d] = %v, want 2.0", i, v)
		}
	}
}

func TestApplyScaleFactors_MultipleSFB(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8

	ics.SFBCB[0][0] = uint8(huffman.EscHCB)
	ics.SFBCB[0][1] = uint8(huffman.EscHCB)

	// SFB 0: sf=104 -> mult=2.0
	// SFB 1: sf=108 -> mult=4.0
	ics.ScaleFactors[0][0] = 104
	ics.ScaleFactors[0][1] = 108

	specData := []float64{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	expected := []float64{2.0, 2.0, 2.0, 2.0, 4.0, 4.0, 4.0, 4.0}
	for i, v := range specData {
		if v != expected[i] {
			t.Errorf("specData[%d] = %v, want %v", i, v, expected[i])
		}
	}
}
