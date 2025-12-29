// internal/spectrum/tns_test.go
package spectrum

import (
	"math"
	"testing"
)

func TestTNSDecodeCoef_Order1(t *testing.T) {
	// Simple case: order=1, coefRes=0 (3-bit), compress=0, coef[0]=1
	coef := [32]uint8{1}
	lpc := make([]float64, TNSMaxOrder+1)

	tnsDecodeCoef(1, 0, 0, coef[:], lpc)

	// lpc[0] should always be 1.0
	if lpc[0] != 1.0 {
		t.Errorf("lpc[0]: got %v, want 1.0", lpc[0])
	}

	// lpc[1] should be tnsCoef03[1] = 0.4338837391
	const tolerance = 1e-9
	if math.Abs(lpc[1]-0.4338837391) > tolerance {
		t.Errorf("lpc[1]: got %v, want 0.4338837391", lpc[1])
	}
}

func TestTNSDecodeCoef_Order2(t *testing.T) {
	// Order=2 to verify Levinson-Durbin recursion
	coef := [32]uint8{1, 2} // indices into table
	lpc := make([]float64, TNSMaxOrder+1)

	tnsDecodeCoef(2, 0, 0, coef[:], lpc)

	// lpc[0] should always be 1.0
	if lpc[0] != 1.0 {
		t.Errorf("lpc[0]: got %v, want 1.0", lpc[0])
	}

	// Verify non-zero coefficients were computed
	if lpc[1] == 0.0 && lpc[2] == 0.0 {
		t.Error("lpc coefficients should be non-zero for order 2")
	}
}

func TestTNSDecodeCoef_Order0(t *testing.T) {
	// Edge case: order=0 should just set lpc[0]=1.0
	coef := [32]uint8{}
	lpc := make([]float64, TNSMaxOrder+1)

	tnsDecodeCoef(0, 0, 0, coef[:], lpc)

	if lpc[0] != 1.0 {
		t.Errorf("lpc[0]: got %v, want 1.0", lpc[0])
	}
}

func TestTNSDecodeCoef_DifferentTables(t *testing.T) {
	// Test that different coefRes/compress values select different tables
	coef := [32]uint8{1}
	lpc03 := make([]float64, TNSMaxOrder+1)
	lpc04 := make([]float64, TNSMaxOrder+1)

	tnsDecodeCoef(1, 0, 0, coef[:], lpc03) // 3-bit, no compress
	tnsDecodeCoef(1, 1, 0, coef[:], lpc04) // 4-bit, no compress

	// Different tables should produce different results
	if lpc03[1] == lpc04[1] {
		t.Errorf("Different tables should produce different lpc[1]: both got %v", lpc03[1])
	}
}

func TestTNSDecodeCoef_LevinsonDurbinRecursion(t *testing.T) {
	// Verify Levinson-Durbin recursion with order=2
	// coef[0]=1 -> tmp2[0] = tnsCoef03[1] = 0.4338837391
	// coef[1]=2 -> tmp2[1] = tnsCoef03[2] = 0.7818314825
	//
	// m=1: lpc[1] = tmp2[0] = 0.4338837391
	// m=2: lpc[2] = tmp2[1] = 0.7818314825
	//      b[1] = lpc[1] + lpc[2]*lpc[1] = 0.4338837391 + 0.7818314825*0.4338837391
	//           = 0.4338837391 + 0.3392193... = 0.7731030...
	//      lpc[1] = b[1] = 0.7731030...
	coef := [32]uint8{1, 2}
	lpc := make([]float64, TNSMaxOrder+1)

	tnsDecodeCoef(2, 0, 0, coef[:], lpc)

	const tolerance = 1e-9

	// lpc[0] is always 1.0
	if math.Abs(lpc[0]-1.0) > tolerance {
		t.Errorf("lpc[0]: got %v, want 1.0", lpc[0])
	}

	// lpc[2] should be the raw coefficient from table
	if math.Abs(lpc[2]-0.7818314825) > tolerance {
		t.Errorf("lpc[2]: got %v, want 0.7818314825", lpc[2])
	}

	// lpc[1] should be updated by the recursion
	// lpc[1] = 0.4338837391 + 0.7818314825 * 0.4338837391
	expected := 0.4338837391 + 0.7818314825*0.4338837391
	if math.Abs(lpc[1]-expected) > tolerance {
		t.Errorf("lpc[1]: got %v, want %v", lpc[1], expected)
	}
}

func TestTNSDecodeCoef_AllTableCombinations(t *testing.T) {
	// Test all 4 table combinations
	tests := []struct {
		coefRes      uint8
		coefCompress uint8
		coefIdx      uint8
		expectedLpc1 float64
	}{
		{0, 0, 1, 0.4338837391}, // tnsCoef03[1]
		{1, 0, 1, 0.2079116908}, // tnsCoef04[1]
		{0, 1, 1, 0.4338837391}, // tnsCoef13[1]
		{1, 1, 1, 0.2079116908}, // tnsCoef14[1]
	}

	const tolerance = 1e-9

	for _, tc := range tests {
		coef := [32]uint8{tc.coefIdx}
		lpc := make([]float64, TNSMaxOrder+1)

		tnsDecodeCoef(1, tc.coefRes, tc.coefCompress, coef[:], lpc)

		if math.Abs(lpc[1]-tc.expectedLpc1) > tolerance {
			t.Errorf("coefRes=%d, coefCompress=%d: lpc[1] got %v, want %v",
				tc.coefRes, tc.coefCompress, lpc[1], tc.expectedLpc1)
		}
	}
}

