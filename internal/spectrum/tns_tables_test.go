// internal/spectrum/tns_tables_test.go
package spectrum

import (
	"math"
	"testing"
)

func TestTNSCoefTables_Length(t *testing.T) {
	// Each table should have 16 entries
	if len(tnsCoef03) != 16 {
		t.Errorf("tnsCoef03: got %d entries, want 16", len(tnsCoef03))
	}
	if len(tnsCoef04) != 16 {
		t.Errorf("tnsCoef04: got %d entries, want 16", len(tnsCoef04))
	}
	if len(tnsCoef13) != 16 {
		t.Errorf("tnsCoef13: got %d entries, want 16", len(tnsCoef13))
	}
	if len(tnsCoef14) != 16 {
		t.Errorf("tnsCoef14: got %d entries, want 16", len(tnsCoef14))
	}
}

func TestTNSCoefTables_Values(t *testing.T) {
	// Verify first values from FAAD2
	const tolerance = 1e-9

	// tns_coef_0_3[0] = 0.0
	if math.Abs(tnsCoef03[0]-0.0) > tolerance {
		t.Errorf("tnsCoef03[0]: got %v, want 0.0", tnsCoef03[0])
	}
	// tns_coef_0_3[1] = 0.4338837391
	if math.Abs(tnsCoef03[1]-0.4338837391) > tolerance {
		t.Errorf("tnsCoef03[1]: got %v, want 0.4338837391", tnsCoef03[1])
	}
	// tns_coef_0_4[7] = 0.9945218954
	if math.Abs(tnsCoef04[7]-0.9945218954) > tolerance {
		t.Errorf("tnsCoef04[7]: got %v, want 0.9945218954", tnsCoef04[7])
	}
}

func TestGetTNSCoefTable(t *testing.T) {
	tests := []struct {
		coefCompress uint8
		coefRes      uint8
		wantTable    *[16]float64
	}{
		{0, 0, &tnsCoef03}, // coef_compress=0, coef_res_bits=3
		{0, 1, &tnsCoef04}, // coef_compress=0, coef_res_bits=4
		{1, 0, &tnsCoef13}, // coef_compress=1, coef_res_bits=3
		{1, 1, &tnsCoef14}, // coef_compress=1, coef_res_bits=4
	}

	for _, tc := range tests {
		got := getTNSCoefTable(tc.coefCompress, tc.coefRes)
		if got != tc.wantTable {
			t.Errorf("getTNSCoefTable(%d, %d): got wrong table", tc.coefCompress, tc.coefRes)
		}
	}
}
