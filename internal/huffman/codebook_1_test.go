// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

func TestHCB1_1Size(t *testing.T) {
	// First-step table must have 32 entries (2^5 bits)
	if len(hcb1_1) != 32 {
		t.Errorf("hcb1_1 size = %d, want 32", len(hcb1_1))
	}
}

func TestHCB1_2Size(t *testing.T) {
	// Second-step table must have 113 entries
	if len(hcb1_2) != 113 {
		t.Errorf("hcb1_2 size = %d, want 113", len(hcb1_2))
	}
}

func TestHCB1_1Values(t *testing.T) {
	// Verify key entries from FAAD2
	tests := []struct {
		idx    int
		offset uint8
		extra  uint8
	}{
		{0, 0, 0},   // 1-bit codeword maps to index 0
		{16, 1, 0},  // 5-bit: 10000 -> offset 1
		{24, 9, 2},  // 7-bit: 11000 -> offset 9, 2 extra bits
		{30, 33, 4}, // 9-bit: 11110 -> offset 33, 4 extra bits
		{31, 49, 6}, // 9/10/11-bit: 11111 -> offset 49, 6 extra bits
	}
	for _, tt := range tests {
		if hcb1_1[tt.idx].Offset != tt.offset || hcb1_1[tt.idx].ExtraBits != tt.extra {
			t.Errorf("hcb1_1[%d] = {%d, %d}, want {%d, %d}",
				tt.idx, hcb1_1[tt.idx].Offset, hcb1_1[tt.idx].ExtraBits,
				tt.offset, tt.extra)
		}
	}
}

func TestHCB1_2Values(t *testing.T) {
	// Verify key entries from FAAD2
	tests := []struct {
		idx        int
		bits       uint8
		x, y, v, w int8
	}{
		{0, 1, 0, 0, 0, 0},     // 1-bit codeword: all zeros
		{1, 5, 1, 0, 0, 0},     // 5-bit: (1,0,0,0)
		{9, 7, 1, -1, 0, 0},    // 7-bit: (1,-1,0,0)
		{112, 11, 1, 1, 1, -1}, // 11-bit: last entry
	}
	for _, tt := range tests {
		e := hcb1_2[tt.idx]
		if e.Bits != tt.bits || e.X != tt.x || e.Y != tt.y || e.V != tt.v || e.W != tt.w {
			t.Errorf("hcb1_2[%d] = {%d, %d, %d, %d, %d}, want {%d, %d, %d, %d, %d}",
				tt.idx, e.Bits, e.X, e.Y, e.V, e.W,
				tt.bits, tt.x, tt.y, tt.v, tt.w)
		}
	}
}
