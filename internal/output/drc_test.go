// internal/output/drc_test.go
package output

import (
	"math"
	"testing"

	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestDRCRefLevel(t *testing.T) {
	// DRC_REF_LEVEL = 20 * 4 = 80 (represents -20 dB)
	// Source: ~/dev/faad2/libfaad/drc.h:38
	if DRCRefLevel != 80 {
		t.Errorf("DRCRefLevel: got %d, want 80", DRCRefLevel)
	}
}

func TestNewDRC(t *testing.T) {
	drc := NewDRC(0.5, 0.75)

	if drc.Cut != 0.5 {
		t.Errorf("Cut: got %v, want 0.5", drc.Cut)
	}
	if drc.Boost != 0.75 {
		t.Errorf("Boost: got %v, want 0.75", drc.Boost)
	}
}

func TestDRCDecode_SingleBand_NoGain(t *testing.T) {
	// DRC with default values should apply no gain
	drc := NewDRC(0.0, 0.0)

	info := &syntax.DRCInfo{
		NumBands:     1,
		ProgRefLevel: DRCRefLevel, // Same as reference = no adjustment
		DynRngSgn:    [17]uint8{0},
		DynRngCtl:    [17]uint8{0},
	}
	info.BandTop[0] = 1024/4 - 1 // Default: entire frame

	// Input: some spectral coefficients
	spec := []float32{1.0, 2.0, 3.0, 4.0}

	drc.Decode(info, spec)

	// With zero control values and prog_ref_level == DRC_REF_LEVEL,
	// the exponent is 0, so factor = 2^0 = 1.0 (no change)
	expected := []float32{1.0, 2.0, 3.0, 4.0}
	for i, v := range spec {
		if math.Abs(float64(v-expected[i])) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want %v", i, v, expected[i])
		}
	}
}

func TestDRCDecode_Compression(t *testing.T) {
	// Full cut (1.0) with max control value
	drc := NewDRC(1.0, 0.0)

	info := &syntax.DRCInfo{
		NumBands:     1,
		ProgRefLevel: DRCRefLevel,   // No level adjustment
		DynRngSgn:    [17]uint8{1},  // Compress
		DynRngCtl:    [17]uint8{24}, // 24 quarter-dB
	}
	info.BandTop[0] = 1024/4 - 1

	// Input
	spec := []float32{1.0, 1.0, 1.0, 1.0}

	drc.Decode(info, spec)

	// exp = (-1.0 * 24 - 0) / 24 = -1.0
	// factor = 2^(-1) = 0.5
	expected := float32(0.5)
	for i, v := range spec {
		if math.Abs(float64(v-expected)) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want %v", i, v, expected)
		}
	}
}

func TestDRCDecode_Boost(t *testing.T) {
	// Full boost (1.0) with max control value
	drc := NewDRC(0.0, 1.0)

	info := &syntax.DRCInfo{
		NumBands:     1,
		ProgRefLevel: DRCRefLevel,   // No level adjustment
		DynRngSgn:    [17]uint8{0},  // Boost
		DynRngCtl:    [17]uint8{24}, // 24 quarter-dB
	}
	info.BandTop[0] = 1024/4 - 1

	// Input
	spec := []float32{1.0, 1.0, 1.0, 1.0}

	drc.Decode(info, spec)

	// exp = (1.0 * 24 - 0) / 24 = 1.0
	// factor = 2^1 = 2.0
	expected := float32(2.0)
	for i, v := range spec {
		if math.Abs(float64(v-expected)) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want %v", i, v, expected)
		}
	}
}

func TestDRCDecode_ProgRefLevelAdjustment(t *testing.T) {
	// Test with prog_ref_level different from DRC_REF_LEVEL
	drc := NewDRC(1.0, 1.0)

	info := &syntax.DRCInfo{
		NumBands:     1,
		ProgRefLevel: 56,           // -14 dB (56/4 = 14), differs from -20 dB
		DynRngSgn:    [17]uint8{0}, // Boost
		DynRngCtl:    [17]uint8{0}, // No control signal
	}
	info.BandTop[0] = 1024/4 - 1

	// Input
	spec := []float32{1.0, 1.0, 1.0, 1.0}

	drc.Decode(info, spec)

	// exp = (1.0 * 0 - (80 - 56)) / 24 = -24/24 = -1.0
	// factor = 2^(-1) = 0.5
	expected := float32(0.5)
	for i, v := range spec {
		if math.Abs(float64(v-expected)) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want %v", i, v, expected)
		}
	}
}

func TestDRCDecode_MultipleBands(t *testing.T) {
	drc := NewDRC(1.0, 1.0)

	// Two bands with different settings
	info := &syntax.DRCInfo{
		NumBands:     2,
		ProgRefLevel: DRCRefLevel,
		DynRngSgn:    [17]uint8{1, 0},   // Band 0: compress, Band 1: boost
		DynRngCtl:    [17]uint8{24, 24}, // Same control value
	}
	// Band 0: samples 0-3 (top=0 means 4 samples: 4*(0+1))
	// Band 1: samples 4-7 (top=1 means 8 samples: 4*(1+1))
	info.BandTop[0] = 0 // Covers samples 0-3
	info.BandTop[1] = 1 // Covers samples 4-7

	// Input: 8 samples
	spec := []float32{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0}

	drc.Decode(info, spec)

	// Band 0: compress, factor = 0.5
	for i := 0; i < 4; i++ {
		if math.Abs(float64(spec[i]-0.5)) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want 0.5", i, spec[i])
		}
	}

	// Band 1: boost, factor = 2.0
	for i := 4; i < 8; i++ {
		if math.Abs(float64(spec[i]-2.0)) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want 2.0", i, spec[i])
		}
	}
}

func TestDRCDecode_NilInfo(t *testing.T) {
	drc := NewDRC(1.0, 1.0)

	// Input should not be modified
	spec := []float32{1.0, 2.0, 3.0, 4.0}
	original := make([]float32, len(spec))
	copy(original, spec)

	// Should not panic
	drc.Decode(nil, spec)

	// Should be unchanged
	for i, v := range spec {
		if v != original[i] {
			t.Errorf("spec[%d]: got %v, want %v", i, v, original[i])
		}
	}
}

func TestDRCDecode_ZeroBands(t *testing.T) {
	drc := NewDRC(1.0, 1.0)

	info := &syntax.DRCInfo{
		NumBands: 0, // No bands
	}

	// Input should not be modified
	spec := []float32{1.0, 2.0, 3.0, 4.0}
	original := make([]float32, len(spec))
	copy(original, spec)

	drc.Decode(info, spec)

	// Should be unchanged
	for i, v := range spec {
		if v != original[i] {
			t.Errorf("spec[%d]: got %v, want %v", i, v, original[i])
		}
	}
}

func TestDRCDecode_ShortSpec(t *testing.T) {
	// Test that DRC correctly handles spec arrays shorter than band_top suggests
	drc := NewDRC(1.0, 1.0)

	info := &syntax.DRCInfo{
		NumBands:     1,
		ProgRefLevel: DRCRefLevel,
		DynRngSgn:    [17]uint8{0}, // Boost
		DynRngCtl:    [17]uint8{24},
	}
	info.BandTop[0] = 255 // Would suggest 1024 samples

	// But we only have 4 samples
	spec := []float32{1.0, 1.0, 1.0, 1.0}

	// Should not panic, should only process available samples
	drc.Decode(info, spec)

	// All 4 samples should be boosted
	expected := float32(2.0)
	for i, v := range spec {
		if math.Abs(float64(v-expected)) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want %v", i, v, expected)
		}
	}
}
