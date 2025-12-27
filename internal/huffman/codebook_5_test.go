// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

func TestHCB5Size(t *testing.T) {
	// Binary search pair table must have 161 entries
	if len(hcb5) != 161 {
		t.Errorf("hcb5 size = %d, want 161", len(hcb5))
	}
}

func TestHCB5Values(t *testing.T) {
	// Verify key entries from FAAD2 ~/dev/faad2/libfaad/codebook/hcb_5.h
	tests := []struct {
		idx    int
		isLeaf uint8
		data   [2]int8
	}{
		// Index 0: internal node, branches to 1 and 2
		{0, 0, [2]int8{1, 2}},
		// Index 1: leaf node, output (0, 0)
		{1, 1, [2]int8{0, 0}},
		// Index 9: leaf node, output (-1, 0)
		{9, 1, [2]int8{-1, 0}},
		// Index 10: leaf node, output (1, 0)
		{10, 1, [2]int8{1, 0}},
		// Index 11: leaf node, output (0, 1)
		{11, 1, [2]int8{0, 1}},
		// Index 12: leaf node, output (0, -1)
		{12, 1, [2]int8{0, -1}},
		// Index 57: leaf node, output (-3, 0)
		{57, 1, [2]int8{-3, 0}},
		// Index 160: leaf node, output (-4, -4) - last entry
		{160, 1, [2]int8{-4, -4}},
	}
	for _, tt := range tests {
		e := hcb5[tt.idx]
		if e.IsLeaf != tt.isLeaf || e.Data != tt.data {
			t.Errorf("hcb5[%d] = {%d, %v}, want {%d, %v}",
				tt.idx, e.IsLeaf, e.Data,
				tt.isLeaf, tt.data)
		}
	}
}

func TestHCB5InternalNodes(t *testing.T) {
	// Verify some internal nodes (IsLeaf=0) have valid branch offsets
	internalIndices := []int{0, 2, 3, 4, 5, 6, 7, 8, 13, 14, 15, 16}
	for _, idx := range internalIndices {
		e := hcb5[idx]
		if e.IsLeaf != 0 {
			t.Errorf("hcb5[%d] should be internal node (IsLeaf=0), got IsLeaf=%d", idx, e.IsLeaf)
		}
	}
}

func TestHCB5LeafNodes(t *testing.T) {
	// Verify some leaf nodes (IsLeaf=1) have valid output values
	leafIndices := []int{1, 9, 10, 11, 12, 17, 18, 19, 20, 160}
	for _, idx := range leafIndices {
		e := hcb5[idx]
		if e.IsLeaf != 1 {
			t.Errorf("hcb5[%d] should be leaf node (IsLeaf=1), got IsLeaf=%d", idx, e.IsLeaf)
		}
	}
}

func TestHCB5SignedValues(t *testing.T) {
	// Codebook 5 is a signed codebook with values from -4 to 4
	// Verify that negative values are correctly stored
	signedTests := []struct {
		idx  int
		data [2]int8
	}{
		{9, [2]int8{-1, 0}},    // negative x
		{12, [2]int8{0, -1}},   // negative y
		{17, [2]int8{1, -1}},   // mixed positive/negative
		{18, [2]int8{-1, 1}},   // mixed negative/positive
		{19, [2]int8{-1, -1}},  // both negative
		{33, [2]int8{-2, 0}},   // larger negative
		{57, [2]int8{-3, 0}},   // -3
		{160, [2]int8{-4, -4}}, // maximum negative
	}
	for _, tt := range signedTests {
		e := hcb5[tt.idx]
		if e.IsLeaf != 1 {
			t.Errorf("hcb5[%d] should be leaf node (IsLeaf=1), got IsLeaf=%d", tt.idx, e.IsLeaf)
			continue
		}
		if e.Data != tt.data {
			t.Errorf("hcb5[%d].Data = %v, want %v", tt.idx, e.Data, tt.data)
		}
	}
}
