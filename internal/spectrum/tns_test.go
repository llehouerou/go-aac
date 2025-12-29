// internal/spectrum/tns_test.go
package spectrum

import (
	"math"
	"testing"

	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/syntax"
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

func TestTNSDecodeFrame_NoTNSData(t *testing.T) {
	// When tns_data_present is false, spectrum should be unchanged
	ics := &syntax.ICStream{
		TNSDataPresent: false,
	}

	original := []float64{1.0, 2.0, 3.0, 4.0}
	spec := make([]float64, len(original))
	copy(spec, original)

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4, // 44100 Hz
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	for i, want := range original {
		if spec[i] != want {
			t.Errorf("spec[%d]: got %v, want %v (should be unchanged)", i, spec[i], want)
		}
	}
}

func TestTNSDecodeFrame_SingleFilter(t *testing.T) {
	// Test with a single TNS filter on long block
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        1,
		NumWindowGroups:   1,
		WindowSequence:    syntax.OnlyLongSequence,
		NumSWB:            49,
		MaxSFB:            49,
		SWBOffsetMax:      1024,
		WindowGroupLength: [8]uint8{1},
	}

	// Set up SWB offsets (simplified - just use linear)
	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
		if ics.SWBOffset[i] > 1024 {
			ics.SWBOffset[i] = 1024
		}
	}

	// Set up TNS filter
	ics.TNS.NFilt[0] = 1
	ics.TNS.CoefRes[0] = 0      // 3-bit coefficients
	ics.TNS.Length[0][0] = 20   // Filter spans 20 SFBs
	ics.TNS.Order[0][0] = 1     // First-order filter
	ics.TNS.Direction[0][0] = 0 // Forward
	ics.TNS.CoefCompress[0][0] = 0
	ics.TNS.Coef[0][0][0] = 0 // Index 0 = coefficient 0.0

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = 1.0
	}

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4, // 44100 Hz
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	// With coefficient 0.0, filter should be identity (no change)
	// Just verify it doesn't crash and runs
	for i := range spec {
		if math.IsNaN(spec[i]) || math.IsInf(spec[i], 0) {
			t.Errorf("spec[%d] is invalid: %v", i, spec[i])
		}
	}
}

func TestTNSDecodeFrame_ShortBlock(t *testing.T) {
	// Test with 8 short windows
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        8,
		NumWindowGroups:   8,
		WindowSequence:    syntax.EightShortSequence,
		NumSWB:            14,
		MaxSFB:            14,
		SWBOffsetMax:      128,
		WindowGroupLength: [8]uint8{1, 1, 1, 1, 1, 1, 1, 1},
	}

	// Set up SWB offsets for short blocks
	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 8)
		if ics.SWBOffset[i] > 128 {
			ics.SWBOffset[i] = 128
		}
	}

	// Set up TNS filter for first window only
	ics.TNS.NFilt[0] = 1
	ics.TNS.CoefRes[0] = 0
	ics.TNS.Length[0][0] = 10
	ics.TNS.Order[0][0] = 1
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefCompress[0][0] = 0
	ics.TNS.Coef[0][0][0] = 0

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = 1.0
	}

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	// Verify no crashes and valid output
	for i := range spec {
		if math.IsNaN(spec[i]) || math.IsInf(spec[i], 0) {
			t.Errorf("spec[%d] is invalid: %v", i, spec[i])
		}
	}
}

