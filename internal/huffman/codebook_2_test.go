// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

func TestHCB2_1Size(t *testing.T) {
	// First-step table must have 32 entries (2^5 bits)
	if len(hcb2_1) != 32 {
		t.Errorf("hcb2_1 size = %d, want 32", len(hcb2_1))
	}
}

func TestHCB2_2Size(t *testing.T) {
	// Second-step table must have 85 entries
	if len(hcb2_2) != 85 {
		t.Errorf("hcb2_2 size = %d, want 85", len(hcb2_2))
	}
}

func TestHCB2_1Values(t *testing.T) {
	// Verify key entries from FAAD2 hcb_2.h
	tests := []struct {
		idx    int
		offset uint8
		extra  uint8
	}{
		{0, 0, 0},   // 3-bit codeword maps to index 0
		{4, 1, 0},   // 4-bit: 00100 -> offset 1
		{6, 2, 0},   // 5-bit: 00110 -> offset 2
		{13, 9, 1},  // 6-bit: 01101 -> offset 9, 1 extra bit
		{24, 31, 1}, // 6-bit: 11000 -> offset 31, 1 extra bit
		{25, 33, 2}, // 7-bit: 11001 -> offset 33, 2 extra bits
		{28, 45, 3}, // 7/8-bit: 11100 -> offset 45, 3 extra bits
		{31, 69, 4}, // 8/9-bit: 11111 -> offset 69, 4 extra bits
	}
	for _, tt := range tests {
		if hcb2_1[tt.idx].Offset != tt.offset || hcb2_1[tt.idx].ExtraBits != tt.extra {
			t.Errorf("hcb2_1[%d] = {%d, %d}, want {%d, %d}",
				tt.idx, hcb2_1[tt.idx].Offset, hcb2_1[tt.idx].ExtraBits,
				tt.offset, tt.extra)
		}
	}
}

func TestHCB2_2Values(t *testing.T) {
	// Verify key entries from FAAD2 hcb_2.h
	tests := []struct {
		idx        int
		bits       uint8
		x, y, v, w int8
	}{
		{0, 3, 0, 0, 0, 0},    // 3-bit codeword: all zeros
		{1, 4, 1, 0, 0, 0},    // 4-bit: (1,0,0,0)
		{2, 5, -1, 0, 0, 0},   // 5-bit: (-1,0,0,0)
		{9, 6, 0, -1, 1, 0},   // 6-bit: (0,-1,1,0)
		{33, 7, 0, 1, -1, 1},  // 7-bit: (0,1,-1,1)
		{51, 8, 1, -1, 0, 1},  // 8-bit: (1,-1,0,1)
		{52, 8, -1, 1, 0, -1}, // 8-bit: (-1,1,0,-1)
		{69, 8, -1, 1, -1, 1}, // 8/9-bit boundary: 8-bit entry
		{70, 8, -1, 1, -1, 1}, // duplicate for 8-bit
		{71, 9, 1, -1, -1, 1}, // 9-bit entry
		{84, 9, 1, 1, 1, -1},  // last entry: 9-bit
	}
	for _, tt := range tests {
		e := hcb2_2[tt.idx]
		if e.Bits != tt.bits || e.X != tt.x || e.Y != tt.y || e.V != tt.v || e.W != tt.w {
			t.Errorf("hcb2_2[%d] = {%d, %d, %d, %d, %d}, want {%d, %d, %d, %d, %d}",
				tt.idx, e.Bits, e.X, e.Y, e.V, e.W,
				tt.bits, tt.x, tt.y, tt.v, tt.w)
		}
	}
}
