// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

func TestHCB6_1Size(t *testing.T) {
	// First-step table must have 32 entries (2^5 = 32)
	if len(hcb6_1) != 32 {
		t.Errorf("hcb6_1 size = %d, want 32", len(hcb6_1))
	}
}

func TestHCB6_2Size(t *testing.T) {
	// Second-step table must have 125 entries (61 + 64 = 125)
	if len(hcb6_2) != 125 {
		t.Errorf("hcb6_2 size = %d, want 125", len(hcb6_2))
	}
}

func TestHCB6_1Values(t *testing.T) {
	// Verify key entries from FAAD2 ~/dev/faad2/libfaad/codebook/hcb_6.h
	tests := []struct {
		idx    int
		offset uint8
		extra  uint8
	}{
		// 4-bit codewords (duplicated entries)
		{0, 0, 0},  // 00000 -> offset 0
		{1, 0, 0},  // duplicate
		{2, 1, 0},  // 00010 -> offset 1
		{3, 1, 0},  // duplicate
		{16, 8, 0}, // 10000 -> offset 8
		{17, 8, 0}, // duplicate

		// 6-bit codewords
		{18, 9, 1},  // 10010 -> offset 9, 1 extra bit
		{19, 11, 1}, // 10011 -> offset 11, 1 extra bit
		{24, 21, 1}, // 11000 -> offset 21, 1 extra bit
		{25, 23, 1}, // 11001 -> offset 23, 1 extra bit

		// 7-bit codewords
		{26, 25, 2}, // 11010 -> offset 25, 2 extra bits
		{27, 29, 2}, // 11011 -> offset 29, 2 extra bits
		{28, 33, 2}, // 11100 -> offset 33, 2 extra bits

		// 7/8-bit codewords
		{29, 37, 3}, // 11101 -> offset 37, 3 extra bits

		// 8/9-bit codewords
		{30, 45, 4}, // 11110 -> offset 45, 4 extra bits

		// 9/10/11-bit codewords
		{31, 61, 6}, // 11111 -> offset 61, 6 extra bits
	}
	for _, tt := range tests {
		if hcb6_1[tt.idx].Offset != tt.offset || hcb6_1[tt.idx].ExtraBits != tt.extra {
			t.Errorf("hcb6_1[%d] = {%d, %d}, want {%d, %d}",
				tt.idx, hcb6_1[tt.idx].Offset, hcb6_1[tt.idx].ExtraBits,
				tt.offset, tt.extra)
		}
	}
}

func TestHCB6_2Values(t *testing.T) {
	// Verify key entries from FAAD2 hcb_6.h
	// Codebook 6 is a signed PAIR codebook (only x, y - no v, w)
	tests := []struct {
		idx  int
		bits uint8
		x, y int8
	}{
		// 4-bit codewords
		{0, 4, 0, 0},
		{1, 4, 1, 0},
		{2, 4, 0, -1},
		{3, 4, 0, 1},
		{4, 4, -1, 0},
		{5, 4, 1, 1},
		{6, 4, -1, 1},
		{7, 4, 1, -1},
		{8, 4, -1, -1},

		// 6-bit codewords
		{9, 6, 2, -1},
		{10, 6, 2, 1},
		{14, 6, -1, 2},
		{24, 6, 2, 2},

		// 7-bit codewords
		{25, 7, -3, 1},
		{26, 7, 3, 1},
		{36, 7, 0, 3},

		// 7/8-bit codewords (duplicated 7-bit entries and 8-bit entries)
		{37, 7, 3, 2},
		{38, 7, 3, 2}, // duplicate
		{39, 8, -3, -2},
		{44, 8, -2, -3},

		// 8/9-bit codewords
		{45, 8, -3, 2},
		{46, 8, -3, 2}, // duplicate
		{47, 8, 3, 3},
		{48, 8, 3, 3}, // duplicate
		{49, 9, 3, -3},
		{60, 9, 0, -4},

		// 9/10/11-bit codewords
		{61, 9, -4, 2},
		{62, 9, -4, 2},    // duplicate
		{63, 9, -4, 2},    // duplicate
		{64, 9, -4, 2},    // duplicate
		{105, 10, -3, -4}, // first 10-bit entry
		{106, 10, -3, -4}, // duplicate
		{107, 10, -3, 4},
		{108, 10, -3, 4}, // duplicate
		{121, 11, 4, 4},
		{122, 11, -4, 4},
		{123, 11, -4, -4},
		{124, 11, 4, -4},
	}
	for _, tt := range tests {
		e := hcb6_2[tt.idx]
		if e.Bits != tt.bits || e.X != tt.x || e.Y != tt.y {
			t.Errorf("hcb6_2[%d] = {%d, %d, %d}, want {%d, %d, %d}",
				tt.idx, e.Bits, e.X, e.Y,
				tt.bits, tt.x, tt.y)
		}
	}
}

func TestHCB6_2SignedValues(t *testing.T) {
	// Codebook 6 is a signed pair codebook with values from -4 to 4
	// Verify that negative values are correctly stored
	signedTests := []struct {
		idx  int
		x, y int8
	}{
		{2, 0, -1},    // negative y
		{4, -1, 0},    // negative x
		{6, -1, 1},    // mixed
		{7, 1, -1},    // mixed
		{8, -1, -1},   // both negative
		{11, -2, 1},   // -2
		{25, -3, 1},   // -3
		{123, -4, -4}, // maximum negative (index 123 in FAAD2)
	}
	for _, tt := range signedTests {
		e := hcb6_2[tt.idx]
		if e.X != tt.x || e.Y != tt.y {
			t.Errorf("hcb6_2[%d] = {x=%d, y=%d}, want {x=%d, y=%d}",
				tt.idx, e.X, e.Y, tt.x, tt.y)
		}
	}
}

func TestHCB6IsPairCodebook(t *testing.T) {
	// Verify that codebook 6 uses HCB2Pair (pair codebook with x, y only)
	// This is the first 2-step PAIR codebook (unlike 1,2,4 which are Quad)
	e := hcb6_2[0]
	// Access X and Y to verify they exist (compile-time check)
	_ = e.X
	_ = e.Y
	// HCB2Pair has no V or W fields - this is a compile-time verification
}
