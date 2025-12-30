// Copyright (c) 2025 Laurent Pelecq
// SPDX-License-Identifier: GPL-2.0-or-later

package spectrum

import (
	"math"
	"testing"
)

func TestMntTableSize(t *testing.T) {
	if len(mntTable) != 128 {
		t.Errorf("mntTable length = %d, want 128", len(mntTable))
	}
}

func TestExpTableSize(t *testing.T) {
	if len(expTable) != 128 {
		t.Errorf("expTable length = %d, want 128", len(expTable))
	}
}

func TestMntTableFirstValue(t *testing.T) {
	// First value from FAAD2
	expected := float32(0.953125)
	if math.Abs(float64(mntTable[0]-expected)) > 1e-7 {
		t.Errorf("mntTable[0] = %v, want %v", mntTable[0], expected)
	}
}

func TestExpTableFirstValues(t *testing.T) {
	// First few values are powers of 0.5
	testCases := []struct {
		index    int
		expected float32
	}{
		{0, 0.5},
		{1, 0.25},
		{2, 0.125},
		{3, 0.0625},
	}
	for _, tc := range testCases {
		if math.Abs(float64(expTable[tc.index]-tc.expected)) > 1e-7 {
			t.Errorf("expTable[%d] = %v, want %v", tc.index, expTable[tc.index], tc.expected)
		}
	}
}

func TestPredictionConstants(t *testing.T) {
	// Verify constants match FAAD2 ic_predict.h:40-41
	if math.Abs(float64(predAlpha-0.90625)) > 1e-7 {
		t.Errorf("predAlpha = %v, want 0.90625", predAlpha)
	}
	if math.Abs(float64(predA-0.953125)) > 1e-7 {
		t.Errorf("predA = %v, want 0.953125", predA)
	}
}

func TestMntTableLastValue(t *testing.T) {
	// Last value from FAAD2
	expected := float32(0.4785156250)
	if math.Abs(float64(mntTable[127]-expected)) > 1e-7 {
		t.Errorf("mntTable[127] = %v, want %v", mntTable[127], expected)
	}
}

func TestExpTableLastNonZeroValue(t *testing.T) {
	// Last non-zero value at index 125 from FAAD2
	expected := float32(0.00000000000000000000000000000000000001175494350822)
	if expTable[125] != expected {
		t.Errorf("expTable[125] = %v, want %v", expTable[125], expected)
	}
}

func TestExpTableUnderflowValues(t *testing.T) {
	// Last two values underflow to zero
	if expTable[126] != 0.0 {
		t.Errorf("expTable[126] = %v, want 0.0 (underflow)", expTable[126])
	}
	if expTable[127] != 0.0 {
		t.Errorf("expTable[127] = %v, want 0.0 (underflow)", expTable[127])
	}
}

func TestExpTablePowersOfHalf(t *testing.T) {
	// Verify expTable[i] = 0.5^(i+1) for first 24 entries
	// (after ~24 entries, float32 precision becomes an issue)
	for i := 0; i < 24; i++ {
		expected := float32(math.Pow(0.5, float64(i+1)))
		if math.Abs(float64(expTable[i]-expected)) > 1e-10 {
			t.Errorf("expTable[%d] = %v, want %v (0.5^%d)", i, expTable[i], expected, i+1)
		}
	}
}

func TestMntTableMonotonicallyDecreasing(t *testing.T) {
	// The mntTable should be monotonically decreasing (with some equal values)
	for i := 1; i < len(mntTable); i++ {
		if mntTable[i] > mntTable[i-1] {
			t.Errorf("mntTable[%d] = %v > mntTable[%d] = %v (not monotonically decreasing)",
				i, mntTable[i], i-1, mntTable[i-1])
		}
	}
}
