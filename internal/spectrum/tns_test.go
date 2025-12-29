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
