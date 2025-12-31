package mdct

import (
	"fmt"
	"math"
	"testing"
)

// TestIMDCT_BasicValidation performs basic sanity checks on IMDCT output.
// Validates that outputs are finite (not NaN or Inf) for known inputs.
func TestIMDCT_BasicValidation(t *testing.T) {
	// Test case: N=256 with impulse at DC
	t.Run("n256_dc_impulse", func(t *testing.T) {
		m := NewMDCT(256)

		input := make([]float32, 128)
		input[0] = 1.0

		output := make([]float32, 256)
		m.IMDCT(input, output)

		// Verify all outputs are finite
		for i := range output {
			if math.IsNaN(float64(output[i])) || math.IsInf(float64(output[i]), 0) {
				t.Errorf("output[%d] = %v (invalid)", i, output[i])
			}
		}
	})

	// Test case: N=2048 (long block)
	t.Run("n2048_dc_impulse", func(t *testing.T) {
		m := NewMDCT(2048)

		input := make([]float32, 1024)
		input[0] = 1.0

		output := make([]float32, 2048)
		m.IMDCT(input, output)

		// Verify all outputs are finite
		for i := range output {
			if math.IsNaN(float64(output[i])) || math.IsInf(float64(output[i]), 0) {
				t.Errorf("output[%d] = %v (invalid)", i, output[i])
			}
		}
	})
}

// TestIMDCT_Sizes verifies both AAC-required sizes work correctly.
func TestIMDCT_Sizes(t *testing.T) {
	sizes := []uint16{256, 2048}

	for _, n := range sizes {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			m := NewMDCT(n)

			input := make([]float32, n/2)
			// Fill with a simple pattern
			for i := range input {
				input[i] = float32(math.Sin(float64(i) * 0.1))
			}

			output := make([]float32, n)
			m.IMDCT(input, output)

			// Verify no NaN/Inf
			for i := range output {
				if math.IsNaN(float64(output[i])) {
					t.Errorf("output[%d] is NaN", i)
				}
				if math.IsInf(float64(output[i]), 0) {
					t.Errorf("output[%d] is Inf", i)
				}
			}
		})
	}
}
