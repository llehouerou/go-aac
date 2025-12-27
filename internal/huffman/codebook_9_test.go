// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

func TestHCB9Size(t *testing.T) {
	// Binary search pair table must have 337 entries (largest binary codebook)
	if len(hcb9) != 337 {
		t.Errorf("hcb9 size = %d, want 337", len(hcb9))
	}
}

func TestHCB9Values(t *testing.T) {
	// Verify key entries from FAAD2 ~/dev/faad2/libfaad/codebook/hcb_9.h
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
		// Index 74: leaf node, output (3, 3)
		{74, 1, [2]int8{3, 3}},
		// Index 116: leaf node, output (4, 4)
		{116, 1, [2]int8{4, 4}},
		// Index 172: leaf node, output (5, 5)
		{172, 1, [2]int8{5, 5}},
		// Index 224: leaf node, output (6, 6)
		{224, 1, [2]int8{6, 6}},
		// Index 282: leaf node, output (7, 7)
		{282, 1, [2]int8{7, 7}},
		// Index 277: leaf node, output (8, 8)
		{277, 1, [2]int8{8, 8}},
		// Index 300: leaf node, output (9, 9)
		{300, 1, [2]int8{9, 9}},
		// Index 325: leaf node, output (10, 10)
		{325, 1, [2]int8{10, 10}},
		// Index 334: leaf node, output (11, 11)
		{334, 1, [2]int8{11, 11}},
		// Index 336: leaf node, output (12, 12) - last entry
		{336, 1, [2]int8{12, 12}},
	}
	for _, tt := range tests {
		e := hcb9[tt.idx]
		if e.IsLeaf != tt.isLeaf || e.Data != tt.data {
			t.Errorf("hcb9[%d] = {%d, %v}, want {%d, %v}",
				tt.idx, e.IsLeaf, e.Data,
				tt.isLeaf, tt.data)
		}
	}
}

func TestHCB9InternalNodes(t *testing.T) {
	// Verify some internal nodes (IsLeaf=0) have valid branch offsets
	internalIndices := []int{0, 2, 3, 4, 7, 8, 10, 11, 12, 13, 14, 15, 16, 17, 18, 23, 24, 25}
	for _, idx := range internalIndices {
		e := hcb9[idx]
		if e.IsLeaf != 0 {
			t.Errorf("hcb9[%d] should be internal node (IsLeaf=0), got IsLeaf=%d", idx, e.IsLeaf)
		}
	}
}

func TestHCB9LeafNodes(t *testing.T) {
	// Verify some leaf nodes (IsLeaf=1) have valid output values
	leafIndices := []int{1, 5, 6, 9, 19, 20, 21, 22, 31, 32, 33, 47, 48, 49, 336}
	for _, idx := range leafIndices {
		e := hcb9[idx]
		if e.IsLeaf != 1 {
			t.Errorf("hcb9[%d] should be leaf node (IsLeaf=1), got IsLeaf=%d", idx, e.IsLeaf)
		}
	}
}

func TestHCB9UnsignedValues(t *testing.T) {
	// Codebook 9 is an unsigned codebook with values from 0 to 12
	// Verify that values are correctly stored
	unsignedTests := []struct {
		idx  int
		data [2]int8
	}{
		{1, [2]int8{0, 0}},     // zero
		{5, [2]int8{1, 0}},     // x=1
		{6, [2]int8{0, 1}},     // y=1
		{9, [2]int8{1, 1}},     // both 1
		{19, [2]int8{2, 1}},    // x=2
		{32, [2]int8{2, 2}},    // both 2
		{74, [2]int8{3, 3}},    // both 3
		{116, [2]int8{4, 4}},   // both 4
		{172, [2]int8{5, 5}},   // both 5
		{224, [2]int8{6, 6}},   // both 6
		{282, [2]int8{7, 7}},   // both 7
		{277, [2]int8{8, 8}},   // both 8
		{300, [2]int8{9, 9}},   // both 9
		{325, [2]int8{10, 10}}, // both 10
		{334, [2]int8{11, 11}}, // both 11
		{336, [2]int8{12, 12}}, // maximum value (both 12)
	}
	for _, tt := range unsignedTests {
		e := hcb9[tt.idx]
		if e.IsLeaf != 1 {
			t.Errorf("hcb9[%d] should be leaf node (IsLeaf=1), got IsLeaf=%d", tt.idx, e.IsLeaf)
			continue
		}
		if e.Data != tt.data {
			t.Errorf("hcb9[%d].Data = %v, want %v", tt.idx, e.Data, tt.data)
		}
	}
}

func TestHCB9BranchOffsets(t *testing.T) {
	// Verify that internal node branch offsets are reasonable
	// For each internal node, ensure offsets point to valid indices
	for i, e := range hcb9 {
		if e.IsLeaf == 0 {
			// Internal node - check that branches are valid
			leftOffset := int(e.Data[0])
			rightOffset := int(e.Data[1])

			// Offsets should be positive and point to valid indices
			if leftOffset <= 0 || rightOffset <= 0 {
				t.Errorf("hcb9[%d] has non-positive branch offset: left=%d, right=%d",
					i, leftOffset, rightOffset)
			}

			// The target index should be within bounds
			leftTarget := i + leftOffset
			rightTarget := i + rightOffset
			if leftTarget >= len(hcb9) || rightTarget >= len(hcb9) {
				t.Errorf("hcb9[%d] branch offset out of bounds: left->%d, right->%d (max=%d)",
					i, leftTarget, rightTarget, len(hcb9)-1)
			}
		}
	}
}

func TestHCB9MaxValue(t *testing.T) {
	// Codebook 9 supports values 0-12, verify maximum values are present
	foundMax := false
	for _, e := range hcb9 {
		if e.IsLeaf == 1 {
			if e.Data[0] == 12 && e.Data[1] == 12 {
				foundMax = true
				break
			}
		}
	}
	if !foundMax {
		t.Error("hcb9 should contain leaf with maximum values (12, 12)")
	}
}

func TestHCB9AllLeafValuesInRange(t *testing.T) {
	// All leaf values should be in the range 0-12
	for i, e := range hcb9 {
		if e.IsLeaf == 1 {
			if e.Data[0] < 0 || e.Data[0] > 12 {
				t.Errorf("hcb9[%d].Data[0] = %d, should be in range 0-12", i, e.Data[0])
			}
			if e.Data[1] < 0 || e.Data[1] > 12 {
				t.Errorf("hcb9[%d].Data[1] = %d, should be in range 0-12", i, e.Data[1])
			}
		}
	}
}
