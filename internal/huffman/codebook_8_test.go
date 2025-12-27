// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

func TestHCB8_1Size(t *testing.T) {
	// First-step table must have 32 entries (2^5 = 32)
	if len(hcb8_1) != 32 {
		t.Errorf("hcb8_1 size = %d, want 32", len(hcb8_1))
	}
}

func TestHCB8_2Size(t *testing.T) {
	// Second-step table must have 83 entries (51 + 32 = 83)
	if len(hcb8_2) != 83 {
		t.Errorf("hcb8_2 size = %d, want 83", len(hcb8_2))
	}
}

func TestHCB8_1Values(t *testing.T) {
	// Verify key entries from FAAD2 ~/dev/faad2/libfaad/codebook/hcb_8.h
	tests := []struct {
		idx    int
		offset uint8
		extra  uint8
	}{
		// 3-bit codewords (4 duplicates because we read 5 bits)
		{0, 0, 0}, // 00000 -> offset 0
		{1, 0, 0}, // duplicate
		{2, 0, 0}, // duplicate
		{3, 0, 0}, // duplicate

		// 4-bit codewords (2 duplicates each)
		{4, 1, 0},  // 00100 -> offset 1
		{5, 1, 0},  // duplicate
		{6, 2, 0},  // 00110 -> offset 2
		{7, 2, 0},  // duplicate
		{8, 3, 0},  // 01000 -> offset 3
		{9, 3, 0},  // duplicate
		{10, 4, 0}, // 01010 -> offset 4
		{11, 4, 0}, // duplicate
		{12, 5, 0}, // 01100 -> offset 5
		{13, 5, 0}, // duplicate

		// 5-bit codewords
		{14, 6, 0},  // 01110 -> offset 6
		{15, 7, 0},  // 01111 -> offset 7
		{16, 8, 0},  // 10000 -> offset 8
		{17, 9, 0},  // 10001 -> offset 9
		{18, 10, 0}, // 10010 -> offset 10
		{19, 11, 0}, // 10011 -> offset 11
		{20, 12, 0}, // 10100 -> offset 12

		// 6-bit codewords
		{21, 13, 1}, // 10101 -> offset 13, 1 extra bit
		{22, 15, 1}, // 10110 -> offset 15, 1 extra bit
		{23, 17, 1}, // 10111 -> offset 17, 1 extra bit
		{24, 19, 1}, // 11000 -> offset 19, 1 extra bit
		{25, 21, 1}, // 11001 -> offset 21, 1 extra bit

		// 7-bit codewords
		{26, 23, 2}, // 11010 -> offset 23, 2 extra bits
		{27, 27, 2}, // 11011 -> offset 27, 2 extra bits
		{28, 31, 2}, // 11100 -> offset 31, 2 extra bits

		// 7/8-bit codewords
		{29, 35, 3}, // 11101 -> offset 35, 3 extra bits

		// 8-bit codewords
		{30, 43, 3}, // 11110 -> offset 43, 3 extra bits

		// 8/9/10-bit codewords
		{31, 51, 5}, // 11111 -> offset 51, 5 extra bits
	}
	for _, tt := range tests {
		if hcb8_1[tt.idx].Offset != tt.offset || hcb8_1[tt.idx].ExtraBits != tt.extra {
			t.Errorf("hcb8_1[%d] = {%d, %d}, want {%d, %d}",
				tt.idx, hcb8_1[tt.idx].Offset, hcb8_1[tt.idx].ExtraBits,
				tt.offset, tt.extra)
		}
	}
}

