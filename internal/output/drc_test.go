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