func TestTNSDecodeCoef_HigherOrder(t *testing.T) {
	// Test with order=5 to verify recursion works for higher orders
	coef := [32]uint8{1, 1, 1, 1, 1}
	lpc := make([]float64, TNSMaxOrder+1)

	tnsDecodeCoef(5, 0, 0, coef[:], lpc)

	// lpc[0] should always be 1.0
	if lpc[0] != 1.0 {
		t.Errorf("lpc[0]: got %v, want 1.0", lpc[0])
	}

	// All coefficients should be non-zero (they're all set to non-zero table value)
	for i := 1; i <= 5; i++ {
		if lpc[i] == 0.0 {
			t.Errorf("lpc[%d] should be non-zero", i)
		}
	}

	// Coefficients beyond order should be zero (from slice initialization)
	for i := 6; i <= TNSMaxOrder; i++ {
		if lpc[i] != 0.0 {
			t.Errorf("lpc[%d] should be zero, got %v", i, lpc[i])
		}
	}
}

func TestTNSDecodeCoef_ZeroCoefficients(t *testing.T) {
	// Test with coef[i]=0 which maps to 0.0 in all tables
	coef := [32]uint8{0, 0, 0}
	lpc := make([]float64, TNSMaxOrder+1)

	tnsDecodeCoef(3, 0, 0, coef[:], lpc)

	// lpc[0] should always be 1.0
	if lpc[0] != 1.0 {
		t.Errorf("lpc[0]: got %v, want 1.0", lpc[0])
	}

	// With all zero reflection coefficients, all LPC coefficients should be 0
	// (except lpc[0] which is always 1.0)
	for i := 1; i <= 3; i++ {
		if lpc[i] != 0.0 {
			t.Errorf("lpc[%d]: got %v, want 0.0", i, lpc[i])
		}
	}
}

func TestTNSARFilter_Identity(t *testing.T) {
	// When lpc = [1, 0, 0, ...], filter should be identity (y = x)
	lpc := make([]float64, TNSMaxOrder+1)
	lpc[0] = 1.0

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	spec := make([]float64, len(input))
	copy(spec, input)

	tnsARFilter(spec, int16(len(spec)), 1, lpc, 0)

	for i, want := range input {
		if spec[i] != want {
			t.Errorf("spec[%d]: got %v, want %v", i, spec[i], want)
		}
	}
}

func TestTNSARFilter_Order1(t *testing.T) {
	// Simple first-order filter: y[n] = x[n] - 0.5*y[n-1]
	lpc := make([]float64, TNSMaxOrder+1)
	lpc[0] = 1.0
	lpc[1] = 0.5

	spec := []float64{1.0, 0.0, 0.0, 0.0, 0.0}

	tnsARFilter(spec, int16(len(spec)), 1, lpc, 1)

	// y[0] = 1.0 - 0.5*0 = 1.0
	// y[1] = 0.0 - 0.5*1.0 = -0.5
	// y[2] = 0.0 - 0.5*(-0.5) = 0.25
	// y[3] = 0.0 - 0.5*0.25 = -0.125
	// y[4] = 0.0 - 0.5*(-0.125) = 0.0625

	expected := []float64{1.0, -0.5, 0.25, -0.125, 0.0625}
	const tolerance = 1e-9

	for i, want := range expected {
		if math.Abs(spec[i]-want) > tolerance {
			t.Errorf("spec[%d]: got %v, want %v", i, spec[i], want)
		}
	}
}

func TestTNSARFilter_Backward(t *testing.T) {
	// Test backward filtering (inc = -1)
	// In FAAD2, for backward filtering, the pointer starts at the last element
	// and uses negative increment. In Go, we pass the full slice with a start offset.
	lpc := make([]float64, TNSMaxOrder+1)
	lpc[0] = 1.0
	lpc[1] = 0.5

	spec := []float64{0.0, 0.0, 0.0, 0.0, 1.0}

	// For backward filtering, we pass startOffset=4 (last element index)
	// The function will process from index 4 down to index 0
	tnsARFilterWithOffset(spec, 4, 5, -1, lpc, 1)

	// Should filter from end to start
	// y[4] = 1.0 - 0.5*0 = 1.0
	// y[3] = 0.0 - 0.5*1.0 = -0.5
	// y[2] = 0.0 - 0.5*(-0.5) = 0.25
	// etc.

	expected := []float64{0.0625, -0.125, 0.25, -0.5, 1.0}
	const tolerance = 1e-9

	for i, want := range expected {
		if math.Abs(spec[i]-want) > tolerance {
			t.Errorf("spec[%d]: got %v, want %v", i, spec[i], want)
		}
	}
}

