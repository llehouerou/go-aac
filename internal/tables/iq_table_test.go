// internal/tables/iq_table_test.go
package tables

import (
	"math"
	"testing"
)

func TestIQTable_Size(t *testing.T) {
	if len(IQTable) != IQTableSize {
		t.Errorf("IQTable size: got %d, want %d", len(IQTable), IQTableSize)
	}
}

func TestIQTable_FirstValues(t *testing.T) {
	// Known values from FAAD2's iq_table.h
	expected := []float64{
		0,
		1,
		2.5198420997897464,
		4.3267487109222245,
		6.3496042078727974,
		8.5498797333834844,
		10.902723556992836,
		13.390518279406722,
		15.999999999999998,
		18.720754407467133,
	}

	for i, want := range expected {
		got := IQTable[i]
		if got != want {
			t.Errorf("IQTable[%d]: got %v, want %v", i, got, want)
		}
	}
}

func TestIQTable_FAAD2Reference(t *testing.T) {
	// Reference values extracted from ~/dev/faad2/libfaad/iq_table.h
	// Testing key indices across the table
	// These are the EXACT values from FAAD2, not recalculated with math.Pow
	tests := []struct {
		index int
		value float64
	}{
		{0, 0},
		{1, 1},
		{2, 2.5198420997897464},
		{8, 15.999999999999998},
		{16, 40.317473596635935},
		{64, 255.99999999999991},
		{100, 464.15888336127773},
		{256, 1625.4986772154357},
		{512, 4095.9999999999982},
		{1000, 9999.9999999999945},
		{1024, 10321.273240738796},
		{2048, 26007.978835446964},
		{4096, 65535.999999999956},
		{8191, 165113.4940829452}, // Last entry
	}

	for _, tc := range tests {
		got := IQTable[tc.index]
		// Allow small floating-point tolerance
		if !floatEquals(got, tc.value, 1e-10) {
			t.Errorf("IQTable[%d]: got %.17g, want %.17g", tc.index, got, tc.value)
		}
	}
}

func floatEquals(a, b, epsilon float64) bool {
	if a == b {
		return true
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

func TestIQTable_Formula(t *testing.T) {
	// Verify the mathematical formula is correct: IQTable[i] = i^(4/3)
	// Note: FAAD2's table was precomputed and may have tiny differences from
	// Go's math.Pow due to floating-point precision, so we use a tolerance.
	for i := 0; i < 100; i++ {
		expected := math.Pow(float64(i), 4.0/3.0)
		got := IQTable[i]
		// Use relative tolerance for larger values, absolute for small
		epsilon := 1e-14
		if expected > 1 {
			epsilon = expected * 1e-14
		}
		if !floatEquals(got, expected, epsilon) {
			t.Errorf("IQTable[%d]: got %.17g, want %.17g", i, got, expected)
		}
	}
}

func TestIQuant(t *testing.T) {
	tests := []struct {
		name     string
		input    int16
		expected float64
		hasError bool
	}{
		{"zero", 0, 0, false},
		{"positive_1", 1, 1, false},
		{"positive_8", 8, 15.999999999999998, false},
		{"negative_1", -1, -1, false},
		{"negative_8", -8, -15.999999999999998, false},
		{"positive_max", 8191, 165113.4940829452, false},
		{"negative_max", -8191, -165113.4940829452, false},
		{"overflow_positive", 8192, 0, true},
		{"overflow_negative", -8192, 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IQuant(tc.input)
			if tc.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !floatEquals(got, tc.expected, 1e-10) {
					t.Errorf("got %v, want %v", got, tc.expected)
				}
			}
		})
	}
}
