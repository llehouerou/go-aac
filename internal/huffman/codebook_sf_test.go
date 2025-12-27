// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

func TestHCBSFSize(t *testing.T) {
	// Scale factor codebook must have 241 entries
	if len(hcbSF) != 241 {
		t.Errorf("hcbSF size = %d, want 241", len(hcbSF))
	}
}

func TestHCBSFValues(t *testing.T) {
	// Verify key entries from FAAD2
	// Structure: [2]uint8 where:
	// - If second value is 0, first value is the decoded scale factor
	// - If second value is non-zero, values are branch offsets
	tests := []struct {
		idx int
		v0  uint8 // First value
		v1  uint8 // Second value (0 = leaf, non-zero = branch)
	}{
		// First entry - branch node
		{0, 1, 2},
		// Leaf node: decodes to 60
		{1, 60, 0},
		// Branch node
		{2, 1, 2},
		// Leaf node: decodes to 59
		{5, 59, 0},
		// Leaf node: decodes to 61
		{9, 61, 0},
		// Leaf node: decodes to 58
		{10, 58, 0},
		// Leaf node: decodes to 0 (special case)
		{171, 0, 0},
		// Leaf node: decodes to 120
		{228, 120, 0},
		// Leaf node: decodes to 119
		{229, 119, 0},
		// Last entry: decodes to 13
		{240, 13, 0},
	}
	for _, tt := range tests {
		if hcbSF[tt.idx][0] != tt.v0 || hcbSF[tt.idx][1] != tt.v1 {
			t.Errorf("hcbSF[%d] = {%d, %d}, want {%d, %d}",
				tt.idx, hcbSF[tt.idx][0], hcbSF[tt.idx][1],
				tt.v0, tt.v1)
		}
	}
}

func TestHCBSFLeafNodes(t *testing.T) {
	// Count leaf nodes (where second value is 0)
	// Each leaf node represents a valid scale factor value
	leafCount := 0
	for i := 0; i < len(hcbSF); i++ {
		if hcbSF[i][1] == 0 {
			leafCount++
		}
	}
	// There should be 121 scale factor values (0-120)
	if leafCount != 121 {
		t.Errorf("leaf node count = %d, want 121", leafCount)
	}
}

func TestHCBSFScaleFactorRange(t *testing.T) {
	// Verify all leaf nodes have scale factor values in valid range [0, 120]
	for i := 0; i < len(hcbSF); i++ {
		if hcbSF[i][1] == 0 { // Leaf node
			sf := hcbSF[i][0]
			if sf > 120 {
				t.Errorf("hcbSF[%d] has invalid scale factor %d (max 120)", i, sf)
			}
		}
	}
}