func TestTNSARFilter_Order2(t *testing.T) {
	// Test second-order filter: y[n] = x[n] - 0.5*y[n-1] - 0.25*y[n-2]
	lpc := make([]float64, TNSMaxOrder+1)
	lpc[0] = 1.0
	lpc[1] = 0.5
	lpc[2] = 0.25

	spec := []float64{1.0, 0.0, 0.0, 0.0}

	tnsARFilter(spec, int16(len(spec)), 1, lpc, 2)

	// y[0] = 1.0 - 0.5*0 - 0.25*0 = 1.0
	// y[1] = 0.0 - 0.5*1.0 - 0.25*0 = -0.5
	// y[2] = 0.0 - 0.5*(-0.5) - 0.25*1.0 = 0.25 - 0.25 = 0.0
	// y[3] = 0.0 - 0.5*0.0 - 0.25*(-0.5) = 0.125

	expected := []float64{1.0, -0.5, 0.0, 0.125}
	const tolerance = 1e-9

	for i, want := range expected {
		if math.Abs(spec[i]-want) > tolerance {
			t.Errorf("spec[%d]: got %v, want %v", i, spec[i], want)
		}
	}
}

func TestTNSARFilter_ZeroSize(t *testing.T) {
	// Edge case: size=0 should do nothing
	lpc := make([]float64, TNSMaxOrder+1)
	lpc[0] = 1.0
	lpc[1] = 0.5

	spec := []float64{1.0, 2.0, 3.0}
	original := make([]float64, len(spec))
	copy(original, spec)

	tnsARFilter(spec, 0, 1, lpc, 1)

	for i, want := range original {
		if spec[i] != want {
			t.Errorf("spec[%d]: got %v, want %v (should be unchanged)", i, spec[i], want)
		}
	}
}

func TestTNSARFilter_ZeroOrder(t *testing.T) {
	// Edge case: order=0 should do nothing (identity filter)
	lpc := make([]float64, TNSMaxOrder+1)
	lpc[0] = 1.0
	lpc[1] = 0.5 // This should be ignored since order=0

	spec := []float64{1.0, 2.0, 3.0}
	original := make([]float64, len(spec))
	copy(original, spec)

	tnsARFilter(spec, int16(len(spec)), 1, lpc, 0)

	for i, want := range original {
		if spec[i] != want {
			t.Errorf("spec[%d]: got %v, want %v (should be unchanged)", i, spec[i], want)
		}
	}
}

func TestTNSARFilter_HigherOrder(t *testing.T) {
	// Test with order=5 to verify ringbuffer wraparound works
	lpc := make([]float64, TNSMaxOrder+1)
	lpc[0] = 1.0
	for i := 1; i <= 5; i++ {
		lpc[i] = 0.1 // Small coefficients
	}

	spec := make([]float64, 10)
	spec[0] = 1.0 // Impulse

	tnsARFilter(spec, int16(len(spec)), 1, lpc, 5)

	// Just verify it runs without panic and produces non-trivial output
	if spec[0] != 1.0 {
		t.Errorf("spec[0]: got %v, want 1.0", spec[0])
	}
	// Later samples should be non-zero due to filter response
	hasNonZero := false
	for i := 1; i < len(spec); i++ {
		if spec[i] != 0.0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("Higher order filter should produce non-zero response")
	}
}

func TestTNSARFilter_PartialSlice(t *testing.T) {
	// Test filtering a portion of a larger spectrum
	lpc := make([]float64, TNSMaxOrder+1)
	lpc[0] = 1.0
	lpc[1] = 0.5

	// Full spectrum, but only filter middle portion
	spec := []float64{100.0, 1.0, 0.0, 0.0, 200.0}

	// Filter only elements 1-3 (3 samples starting at offset 1)
	tnsARFilterWithOffset(spec, 1, 3, 1, lpc, 1)

	// spec[0] and spec[4] should be unchanged
	if spec[0] != 100.0 {
		t.Errorf("spec[0]: got %v, want 100.0 (unchanged)", spec[0])
	}
	if spec[4] != 200.0 {
		t.Errorf("spec[4]: got %v, want 200.0 (unchanged)", spec[4])
	}

	// Filtered portion: y[1] = 1.0, y[2] = 0 - 0.5*1 = -0.5, y[3] = 0 - 0.5*(-0.5) = 0.25
	const tolerance = 1e-9
	if math.Abs(spec[1]-1.0) > tolerance {
		t.Errorf("spec[1]: got %v, want 1.0", spec[1])
	}
	if math.Abs(spec[2]-(-0.5)) > tolerance {
		t.Errorf("spec[2]: got %v, want -0.5", spec[2])
	}
	if math.Abs(spec[3]-0.25) > tolerance {
		t.Errorf("spec[3]: got %v, want 0.25", spec[3])
	}
}