func TestTNSDecodeFrame_MultipleFilters(t *testing.T) {
	// Test with multiple TNS filters per window
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        1,
		NumWindowGroups:   1,
		WindowSequence:    syntax.OnlyLongSequence,
		NumSWB:            49,
		MaxSFB:            49,
		SWBOffsetMax:      1024,
		WindowGroupLength: [8]uint8{1},
	}

	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
		if ics.SWBOffset[i] > 1024 {
			ics.SWBOffset[i] = 1024
		}
	}

	// Two filters
	ics.TNS.NFilt[0] = 2
	ics.TNS.CoefRes[0] = 1 // 4-bit coefficients

	// First filter: SFB 30-40
	ics.TNS.Length[0][0] = 10
	ics.TNS.Order[0][0] = 2
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefCompress[0][0] = 0
	ics.TNS.Coef[0][0][0] = 1
	ics.TNS.Coef[0][0][1] = 2

	// Second filter: SFB 40-49
	ics.TNS.Length[0][1] = 9
	ics.TNS.Order[0][1] = 1
	ics.TNS.Direction[0][1] = 1 // Backward
	ics.TNS.CoefCompress[0][1] = 0
	ics.TNS.Coef[0][1][0] = 3

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = float64(i % 10)
	}

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	// Verify no invalid values
	for i := range spec {
		if math.IsNaN(spec[i]) || math.IsInf(spec[i], 0) {
			t.Errorf("spec[%d] is invalid: %v", i, spec[i])
		}
	}
}

func TestTNSDecodeFrame_MaxOrder(t *testing.T) {
	// Test with maximum filter order (20)
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        1,
		NumWindowGroups:   1,
		WindowSequence:    syntax.OnlyLongSequence,
		NumSWB:            49,
		MaxSFB:            49,
		SWBOffsetMax:      1024,
		WindowGroupLength: [8]uint8{1},
	}

	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
		if ics.SWBOffset[i] > 1024 {
			ics.SWBOffset[i] = 1024
		}
	}

	ics.TNS.NFilt[0] = 1
	ics.TNS.CoefRes[0] = 1
	ics.TNS.Length[0][0] = 40
	ics.TNS.Order[0][0] = 20 // Max order
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefCompress[0][0] = 0

	// Fill all 20 coefficients
	for i := 0; i < 20; i++ {
		ics.TNS.Coef[0][0][i] = uint8(i % 16)
	}

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = 1.0
	}

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	// Should not crash and produce valid output
	for i := range spec {
		if math.IsNaN(spec[i]) || math.IsInf(spec[i], 0) {
			t.Errorf("spec[%d] is invalid: %v", i, spec[i])
		}
	}
}

func TestTNSDecodeFrame_OrderExceedsMax(t *testing.T) {
	// Test that order > TNSMaxOrder is clamped
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        1,
		NumWindowGroups:   1,
		WindowSequence:    syntax.OnlyLongSequence,
		NumSWB:            49,
		MaxSFB:            49,
		SWBOffsetMax:      1024,
		WindowGroupLength: [8]uint8{1},
	}

	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
		if ics.SWBOffset[i] > 1024 {
			ics.SWBOffset[i] = 1024
		}
	}

	ics.TNS.NFilt[0] = 1
	ics.TNS.CoefRes[0] = 0
	ics.TNS.Length[0][0] = 30
	ics.TNS.Order[0][0] = 25 // Exceeds TNSMaxOrder (20)
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefCompress[0][0] = 0

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = 1.0
	}

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	// Should not panic
	TNSDecodeFrame(spec, cfg)

	for i := range spec {
		if math.IsNaN(spec[i]) || math.IsInf(spec[i], 0) {
			t.Errorf("spec[%d] is invalid: %v", i, spec[i])
		}
	}
}

func TestTNSDecodeFrame_ZeroRegion(t *testing.T) {
	// Test when filter region computes to zero size
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        1,
		NumWindowGroups:   1,
		WindowSequence:    syntax.OnlyLongSequence,
		NumSWB:            49,
		MaxSFB:            5, // Very low max_sfb
		SWBOffsetMax:      100,
		WindowGroupLength: [8]uint8{1},
	}

	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
	}

	ics.TNS.NFilt[0] = 1
	ics.TNS.CoefRes[0] = 0
	ics.TNS.Length[0][0] = 10
	ics.TNS.Order[0][0] = 5
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefCompress[0][0] = 0

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = 1.0
	}

	original := make([]float64, len(spec))
	copy(original, spec)

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	// When region is zero or negative, spectrum should be unchanged
	// (filter is skipped)
	for i, want := range original {
		if spec[i] != want {
			t.Errorf("spec[%d]: got %v, want %v (should be unchanged)", i, spec[i], want)
		}
	}
}

