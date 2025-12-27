// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

func TestHCB10_1Size(t *testing.T) {
	// First-step table must have 64 entries (2^6 bits)
	// NOTE: This is the only 2-step codebook with 6-bit first step
	if len(hcb10_1) != 64 {
		t.Errorf("hcb10_1 size = %d, want 64", len(hcb10_1))
	}
}

func TestHCB10_2Size(t *testing.T) {
	// Second-step table must have 209 entries (145 + 64 = 209)
	if len(hcb10_2) != 209 {
		t.Errorf("hcb10_2 size = %d, want 209", len(hcb10_2))
	}
}

func TestHCB10_1Values(t *testing.T) {
	// Verify key entries from FAAD2 ~/dev/faad2/libfaad/codebook/hcb_10.h
	tests := []struct {
		idx    int
		offset uint8
		extra  uint8
	}{
		// 4 bit codewords (indices 0-11, all map to offsets 0-2 with 0 extra bits)
		{0, 0, 0}, // 000000 -> offset 0
		{4, 1, 0}, // 000100 -> offset 1
		{8, 2, 0}, // 001000 -> offset 2
		// 5 bit codewords
		{12, 3, 0},  // 001100 -> offset 3
		{14, 4, 0},  // 001110 -> offset 4
		{16, 5, 0},  // 010000 -> offset 5
		{26, 10, 0}, // 011010 -> offset 10
		// 6 bit codewords
		{28, 11, 0}, // 011100 -> offset 11
		{29, 12, 0}, // 011101 -> offset 12
		{41, 24, 0}, // 101001 -> offset 24
		// 7 bit codewords
		{42, 25, 1}, // 101010 -> offset 25, 1 extra bit
		{43, 27, 1}, // 101011 -> offset 27, 1 extra bit
		{49, 39, 1}, // 110001 -> offset 39, 1 extra bit
		// 7/8 bit codewords
		{50, 41, 2}, // 110010 -> offset 41, 2 extra bits
		// 8 bit codewords
		{51, 45, 2}, // 110011 -> offset 45, 2 extra bits
		{55, 61, 2}, // 110111 -> offset 61, 2 extra bits (index 55 = binary 110111)
		// 8/9 bit codewords
		{56, 65, 3}, // 111000 -> offset 65, 3 extra bits (index 56 = binary 111000)
		// 9 bit codewords
		{57, 73, 3}, // 111001 -> offset 73, 3 extra bits
		{59, 89, 3}, // 111011 -> offset 89, 3 extra bits (index 59 = binary 111011)
		// 9/10 bit codewords
		{60, 97, 4}, // 111100 -> offset 97, 4 extra bits
		// 10 bit codewords
		{61, 113, 4}, // 111101 -> offset 113, 4 extra bits
		{62, 129, 4}, // 111110 -> offset 129, 4 extra bits
		// 10/11/12 bit codewords (last entry - tested separately)
	}
	for _, tt := range tests {
		if hcb10_1[tt.idx].Offset != tt.offset || hcb10_1[tt.idx].ExtraBits != tt.extra {
			t.Errorf("hcb10_1[%d] = {%d, %d}, want {%d, %d}",
				tt.idx, hcb10_1[tt.idx].Offset, hcb10_1[tt.idx].ExtraBits,
				tt.offset, tt.extra)
		}
	}
}

func TestHCB10_1LastEntry(t *testing.T) {
	// Last entry: 10/11/12 bit codewords
	// { /* 111111 */ 145, 6 }
	if hcb10_1[63].Offset != 145 || hcb10_1[63].ExtraBits != 6 {
		t.Errorf("hcb10_1[63] = {%d, %d}, want {145, 6}",
			hcb10_1[63].Offset, hcb10_1[63].ExtraBits)
	}
}

