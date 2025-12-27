// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

func TestHCB11_1Size(t *testing.T) {
	// First-step table must have 32 entries (2^5 bits)
	if len(hcb11_1) != 32 {
		t.Errorf("hcb11_1 size = %d, want 32", len(hcb11_1))
	}
}

func TestHCB11_2Size(t *testing.T) {
	// Second-step table must have 374 entries (246 + 128 = 374)
	// From hcb_11.h: "Size of second level table is 246 + 128 = 374"
	if len(hcb11_2) != 374 {
		t.Errorf("hcb11_2 size = %d, want 374", len(hcb11_2))
	}
}

func TestHCB11_1Values(t *testing.T) {
	// Verify key entries from FAAD2 ~/dev/faad2/libfaad/codebook/hcb_11.h
	tests := []struct {
		idx    int
		offset uint8
		extra  uint8
	}{
		// 4 bit codewords (indices 0-3)
		{0, 0, 0}, // 00000 -> offset 0
		{1, 0, 0}, // duplicate
		{2, 1, 0}, // 00010 -> offset 1
		{3, 1, 0}, // duplicate
		// 5 bit codewords (indices 4-9)
		{4, 2, 0}, // 00100 -> offset 2
		{5, 3, 0}, // 00101 -> offset 3
		{6, 4, 0}, // 00110 -> offset 4
		{7, 5, 0}, // 00111 -> offset 5
		{8, 6, 0}, // 01000 -> offset 6
		{9, 7, 0}, // 01001 -> offset 7
		// 6 bit codewords (indices 10-12)
		{10, 8, 1},  // 01010 -> offset 8, 1 extra bit
		{11, 10, 1}, // 01011 -> offset 10, 1 extra bit
		{12, 12, 1}, // 01100 -> offset 12, 1 extra bit
		// 6/7 bit codewords (index 13)
		{13, 14, 2}, // 01101 -> offset 14, 2 extra bits
		// 7 bit codewords (indices 14-16)
		{14, 18, 2}, // 01110 -> offset 18, 2 extra bits
		{15, 22, 2}, // 01111 -> offset 22, 2 extra bits
		{16, 26, 2}, // 10000 -> offset 26, 2 extra bits
		// 7/8 bit codewords (index 17)
		{17, 30, 3}, // 10001 -> offset 30, 3 extra bits
		// 8 bit codewords (indices 18-23)
		{18, 38, 3}, // 10010 -> offset 38
		{19, 46, 3}, // 10011 -> offset 46
		{20, 54, 3}, // 10100 -> offset 54
		{21, 62, 3}, // 10101 -> offset 62
		{22, 70, 3}, // 10110 -> offset 70
		{23, 78, 3}, // 10111 -> offset 78
		// 8/9 bit codewords (index 24)
		{24, 86, 4}, // 11000 -> offset 86, 4 extra bits
		// 9 bit codewords (indices 25-27)
		{25, 102, 4}, // 11001 -> offset 102
		{26, 118, 4}, // 11010 -> offset 118
		{27, 134, 4}, // 11011 -> offset 134
		// 9/10 bit codewords (index 28)
		{28, 150, 5}, // 11100 -> offset 150, 5 extra bits
		// 10 bit codewords (indices 29-30)
		{29, 182, 5}, // 11101 -> offset 182
		{30, 214, 5}, // 11110 -> offset 214
		// 10/11/12 bit codewords (index 31)
		{31, 246, 7}, // 11111 -> offset 246, 7 extra bits
	}
	for _, tt := range tests {
		if hcb11_1[tt.idx].Offset != tt.offset || hcb11_1[tt.idx].ExtraBits != tt.extra {
			t.Errorf("hcb11_1[%d] = {%d, %d}, want {%d, %d}",
				tt.idx, hcb11_1[tt.idx].Offset, hcb11_1[tt.idx].ExtraBits,
				tt.offset, tt.extra)
		}
	}
}

func TestHCB11_1LastEntry(t *testing.T) {
	// Last entry: 10/11/12 bit codewords
	// { /* 11111 */ 246, 7 }
	if hcb11_1[31].Offset != 246 || hcb11_1[31].ExtraBits != 7 {
		t.Errorf("hcb11_1[31] = {%d, %d}, want {246, 7}",
			hcb11_1[31].Offset, hcb11_1[31].ExtraBits)
	}
}

