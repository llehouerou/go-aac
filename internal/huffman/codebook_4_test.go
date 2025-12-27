// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

func TestHCB4_1Size(t *testing.T) {
	// First-step table must have 32 entries (2^5 bits)
	if len(hcb4_1) != 32 {
		t.Errorf("hcb4_1 size = %d, want 32", len(hcb4_1))
	}
}

func TestHCB4_2Size(t *testing.T) {
	// Second-step table must have 184 entries (56 + 128)
	if len(hcb4_2) != 184 {
		t.Errorf("hcb4_2 size = %d, want 184", len(hcb4_2))
	}
}

func TestHCB4_1Values(t *testing.T) {
	// Verify key entries from FAAD2
	tests := []struct {
		idx    int
		offset uint8
		extra  uint8
	}{
		{0, 0, 0},   // 4-bit: 00000 -> offset 0
		{2, 1, 0},   // 4-bit: 00010 -> offset 1
		{20, 10, 0}, // 5-bit: 10100 -> offset 10
		{26, 16, 2}, // 7-bit: 11010 -> offset 16, 2 extra bits
		{27, 20, 2}, // 7-bit: 11011 -> offset 20, 2 extra bits
		{28, 24, 3}, // 7/8-bit: 11100 -> offset 24, 3 extra bits
		{29, 32, 3}, // 8-bit: 11101 -> offset 32, 3 extra bits
		{30, 40, 4}, // 8/9-bit: 11110 -> offset 40, 4 extra bits
		{31, 56, 7}, // 9/10/11/12-bit: 11111 -> offset 56, 7 extra bits
	}
	for _, tt := range tests {
		if hcb4_1[tt.idx].Offset != tt.offset || hcb4_1[tt.idx].ExtraBits != tt.extra {
			t.Errorf("hcb4_1[%d] = {%d, %d}, want {%d, %d}",
				tt.idx, hcb4_1[tt.idx].Offset, hcb4_1[tt.idx].ExtraBits,
				tt.offset, tt.extra)
		}
	}
}

func TestHCB4_2Values(t *testing.T) {
	// Verify key entries from FAAD2
	// Note: Codebook 4 is UNSIGNED (unlike codebooks 1 and 2)
	tests := []struct {
		idx        int
		bits       uint8
		x, y, v, w int8
	}{
		{0, 4, 1, 1, 1, 1},    // 4-bit: first entry
		{7, 4, 0, 0, 0, 0},    // 4-bit: all zeros
		{16, 7, 2, 1, 1, 1},   // 7-bit: (2,1,1,1)
		{54, 9, 0, 2, 0, 0},   // 9-bit: (0,2,0,0)
		{55, 9, 0, 0, 2, 0},   // 9-bit: (0,0,2,0)
		{182, 12, 2, 2, 0, 2}, // 12-bit: (2,2,0,2)
		{183, 12, 2, 0, 2, 2}, // 12-bit: last entry (2,0,2,2)
	}
	for _, tt := range tests {
		e := hcb4_2[tt.idx]
		if e.Bits != tt.bits || e.X != tt.x || e.Y != tt.y || e.V != tt.v || e.W != tt.w {
			t.Errorf("hcb4_2[%d] = {%d, %d, %d, %d, %d}, want {%d, %d, %d, %d, %d}",
				tt.idx, e.Bits, e.X, e.Y, e.V, e.W,
				tt.bits, tt.x, tt.y, tt.v, tt.w)
		}
	}
}

func TestHCB4_FirstStepDuplicates(t *testing.T) {
	// 4-bit codewords appear twice in the 5-bit first-step table
	// Indices 0,1 should match, 2,3 should match, etc.
	for i := 0; i < 20; i += 2 {
		if hcb4_1[i] != hcb4_1[i+1] {
			t.Errorf("hcb4_1[%d] = %v, hcb4_1[%d] = %v, want equal (4-bit codeword duplicate)",
				i, hcb4_1[i], i+1, hcb4_1[i+1])
		}
	}
}
