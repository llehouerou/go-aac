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

	// Validate all entries match formula: sqrt(2/N) * exp(j * 2*PI * (k + 1/8) / N)
	// Tolerance of 1e-7 accounts for float32 precision (~7 decimal digits)
	const tolerance = 1e-7

	t.Run("AllEntries_N2048", func(t *testing.T) {
		n := 2048.0
		scale := math.Sqrt(2.0 / n)
		for k := 0; k < len(mdctTab2048); k++ {
			angle := 2.0 * math.Pi * (float64(k) + 0.125) / n
			expectedRe := float32(scale * math.Cos(angle))
			expectedIm := float32(scale * math.Sin(angle))

			if math.Abs(float64(mdctTab2048[k].Re-expectedRe)) > tolerance {
				t.Errorf("mdctTab2048[%d].Re = %v, want %v", k, mdctTab2048[k].Re, expectedRe)
			}
			if math.Abs(float64(mdctTab2048[k].Im-expectedIm)) > tolerance {
				t.Errorf("mdctTab2048[%d].Im = %v, want %v", k, mdctTab2048[k].Im, expectedIm)
			}
		}
	})

	t.Run("AllEntries_N256", func(t *testing.T) {
		n := 256.0
		scale := math.Sqrt(2.0 / n)
		for k := 0; k < len(mdctTab256); k++ {
			angle := 2.0 * math.Pi * (float64(k) + 0.125) / n
			expectedRe := float32(scale * math.Cos(angle))
			expectedIm := float32(scale * math.Sin(angle))

			if math.Abs(float64(mdctTab256[k].Re-expectedRe)) > tolerance {
				t.Errorf("mdctTab256[%d].Re = %v, want %v", k, mdctTab256[k].Re, expectedRe)
			}
			if math.Abs(float64(mdctTab256[k].Im-expectedIm)) > tolerance {
				t.Errorf("mdctTab256[%d].Im = %v, want %v", k, mdctTab256[k].Im, expectedIm)
			}
		}
	})
}

func TestIMDCT_Linearity(t *testing.T) {
	// Test that IMDCT(2*x) = 2*IMDCT(x) (linearity)
	m := NewMDCT(256)

	input1 := make([]float32, 128) // N/2 input
	input2 := make([]float32, 128)
	for i := range input1 {
		input1[i] = float32(i) * 0.01
		input2[i] = float32(i) * 0.02 // 2x
	}

	output1 := make([]float32, 256) // N output
	output2 := make([]float32, 256)

	m.IMDCT(input1, output1)
	m.IMDCT(input2, output2)

	for i := range output1 {
		expected := output1[i] * 2
		if math.Abs(float64(output2[i]-expected)) > 1e-4 {
			t.Errorf("output2[%d] = %v, want %v (2*output1)", i, output2[i], expected)
		}
	}
}

func TestIMDCT_DCInput(t *testing.T) {
	// DC input should produce a known pattern
	m := NewMDCT(256)

	input := make([]float32, 128)
	input[0] = 1.0 // DC component only

	output := make([]float32, 256)
	m.IMDCT(input, output)

	// Output should not be all zeros
	hasNonZero := false
	for _, v := range output {
		if v != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("IMDCT of DC produced all zeros")
	}
}
