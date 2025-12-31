package mdct

import (
	"math"
	"testing"
)

func TestNewMDCT_CreatesValidInstance(t *testing.T) {
	tests := []struct {
		n       uint16
		fftSize uint16
	}{
		{256, 64},   // short blocks
		{2048, 512}, // long blocks
	}

	for _, tt := range tests {
		m := NewMDCT(tt.n)
		if m == nil {
			t.Fatalf("NewMDCT(%d) returned nil", tt.n)
		}
		if m.N != tt.n {
			t.Errorf("N = %d, want %d", m.N, tt.n)
		}
		if m.N2 != tt.n/2 {
			t.Errorf("N2 = %d, want %d", m.N2, tt.n/2)
		}
		if m.N4 != tt.n/4 {
			t.Errorf("N4 = %d, want %d", m.N4, tt.n/4)
		}
		if m.N8 != tt.n/8 {
			t.Errorf("N8 = %d, want %d", m.N8, tt.n/8)
		}
		if m.cfft == nil {
			t.Error("cfft is nil")
		}
		if len(m.sincos) != int(tt.fftSize) {
			t.Errorf("sincos length = %d, want %d", len(m.sincos), tt.fftSize)
		}
	}
}

func TestMDCTTables_MatchFAAD2(t *testing.T) {
	// Test table sizes
	t.Run("TableSizes", func(t *testing.T) {
		if len(mdctTab2048) != 512 {
			t.Errorf("mdctTab2048 length = %d, want 512", len(mdctTab2048))
		}
		if len(mdctTab256) != 64 {
			t.Errorf("mdctTab256 length = %d, want 64", len(mdctTab256))
		}
	})

	// Test first entries match the formula: sqrt(2/N) * exp(j * 2*PI * (k + 1/8) / N)
	// For k=0: angle = 2*PI * 0.125 / N
	t.Run("FirstEntry_N2048", func(t *testing.T) {
		n := 2048.0
		scale := math.Sqrt(2.0 / n)
		angle := 2.0 * math.Pi * 0.125 / n
		expectedRe := float32(scale * math.Cos(angle))
		expectedIm := float32(scale * math.Sin(angle))

		// Verify scale is approximately 0.03125
		expectedScale := float32(math.Sqrt(2.0 / 2048.0))
		if math.Abs(float64(expectedScale)-0.03125) > 0.0001 {
			t.Errorf("scale for N=2048: got %v, want ~0.03125", expectedScale)
		}

		gotRe := mdctTab2048[0].Re
		gotIm := mdctTab2048[0].Im

		// Allow small tolerance for floating-point comparison
		const tolerance = 1e-7
		if math.Abs(float64(gotRe-expectedRe)) > tolerance {
			t.Errorf("mdctTab2048[0].Re = %v, want %v", gotRe, expectedRe)
		}
		if math.Abs(float64(gotIm-expectedIm)) > tolerance {
			t.Errorf("mdctTab2048[0].Im = %v, want %v", gotIm, expectedIm)
		}
	})

	t.Run("FirstEntry_N256", func(t *testing.T) {
		n := 256.0
		scale := math.Sqrt(2.0 / n)
		angle := 2.0 * math.Pi * 0.125 / n
		expectedRe := float32(scale * math.Cos(angle))
		expectedIm := float32(scale * math.Sin(angle))

		// Verify scale is approximately 0.0884
		expectedScale := float32(math.Sqrt(2.0 / 256.0))
		if math.Abs(float64(expectedScale)-0.0884) > 0.001 {
			t.Errorf("scale for N=256: got %v, want ~0.0884", expectedScale)
		}

		gotRe := mdctTab256[0].Re
		gotIm := mdctTab256[0].Im

		// Allow small tolerance for floating-point comparison
		const tolerance = 1e-7
		if math.Abs(float64(gotRe-expectedRe)) > tolerance {
			t.Errorf("mdctTab256[0].Re = %v, want %v", gotRe, expectedRe)
		}
		if math.Abs(float64(gotIm-expectedIm)) > tolerance {
			t.Errorf("mdctTab256[0].Im = %v, want %v", gotIm, expectedIm)
		}
	})
}
