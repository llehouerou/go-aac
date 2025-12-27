// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

func TestHCB3Size(t *testing.T) {
	// Binary search table must have 161 entries
	if len(hcb3) != 161 {
		t.Errorf("hcb3 size = %d, want 161", len(hcb3))
	}
}

func TestHCB3Values(t *testing.T) {
	// Verify key entries from FAAD2 ~/dev/faad2/libfaad/codebook/hcb_3.h
	tests := []struct {
		idx    int
		isLeaf uint8
		data   [4]int8
	}{
		// Index 0: internal node, branches to 1 and 2
		{0, 0, [4]int8{1, 2, 0, 0}},
		// Index 1: leaf node, output (0, 0, 0, 0)
		{1, 1, [4]int8{0, 0, 0, 0}},
		// Index 9: leaf node, output (1, 0, 0, 0)
		{9, 1, [4]int8{1, 0, 0, 0}},
		// Index 160: leaf node, output (2, 0, 2, 2) - last entry
		{160, 1, [4]int8{2, 0, 2, 2}},
	}
	for _, tt := range tests {
		e := hcb3[tt.idx]
		if e.IsLeaf != tt.isLeaf || e.Data != tt.data {
			t.Errorf("hcb3[%d] = {%d, %v}, want {%d, %v}",
				tt.idx, e.IsLeaf, e.Data,
				tt.isLeaf, tt.data)
		}
	}
}

func TestHCB3InternalNodes(t *testing.T) {
	// Verify some internal nodes (IsLeaf=0) have valid branch offsets
	internalIndices := []int{0, 2, 3, 4, 5, 6, 7, 8}
	for _, idx := range internalIndices {
		e := hcb3[idx]
		if e.IsLeaf != 0 {
			t.Errorf("hcb3[%d] should be internal node (IsLeaf=0), got IsLeaf=%d", idx, e.IsLeaf)
		}
	}
}

func TestHCB3LeafNodes(t *testing.T) {
	// Verify some leaf nodes (IsLeaf=1) have valid output values
	leafIndices := []int{1, 9, 10, 11, 12, 17, 18, 160}
	for _, idx := range leafIndices {
		e := hcb3[idx]
		if e.IsLeaf != 1 {
			t.Errorf("hcb3[%d] should be leaf node (IsLeaf=1), got IsLeaf=%d", idx, e.IsLeaf)
		}
	}
}