func TestTNSDecodeFrame_BackwardDirection(t *testing.T) {
	// Test backward filtering direction explicitly
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        1,
		NumWindowGroups:   1,
		WindowSequence:    syntax.OnlyLongSequence,
		NumSWB:            49,
		MaxSFB:            49,
		SWBOffsetMax:      1024,
		WindowGroupLength: [8]uint8{1},
	}

	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
		if ics.SWBOffset[i] > 1024 {
			ics.SWBOffset[i] = 1024
		}
	}

	ics.TNS.NFilt[0] = 1
	ics.TNS.CoefRes[0] = 0
	ics.TNS.Length[0][0] = 20
	ics.TNS.Order[0][0] = 1
	ics.TNS.Direction[0][0] = 1 // Backward direction
	ics.TNS.CoefCompress[0][0] = 0
	ics.TNS.Coef[0][0][0] = 1 // Non-zero coefficient

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = 1.0
	}

	original := make([]float64, len(spec))
	copy(original, spec)

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	// Verify no invalid values
	for i := range spec {
		if math.IsNaN(spec[i]) || math.IsInf(spec[i], 0) {
			t.Errorf("spec[%d] is invalid: %v", i, spec[i])
		}
	}

	// Verify spectrum was actually modified
	modified := false
	for i := range spec {
		if spec[i] != original[i] {
			modified = true
			break
		}
	}
	if !modified {
		t.Error("Spectrum was not modified despite non-zero filter coefficient")
	}
}

func TestTNSDecodeFrame_NonZeroCoefficients(t *testing.T) {
	// Test with non-zero coefficients to verify actual filtering
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        1,
		NumWindowGroups:   1,
		WindowSequence:    syntax.OnlyLongSequence,
		NumSWB:            49,
		MaxSFB:            49,
		SWBOffsetMax:      1024,
		WindowGroupLength: [8]uint8{1},
	}

	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
		if ics.SWBOffset[i] > 1024 {
			ics.SWBOffset[i] = 1024
		}
	}

	ics.TNS.NFilt[0] = 1
	ics.TNS.CoefRes[0] = 0
	ics.TNS.Length[0][0] = 49 // Filter spans all SFBs (bottom=0, top=49)
	ics.TNS.Order[0][0] = 1
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefCompress[0][0] = 0
	ics.TNS.Coef[0][0][0] = 1 // Index 1 = 0.4338837391

	// Set up impulse response test
	// Place impulse within the filter region (start of spectrum)
	spec := make([]float64, 1024)
	spec[0] = 1.0 // Impulse at start

	original := make([]float64, 1024)
	copy(original, spec)

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	// Verify spectrum was actually modified (not identity)
	modified := false
	for i := range spec {
		if spec[i] != original[i] {
			modified = true
			break
		}
	}
	if !modified {
		t.Error("Spectrum was not modified despite non-zero filter coefficient")
	}

	// Verify no invalid values
	for i := range spec {
		if math.IsNaN(spec[i]) || math.IsInf(spec[i], 0) {
			t.Errorf("spec[%d] is invalid: %v", i, spec[i])
		}
	}
}