func TestHCB8_2Values(t *testing.T) {
	// Verify key entries from FAAD2 hcb_8.h
	// Codebook 8 is an UNSIGNED PAIR codebook (values 0-7)
	tests := []struct {
		idx  int
		bits uint8
		x, y int8
	}{
		// 3-bit codeword
		{0, 3, 1, 1},

		// 4-bit codewords
		{1, 4, 2, 1},
		{2, 4, 1, 0},
		{3, 4, 1, 2},
		{4, 4, 0, 1},
		{5, 4, 2, 2},

		// 5-bit codewords
		{6, 5, 0, 0},
		{7, 5, 2, 0},
		{8, 5, 0, 2},
		{9, 5, 3, 1},
		{10, 5, 1, 3},
		{11, 5, 3, 2},
		{12, 5, 2, 3},

		// 6-bit codewords
		{13, 6, 3, 3},
		{14, 6, 4, 1},
		{15, 6, 1, 4},
		{16, 6, 4, 2},
		{17, 6, 2, 4},
		{18, 6, 3, 0},
		{19, 6, 0, 3},
		{20, 6, 4, 3},
		{21, 6, 3, 4},
		{22, 6, 5, 2},

		// 7-bit codewords
		{23, 7, 5, 1},
		{24, 7, 2, 5},
		{25, 7, 1, 5},
		{26, 7, 5, 3},
		{27, 7, 3, 5},
		{28, 7, 4, 4},
		{29, 7, 5, 4},
		{30, 7, 0, 4},
		{31, 7, 4, 5},
		{32, 7, 4, 0},
		{33, 7, 2, 6},
		{34, 7, 6, 2},

		// 7/8-bit codewords (duplicated 7-bit entries)
		{35, 7, 6, 1},
		{36, 7, 6, 1}, // duplicate
		{37, 7, 1, 6},
		{38, 7, 1, 6}, // duplicate
		{39, 8, 3, 6},
		{40, 8, 6, 3},
		{41, 8, 5, 5},
		{42, 8, 5, 0},

		// 8-bit codewords
		{43, 8, 6, 4},
		{44, 8, 0, 5},
		{45, 8, 4, 6},
		{46, 8, 7, 1},
		{47, 8, 7, 2},
		{48, 8, 2, 7},
		{49, 8, 6, 5},
		{50, 8, 7, 3},

		// 8/9/10-bit codewords
		{51, 8, 1, 7},
		{52, 8, 1, 7}, // duplicate
		{53, 8, 1, 7}, // duplicate
		{54, 8, 1, 7}, // duplicate
		{55, 8, 5, 6},
		{56, 8, 5, 6}, // duplicate
		{57, 8, 5, 6}, // duplicate
		{58, 8, 5, 6}, // duplicate
		{59, 8, 3, 7},
		{60, 8, 3, 7}, // duplicate
		{61, 8, 3, 7}, // duplicate
		{62, 8, 3, 7}, // duplicate
		{63, 9, 6, 6},
		{64, 9, 6, 6}, // duplicate
		{65, 9, 7, 4},
		{66, 9, 7, 4}, // duplicate
		{67, 9, 6, 0},
		{68, 9, 6, 0}, // duplicate
		{69, 9, 4, 7},
		{70, 9, 4, 7}, // duplicate
		{71, 9, 0, 6},
		{72, 9, 0, 6}, // duplicate
		{73, 9, 7, 5},
		{74, 9, 7, 5}, // duplicate
		{75, 9, 7, 6},
		{76, 9, 7, 6}, // duplicate
		{77, 9, 6, 7},
		{78, 9, 6, 7}, // duplicate
		{79, 10, 5, 7},
		{80, 10, 7, 0},
		{81, 10, 0, 7},
		{82, 10, 7, 7},
	}
	for _, tt := range tests {
		e := hcb8_2[tt.idx]
		if e.Bits != tt.bits || e.X != tt.x || e.Y != tt.y {
			t.Errorf("hcb8_2[%d] = {%d, %d, %d}, want {%d, %d, %d}",
				tt.idx, e.Bits, e.X, e.Y,
				tt.bits, tt.x, tt.y)
		}
	}
}

func TestHCB8_2UnsignedValues(t *testing.T) {
	// Codebook 8 is an unsigned pair codebook with values from 0 to 7
	// All values should be non-negative
	for idx, e := range hcb8_2 {
		if e.X < 0 || e.Y < 0 {
			t.Errorf("hcb8_2[%d] has negative value: x=%d, y=%d (codebook 8 is unsigned)",
				idx, e.X, e.Y)
		}
		if e.X > 7 || e.Y > 7 {
			t.Errorf("hcb8_2[%d] has value > 7: x=%d, y=%d (codebook 8 max is 7)",
				idx, e.X, e.Y)
		}
	}
}

func TestHCB8IsPairCodebook(t *testing.T) {
	// Verify that codebook 8 uses HCB2Pair (pair codebook with x, y only)
	e := hcb8_2[0]
	// Access X and Y to verify they exist (compile-time check)
	_ = e.X
	_ = e.Y
	// HCB2Pair has no V or W fields - this is a compile-time verification
}

func TestHCB8_1ExtraBitsProgression(t *testing.T) {
	// Verify that extra bits are correctly assigned based on codeword lengths
	// Entries with 0 extra bits are complete codewords
	// Entries with 1+ extra bits need additional lookups

	// Check that 0 extra bits entries are at the beginning (shorter codewords)
	for i := 0; i <= 20; i++ {
		if hcb8_1[i].ExtraBits != 0 {
			t.Errorf("hcb8_1[%d].ExtraBits = %d, expected 0 for short codewords",
				i, hcb8_1[i].ExtraBits)
		}
	}

	// Check that 1-5 extra bits entries are at the end (longer codewords)
	// Index 21-25: 1 extra bit (6-bit codewords)
	for i := 21; i <= 25; i++ {
		if hcb8_1[i].ExtraBits != 1 {
			t.Errorf("hcb8_1[%d].ExtraBits = %d, expected 1", i, hcb8_1[i].ExtraBits)
		}
	}

	// Index 26-28: 2 extra bits (7-bit codewords)
	for i := 26; i <= 28; i++ {
		if hcb8_1[i].ExtraBits != 2 {
			t.Errorf("hcb8_1[%d].ExtraBits = %d, expected 2", i, hcb8_1[i].ExtraBits)
		}
	}

	// Index 29: 3 extra bits (7/8-bit codewords)
	if hcb8_1[29].ExtraBits != 3 {
		t.Errorf("hcb8_1[29].ExtraBits = %d, expected 3", hcb8_1[29].ExtraBits)
	}

	// Index 30: 3 extra bits (8-bit codewords)
	if hcb8_1[30].ExtraBits != 3 {
		t.Errorf("hcb8_1[30].ExtraBits = %d, expected 3", hcb8_1[30].ExtraBits)
	}

	// Index 31: 5 extra bits (8/9/10-bit codewords)
	if hcb8_1[31].ExtraBits != 5 {
		t.Errorf("hcb8_1[31].ExtraBits = %d, expected 5", hcb8_1[31].ExtraBits)
	}
}