func TestHCB11_2Values(t *testing.T) {
	// Verify key entries from FAAD2 ~/dev/faad2/libfaad/codebook/hcb_11.h
	tests := []struct {
		idx  int
		bits uint8
		x, y int8
	}{
		// 4 bit codewords (indices 0-1)
		{0, 4, 0, 0},
		{1, 4, 1, 1},
		// 5 bit codewords (indices 2-7)
		{2, 5, 16, 16}, // escape code pair
		{3, 5, 1, 0},
		{4, 5, 0, 1},
		{5, 5, 2, 1},
		{6, 5, 1, 2},
		{7, 5, 2, 2},
		// 6 bit codewords (indices 8-13)
		{8, 6, 1, 3},
		{9, 6, 3, 1},
		{10, 6, 3, 2},
		{11, 6, 2, 0},
		{12, 6, 2, 3},
		{13, 6, 0, 2},
		// 6/7 bit codewords (indices 14-17)
		{14, 6, 3, 3}, // duplicated
		{15, 6, 3, 3}, // duplicated
		{16, 7, 4, 1},
		{17, 7, 1, 4},
		// 7 bit codewords (indices 18-29)
		{18, 7, 4, 2},
		{19, 7, 2, 4},
		{29, 7, 5, 3},
		// 7/8 bit codewords (indices 30-37)
		{30, 7, 3, 5}, // duplicated
		{31, 7, 3, 5}, // duplicated
		{32, 7, 5, 4}, // duplicated
		{33, 7, 5, 4}, // duplicated
		{34, 8, 4, 5},
		{35, 8, 6, 2},
		{36, 8, 2, 6},
		{37, 8, 6, 1},
		// 8 bit codewords - spot check
		{38, 8, 6, 3},
		{41, 8, 4, 16}, // escape value
		{42, 8, 3, 16}, // escape value
		{43, 8, 16, 5}, // escape value
		{44, 8, 16, 3}, // escape value
		{53, 8, 5, 16}, // escape value
		{85, 8, 5, 0},
		// 8/9 bit codewords
		{86, 8, 16, 14}, // escape value, duplicated
		{87, 8, 16, 14}, // duplicated
		{100, 9, 8, 4},
		{101, 9, 16, 15}, // escape value
		// 9 bit codewords
		{102, 9, 12, 16}, // escape value
		{103, 9, 1, 8},
		{149, 9, 6, 10},
		// 9/10 bit codewords
		{150, 9, 13, 3}, // duplicated
		{151, 9, 13, 3}, // duplicated
		{160, 10, 11, 4},
		{161, 10, 9, 8},
		// 10 bit codewords
		{182, 10, 11, 1},
		{183, 10, 3, 12},
		{245, 10, 0, 9},
		// 10/11/12 bit codewords
		{246, 10, 9, 13}, // duplicated 4x
		{249, 10, 9, 13}, // duplicated
		{282, 11, 9, 14}, // duplicated 2x
		{283, 11, 9, 14}, // duplicated
		// 12 bit codewords
		{368, 12, 0, 14},
		{369, 12, 0, 12},
		{370, 12, 15, 14},
		{371, 12, 15, 0},
		{372, 12, 0, 15},
		{373, 12, 15, 15}, // last entry
	}
	for _, tt := range tests {
		e := hcb11_2[tt.idx]
		if e.Bits != tt.bits || e.X != tt.x || e.Y != tt.y {
			t.Errorf("hcb11_2[%d] = {%d, %d, %d}, want {%d, %d, %d}",
				tt.idx, e.Bits, e.X, e.Y,
				tt.bits, tt.x, tt.y)
		}
	}
}

func TestHCB11EscapeValues(t *testing.T) {
	// Codebook 11 uses escape codes (16) for values >= 16
	// Verify that escape values (16) are present
	foundEscape := false
	for _, e := range hcb11_2 {
		if e.X == 16 || e.Y == 16 {
			foundEscape = true
			break
		}
	}
	if !foundEscape {
		t.Error("hcb11_2 should contain entries with escape value (16)")
	}
}

func TestHCB11ValueRange(t *testing.T) {
	// Codebook 11 is an unsigned codebook with values 0-16
	// (16 is escape code, actual values 0-15 plus escape)
	for i, e := range hcb11_2 {
		if e.X < 0 || e.X > 16 {
			t.Errorf("hcb11_2[%d].X = %d, should be in range 0-16", i, e.X)
		}
		if e.Y < 0 || e.Y > 16 {
			t.Errorf("hcb11_2[%d].Y = %d, should be in range 0-16", i, e.Y)
		}
	}
}

func TestHCB11MaxValue(t *testing.T) {
	// Verify maximum non-escape value (15) and escape value (16) are present
	found15 := false
	found16 := false
	for _, e := range hcb11_2 {
		if e.X == 15 || e.Y == 15 {
			found15 = true
		}
		if e.X == 16 || e.Y == 16 {
			found16 = true
		}
		if found15 && found16 {
			break
		}
	}
	if !found15 {
		t.Error("hcb11_2 should contain entry with value 15")
	}
	if !found16 {
		t.Error("hcb11_2 should contain entry with escape value 16")
	}
}

func TestHCB11FirstStepBits(t *testing.T) {
	// Verify that the first-step table uses 5 bits (32 entries)
	expectedSize := 32 // 2^5
	if len(hcb11_1) != expectedSize {
		t.Errorf("hcb11_1 should have %d entries for 5-bit lookup, got %d",
			expectedSize, len(hcb11_1))
	}
}

func TestHCB11SecondStepTableConsistency(t *testing.T) {
	// Verify that the calculated size matches: 246 + 128 = 374
	// The last entry of hcb11_1 is {246, 7}, meaning offset 246 with 7 extra bits
	// 2^7 = 128 entries, so 246 + 128 = 374
	expectedSize := 246 + 128
	if len(hcb11_2) != expectedSize {
		t.Errorf("hcb11_2 should have %d entries (246 + 128), got %d",
			expectedSize, len(hcb11_2))
	}
}

func TestHCB11LastEntryMaxBitLength(t *testing.T) {
	// The last entries use 12-bit codewords
	lastIdx := len(hcb11_2) - 1
	if hcb11_2[lastIdx].Bits != 12 {
		t.Errorf("hcb11_2[%d].Bits = %d, want 12 (maximum codeword length)",
			lastIdx, hcb11_2[lastIdx].Bits)
	}
}