func TestTNSMAFilter_Basic(t *testing.T) {
	// Simple test: apply MA filter with known coefficients
	// MA filter: y[n] = x[n] + lpc[1]*x[n-1] + lpc[2]*x[n-2] + ...
	spec := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	// Simple 2nd order filter: y[n] = x[n] + 0.5*x[n-1] + 0.25*x[n-2]
	lpc := []float64{1.0, 0.5, 0.25}

	tnsMAFilter(spec, 5, 1, lpc, 2)

	// Expected results:
	// y[0] = x[0] + 0.5*0 + 0.25*0 = 1.0
	// y[1] = x[1] + 0.5*x[0] + 0.25*0 = 2.0 + 0.5*1.0 = 2.5
	// y[2] = x[2] + 0.5*x[1] + 0.25*x[0] = 3.0 + 0.5*2.0 + 0.25*1.0 = 4.25
	// y[3] = x[3] + 0.5*x[2] + 0.25*x[1] = 4.0 + 0.5*3.0 + 0.25*2.0 = 6.0
	// y[4] = x[4] + 0.5*x[3] + 0.25*x[2] = 5.0 + 0.5*4.0 + 0.25*3.0 = 7.75
	expected := []float64{1.0, 2.5, 4.25, 6.0, 7.75}

	for i, exp := range expected {
		if math.Abs(spec[i]-exp) > 1e-10 {
			t.Errorf("sample %d: got %v, want %v", i, spec[i], exp)
		}
	}
}

func TestTNSMAFilter_BackwardDirection(t *testing.T) {
	// Test backward direction (inc = -1)
	spec := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	lpc := []float64{1.0, 0.5, 0.25}

	// Start from index 4, go backward
	tnsMAFilterWithOffset(spec, 4, 5, -1, lpc, 2)

	// Processing order: spec[4], spec[3], spec[2], spec[1], spec[0]
	// y[4] = x[4] + 0.5*0 + 0.25*0 = 5.0
	// y[3] = x[3] + 0.5*x[4] + 0.25*0 = 4.0 + 0.5*5.0 = 6.5
	// y[2] = x[2] + 0.5*x[3] + 0.25*x[4] = 3.0 + 0.5*4.0 + 0.25*5.0 = 6.25
	// y[1] = x[1] + 0.5*x[2] + 0.25*x[3] = 2.0 + 0.5*3.0 + 0.25*4.0 = 4.5
	// y[0] = x[0] + 0.5*x[1] + 0.25*x[2] = 1.0 + 0.5*2.0 + 0.25*3.0 = 2.75
	expected := []float64{2.75, 4.5, 6.25, 6.5, 5.0}

	for i, exp := range expected {
		if math.Abs(spec[i]-exp) > 1e-10 {
			t.Errorf("sample %d: got %v, want %v", i, spec[i], exp)
		}
	}
}

func TestTNSMAFilter_ZeroOrder(t *testing.T) {
	spec := []float64{1.0, 2.0, 3.0}
	original := make([]float64, len(spec))
	copy(original, spec)

	lpc := []float64{1.0}
	tnsMAFilter(spec, 3, 1, lpc, 0)

	// Zero order filter should not modify spectrum
	for i := range spec {
		if spec[i] != original[i] {
			t.Errorf("sample %d modified with zero order: got %v, want %v", i, spec[i], original[i])
		}
	}
}

func TestTNSMAFilter_ZeroSize(t *testing.T) {
	spec := []float64{1.0, 2.0, 3.0}
	original := make([]float64, len(spec))
	copy(original, spec)

	lpc := []float64{1.0, 0.5}
	tnsMAFilter(spec, 0, 1, lpc, 1)

	// Zero size should not modify spectrum
	for i := range spec {
		if spec[i] != original[i] {
			t.Errorf("sample %d modified with zero size: got %v, want %v", i, spec[i], original[i])
		}
	}
}

func TestTNSEncodeFrame_NoTNSData(t *testing.T) {
	spec := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	original := make([]float64, len(spec))
	copy(original, spec)

	ics := &syntax.ICStream{
		TNSDataPresent: false,
	}

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4, // 44100 Hz
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSEncodeFrame(spec, cfg)

	// No TNS data - spectrum should be unchanged
	for i := range spec {
		if spec[i] != original[i] {
			t.Errorf("sample %d modified without TNS data: got %v, want %v", i, spec[i], original[i])
		}
	}
}
