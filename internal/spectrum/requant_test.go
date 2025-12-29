package spectrum

import (
	"math"
	"testing"

	"github.com/llehouerou/go-aac/internal/tables"
)

func TestInverseQuantize_ZeroValues(t *testing.T) {
	quantData := []int16{0, 0, 0, 0}
	specData := make([]float64, 4)

	err := InverseQuantize(quantData, specData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, v := range specData {
		if v != 0 {
			t.Errorf("specData[%d]: got %f, want 0", i, v)
		}
	}
}

func TestInverseQuantize_PositiveValues(t *testing.T) {
	quantData := []int16{1, 2, 3, 4}
	specData := make([]float64, 4)

	err := InverseQuantize(quantData, specData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify against IQTable values
	for i, q := range quantData {
		expected := tables.IQTable[q]
		if specData[i] != expected {
			t.Errorf("specData[%d]: got %f, want %f", i, specData[i], expected)
		}
	}
}

func TestInverseQuantize_NegativeValues(t *testing.T) {
	quantData := []int16{-1, -2, -3, -4}
	specData := make([]float64, 4)

	err := InverseQuantize(quantData, specData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Negative values should return negative of IQTable lookup
	for i, q := range quantData {
		expected := -tables.IQTable[-q]
		if specData[i] != expected {
			t.Errorf("specData[%d]: got %f, want %f", i, specData[i], expected)
		}
	}
}

func TestInverseQuantize_MixedValues(t *testing.T) {
	quantData := []int16{0, 5, -10, 100, -200}
	specData := make([]float64, 5)

	err := InverseQuantize(quantData, specData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check each value individually
	if specData[0] != 0 {
		t.Errorf("specData[0]: got %f, want 0", specData[0])
	}
	if specData[1] != tables.IQTable[5] {
		t.Errorf("specData[1]: got %f, want %f", specData[1], tables.IQTable[5])
	}
	if specData[2] != -tables.IQTable[10] {
		t.Errorf("specData[2]: got %f, want %f", specData[2], -tables.IQTable[10])
	}
	if specData[3] != tables.IQTable[100] {
		t.Errorf("specData[3]: got %f, want %f", specData[3], tables.IQTable[100])
	}
	if specData[4] != -tables.IQTable[200] {
		t.Errorf("specData[4]: got %f, want %f", specData[4], -tables.IQTable[200])
	}
}

func TestInverseQuantize_LargeValues(t *testing.T) {
	// Test near the table limit (8191)
	quantData := []int16{8000, -8000, 8191, -8191}
	specData := make([]float64, 4)

	err := InverseQuantize(quantData, specData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if specData[0] != tables.IQTable[8000] {
		t.Errorf("specData[0]: got %f, want %f", specData[0], tables.IQTable[8000])
	}
	if specData[1] != -tables.IQTable[8000] {
		t.Errorf("specData[1]: got %f, want %f", specData[1], -tables.IQTable[8000])
	}
	if specData[2] != tables.IQTable[8191] {
		t.Errorf("specData[2]: got %f, want %f", specData[2], tables.IQTable[8191])
	}
	if specData[3] != -tables.IQTable[8191] {
		t.Errorf("specData[3]: got %f, want %f", specData[3], -tables.IQTable[8191])
	}
}

func TestInverseQuantize_Overflow(t *testing.T) {
	// Values >= 8192 should error
	quantData := []int16{8192}
	specData := make([]float64, 1)

	err := InverseQuantize(quantData, specData)
	if err != tables.ErrIQTableOverflow {
		t.Errorf("expected ErrIQTableOverflow, got %v", err)
	}

	// Negative overflow
	quantData = []int16{-8192}
	err = InverseQuantize(quantData, specData)
	if err != tables.ErrIQTableOverflow {
		t.Errorf("expected ErrIQTableOverflow for negative, got %v", err)
	}
}

func TestInverseQuantize_Formula(t *testing.T) {
	// Verify the formula: spec = sign(q) * |q|^(4/3)
	testCases := []int16{1, 2, 8, 27, 64, 125}

	for _, q := range testCases {
		quantData := []int16{q}
		specData := make([]float64, 1)

		err := InverseQuantize(quantData, specData)
		if err != nil {
			t.Fatalf("unexpected error for q=%d: %v", q, err)
		}

		// Calculate expected: q^(4/3)
		expected := math.Pow(float64(q), 4.0/3.0)

		// Allow small floating point tolerance
		if math.Abs(specData[0]-expected) > 1e-6 {
			t.Errorf("q=%d: got %f, want %f (diff=%e)", q, specData[0], expected, specData[0]-expected)
		}
	}
}

func TestInverseQuantize_EmptyInput(t *testing.T) {
	quantData := []int16{}
	specData := []float64{}

	err := InverseQuantize(quantData, specData)
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
}

func TestInverseQuantize_LengthMismatch(t *testing.T) {
	quantData := []int16{1, 2, 3}
	specData := make([]float64, 2) // Too short

	err := InverseQuantize(quantData, specData)
	if err == nil {
		t.Error("expected error for length mismatch, got nil")
	}
}
