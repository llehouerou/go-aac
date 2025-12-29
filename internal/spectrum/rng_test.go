// Package spectrum tests for rng.go
package spectrum

import (
	"math/bits"
	"testing"
)

func TestParityTable_KnownValues(t *testing.T) {
	// Parity of 0x00 = 0 (even number of 1s)
	if parity[0x00] != 0 {
		t.Errorf("parity[0x00] = %d, want 0", parity[0x00])
	}

	// Parity of 0x01 = 1 (odd number of 1s)
	if parity[0x01] != 1 {
		t.Errorf("parity[0x01] = %d, want 1", parity[0x01])
	}

	// Parity of 0xFF = 0 (8 ones = even)
	if parity[0xFF] != 0 {
		t.Errorf("parity[0xFF] = %d, want 0", parity[0xFF])
	}

	// Parity of 0x03 = 0 (2 ones = even)
	if parity[0x03] != 0 {
		t.Errorf("parity[0x03] = %d, want 0", parity[0x03])
	}

	// Parity of 0x07 = 1 (3 ones = odd)
	if parity[0x07] != 1 {
		t.Errorf("parity[0x07] = %d, want 1", parity[0x07])
	}
}

func TestParityTable_AllValues(t *testing.T) {
	// Verify all 256 entries against computed parity
	for i := 0; i < 256; i++ {
		expected := uint8(bits.OnesCount8(uint8(i)) % 2)
		if parity[i] != expected {
			t.Errorf("parity[0x%02X] = %d, want %d", i, parity[i], expected)
		}
	}
}

func TestRNG_Deterministic(t *testing.T) {
	// Same initial state should produce same sequence
	r1a, r2a := uint32(1), uint32(2)
	r1b, r2b := uint32(1), uint32(2)

	for i := 0; i < 100; i++ {
		valA := RNG(&r1a, &r2a)
		valB := RNG(&r1b, &r2b)
		if valA != valB {
			t.Errorf("iteration %d: valA=%d, valB=%d", i, valA, valB)
		}
	}
}

func TestRNG_StateUpdates(t *testing.T) {
	// RNG should modify state
	r1, r2 := uint32(0x12345678), uint32(0x87654321)
	origR1, origR2 := r1, r2

	_ = RNG(&r1, &r2)

	if r1 == origR1 && r2 == origR2 {
		t.Error("RNG did not update state")
	}
}

func TestRNG_FullCoverage(t *testing.T) {
	// Generate many values, verify they're not all the same
	r1, r2 := uint32(1), uint32(1)
	seen := make(map[uint32]bool)

	for i := 0; i < 1000; i++ {
		val := RNG(&r1, &r2)
		seen[val] = true
	}

	// Should have many distinct values
	if len(seen) < 500 {
		t.Errorf("only %d distinct values in 1000 iterations", len(seen))
	}
}

// TestRNG_FAAD2Reference validates against exact FAAD2 ne_rng() output.
// Reference values generated from ~/dev/faad2/libfaad/common.c:231-241
func TestRNG_FAAD2Reference(t *testing.T) {
	tests := []struct {
		name        string
		initR1      uint32
		initR2      uint32
		expectedSeq []struct {
			val, r1, r2 uint32
		}
	}{
		{
			name:   "r1=1, r2=2",
			initR1: 1,
			initR2: 2,
			expectedSeq: []struct{ val, r1, r2 uint32 }{
				{2147483652, 2147483648, 4},
				{1073741832, 1073741824, 8},
				{536870928, 536870912, 16},
				{268435488, 268435456, 32},
				{134217792, 134217728, 64},
				{67108992, 67108864, 128},
				{33554688, 33554432, 256},
				{16777728, 16777216, 512},
				{8389632, 8388608, 1024},
				{4196352, 4194304, 2048},
			},
		},
		{
			name:   "r1=0x12345678, r2=0x87654321",
			initR1: 0x12345678,
			initR2: 0x87654321,
			expectedSeq: []struct{ val, r1, r2 uint32 }{
				{2278600063, 2300193596, 248153667},
				{3642235160, 3297580446, 496307334},
				{3647771586, 3796273871, 992614669},
				{125269884, 1898136935, 1985229339},
				{3560556164, 949068467, 3970458679},
				{3306690870, 474534233, 3645950063},
				{1015412850, 2384750764, 2996932830},
				{2723223018, 3339859030, 1698898364},
				{688871763, 3817413163, 3397796728},
				{3838315492, 1908706581, 2500626161},
			},
		},
		{
			name:   "r1=0, r2=0 (degenerate)",
			initR1: 0,
			initR2: 0,
			expectedSeq: []struct{ val, r1, r2 uint32 }{
				{0, 0, 0},
				{0, 0, 0},
				{0, 0, 0},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r1, r2 := tc.initR1, tc.initR2
			for i, exp := range tc.expectedSeq {
				val := RNG(&r1, &r2)
				if val != exp.val {
					t.Errorf("iteration %d: val=%d, want %d", i, val, exp.val)
				}
				if r1 != exp.r1 {
					t.Errorf("iteration %d: r1=%d, want %d", i, r1, exp.r1)
				}
				if r2 != exp.r2 {
					t.Errorf("iteration %d: r2=%d, want %d", i, r2, exp.r2)
				}
			}
		})
	}
}