func TestHCB10_2Values(t *testing.T) {
	// Verify key entries from FAAD2 ~/dev/faad2/libfaad/codebook/hcb_10.h
	// Count entries from the C source file to get correct indices
	tests := []struct {
		idx  int
		bits uint8
		x, y int8
	}{
		// 4 bit codewords (indices 0-2)
		{0, 4, 1, 1},
		{1, 4, 1, 2},
		{2, 4, 2, 1},
		// 5 bit codewords (indices 3-10)
		{3, 5, 2, 2},
		{4, 5, 1, 0},
		{5, 5, 0, 1},
		{10, 5, 3, 3},
		// 6 bit codewords (indices 11-24)
		{11, 6, 2, 0},
		{12, 6, 0, 2},
		{24, 6, 5, 2},
		// 7 bit codewords (indices 25-40)
		{25, 7, 1, 5},
		{26, 7, 5, 1},
		{40, 7, 6, 4},
		// 7/8 bit codewords (indices 41-44)
		{41, 7, 4, 6},
		{42, 7, 4, 6}, // duplicate for 7-bit
		{43, 8, 6, 5},
		{44, 8, 7, 2},
		// 8 bit codewords (indices 45-64)
		{45, 8, 3, 7},
		{64, 8, 4, 8},
		// 8/9 bit codewords (indices 65-72)
		{65, 8, 5, 7},
		{66, 8, 5, 7}, // 8-bit duplicated
		{71, 9, 7, 6},
		{72, 9, 6, 7},
		// 9 bit codewords (indices 73-96)
		{73, 9, 9, 2},
		{96, 9, 9, 6},
		// 9/10 bit codewords (indices 97-112)
		{97, 9, 6, 9},
		{98, 9, 6, 9},
		{107, 10, 7, 9},
		{112, 10, 9, 7},
		// 10 bit codewords (indices 113-144)
		{113, 10, 0, 7},
		{144, 10, 9, 9},
		// 10/11/12 bit codewords (indices 145-208)
		{145, 10, 12, 6},
		{146, 10, 12, 6},
		{147, 10, 12, 6},
		{148, 10, 12, 6},
		// 11 bit codewords (duplicated twice)
		{157, 11, 9, 10},
		{158, 11, 9, 10},
		// 12 bit codewords
		{201, 12, 12, 9},
		{208, 12, 12, 12}, // last entry
	}
	for _, tt := range tests {
		e := hcb10_2[tt.idx]
		if e.Bits != tt.bits || e.X != tt.x || e.Y != tt.y {
			t.Errorf("hcb10_2[%d] = {%d, %d, %d}, want {%d, %d, %d}",
				tt.idx, e.Bits, e.X, e.Y,
				tt.bits, tt.x, tt.y)
		}
	}
}

func TestHCB10UnsignedValues(t *testing.T) {
	// Codebook 10 is an unsigned codebook with values from 0 to 12
	// Verify that all values are in the valid range
	for i, e := range hcb10_2 {
		if e.X < 0 || e.X > 12 {
			t.Errorf("hcb10_2[%d].X = %d, should be in range 0-12", i, e.X)
		}
		if e.Y < 0 || e.Y > 12 {
			t.Errorf("hcb10_2[%d].Y = %d, should be in range 0-12", i, e.Y)
		}
	}
}

func TestHCB10MaxValue(t *testing.T) {
	// Codebook 10 supports values 0-12, verify maximum values are present
	foundMax := false
	for _, e := range hcb10_2 {
		if e.X == 12 && e.Y == 12 {
			foundMax = true
			break
		}
	}
	if !foundMax {
		t.Error("hcb10_2 should contain entry with maximum values (12, 12)")
	}
}

func TestHCB10FirstStepBits(t *testing.T) {
	// Verify that the first-step table uses 6 bits (64 entries)
	// This is the only 2-step codebook with 6-bit first step
	expectedSize := 64 // 2^6
	if len(hcb10_1) != expectedSize {
		t.Errorf("hcb10_1 should have %d entries for 6-bit lookup, got %d",
			expectedSize, len(hcb10_1))
	}
}

func TestHCB10SecondStepTableConsistency(t *testing.T) {
	// Verify that the calculated size matches: 145 + 64 = 209
	// The last entry of hcb10_1 is {145, 6}, meaning offset 145 with 6 extra bits
	// 2^6 = 64 entries, so 145 + 64 = 209
	expectedSize := 145 + 64
	if len(hcb10_2) != expectedSize {
		t.Errorf("hcb10_2 should have %d entries (145 + 64), got %d",
			expectedSize, len(hcb10_2))
	}
}
