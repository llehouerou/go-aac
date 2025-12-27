// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

func TestHCB7Size(t *testing.T) {
	// Binary search pair table must have 127 entries
	if len(hcb7) != 127 {
		t.Errorf("hcb7 size = %d, want 127", len(hcb7))
	}
}

func TestHCB7Values(t *testing.T) {
	// Verify key entries from FAAD2 ~/dev/faad2/libfaad/codebook/hcb_7.h
	tests := []struct {
		idx    int
		isLeaf uint8
		data   [2]int8
	}{
		// Index 0: internal node, branches to 1 and 2
		{0, 0, [2]int8{1, 2}},
		// Index 1: leaf node, output (0, 0)
		{1, 1, [2]int8{0, 0}},
		// Index 5: leaf node, output (1, 0)
		{5, 1, [2]int8{1, 0}},
		// Index 6: leaf node, output (0, 1)
		{6, 1, [2]int8{0, 1}},
		// Index 9: leaf node, output (1, 1)
		{9, 1, [2]int8{1, 1}},
		// Index 19: leaf node, output (2, 1)
		{19, 1, [2]int8{2, 1}},
		// Index 20: leaf node, output (1, 2)
		{20, 1, [2]int8{1, 2}},
		// Index 53: leaf node, output (3, 3)
		{53, 1, [2]int8{3, 3}},
		// Index 126: leaf node, output (7, 7) - last entry
		{126, 1, [2]int8{7, 7}},
	}
	for _, tt := range tests {
		e := hcb7[tt.idx]
		if e.IsLeaf != tt.isLeaf || e.Data != tt.data {
			t.Errorf("hcb7[%d] = {%d, %v}, want {%d, %v}",
				tt.idx, e.IsLeaf, e.Data,
				tt.isLeaf, tt.data)
		}
	}
}

func TestHCB7InternalNodes(t *testing.T) {
	// Verify some internal nodes (IsLeaf=0) have valid branch offsets
	internalIndices := []int{0, 2, 3, 4, 7, 8, 10, 11, 12, 13, 14, 15, 16, 17, 18}
	for _, idx := range internalIndices {
		e := hcb7[idx]
		if e.IsLeaf != 0 {
			t.Errorf("hcb7[%d] should be internal node (IsLeaf=0), got IsLeaf=%d", idx, e.IsLeaf)
		}
	}
}

func TestHCB7LeafNodes(t *testing.T) {
	// Verify some leaf nodes (IsLeaf=1) have valid output values
	leafIndices := []int{1, 5, 6, 9, 19, 20, 21, 22, 31, 32, 33, 34, 35, 126}
	for _, idx := range leafIndices {
		e := hcb7[idx]
		if e.IsLeaf != 1 {
			t.Errorf("hcb7[%d] should be leaf node (IsLeaf=1), got IsLeaf=%d", idx, e.IsLeaf)
		}
	}
}

func TestHCB7UnsignedValues(t *testing.T) {
	// Codebook 7 is an unsigned codebook with values from 0 to 7
	// Verify that values are correctly stored
	unsignedTests := []struct {
		idx  int
		data [2]int8
	}{
		{1, [2]int8{0, 0}},   // zero
		{5, [2]int8{1, 0}},   // x=1
		{6, [2]int8{0, 1}},   // y=1
		{9, [2]int8{1, 1}},   // both 1
		{19, [2]int8{2, 1}},  // x=2
		{33, [2]int8{2, 2}},  // both 2
		{53, [2]int8{3, 3}},  // both 3
		{96, [2]int8{4, 4}},  // both 4
		{105, [2]int8{5, 5}}, // both 5
		{124, [2]int8{6, 6}}, // both 6
		{126, [2]int8{7, 7}}, // maximum value (both 7)
	}
	for _, tt := range unsignedTests {
		e := hcb7[tt.idx]
		if e.IsLeaf != 1 {
			t.Errorf("hcb7[%d] should be leaf node (IsLeaf=1), got IsLeaf=%d", tt.idx, e.IsLeaf)
			continue
		}
		if e.Data != tt.data {
			t.Errorf("hcb7[%d].Data = %v, want %v", tt.idx, e.Data, tt.data)
		}
	}
}

func TestHCB7BranchOffsets(t *testing.T) {
	// Verify that internal node branch offsets are reasonable
	// For each internal node, ensure offsets point to valid indices
	for i, e := range hcb7 {
		if e.IsLeaf == 0 {
			// Internal node - check that branches are valid
			leftOffset := int(e.Data[0])
			rightOffset := int(e.Data[1])

			// Offsets should be positive and point to valid indices
			if leftOffset <= 0 || rightOffset <= 0 {
				t.Errorf("hcb7[%d] has non-positive branch offset: left=%d, right=%d",
					i, leftOffset, rightOffset)
			}

			// The target index should be within bounds
			leftTarget := i + leftOffset
			rightTarget := i + rightOffset
			if leftTarget >= len(hcb7) || rightTarget >= len(hcb7) {
				t.Errorf("hcb7[%d] branch offset out of bounds: left->%d, right->%d (max=%d)",
					i, leftTarget, rightTarget, len(hcb7)-1)
			}
		}
	}
}
