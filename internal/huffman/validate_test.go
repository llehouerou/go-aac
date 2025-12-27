// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

// TestAllCodebookSizes verifies that all codebook table sizes match FAAD2.
// This ensures we copied the complete tables without truncation or errors.
//
// Reference: ~/dev/faad2/libfaad/codebook/hcb_*.h
func TestAllCodebookSizes(t *testing.T) {
	tests := []struct {
		name string
		got  int
		want int
	}{
		// Codebook 1: 2-step quad codebook
		{"hcb1_1", len(hcb1_1), 32},
		{"hcb1_2", len(hcb1_2), 113},

		// Codebook 2: 2-step quad codebook
		{"hcb2_1", len(hcb2_1), 32},
		{"hcb2_2", len(hcb2_2), 85},

		// Codebook 3: binary quad codebook
		{"hcb3", len(hcb3), 161},

		// Codebook 4: 2-step quad codebook (unsigned)
		{"hcb4_1", len(hcb4_1), 32},
		{"hcb4_2", len(hcb4_2), 184},

		// Codebook 5: binary pair codebook (signed)
		{"hcb5", len(hcb5), 161},

		// Codebook 6: 2-step pair codebook
		{"hcb6_1", len(hcb6_1), 32},
		{"hcb6_2", len(hcb6_2), 125},

		// Codebook 7: binary pair codebook (unsigned)
		{"hcb7", len(hcb7), 127},

		// Codebook 8: 2-step pair codebook (unsigned)
		{"hcb8_1", len(hcb8_1), 32},
		{"hcb8_2", len(hcb8_2), 83},

		// Codebook 9: binary pair codebook (unsigned, largest)
		{"hcb9", len(hcb9), 337},

		// Codebook 10: 2-step pair codebook (6-bit first step)
		{"hcb10_1", len(hcb10_1), 64},
		{"hcb10_2", len(hcb10_2), 209},

		// Codebook 11: 2-step pair codebook with escape
		{"hcb11_1", len(hcb11_1), 32},
		{"hcb11_2", len(hcb11_2), 374},

		// Scale factor codebook: binary search
		{"hcbSF", len(hcbSF), 241},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s: got size %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestCodebookConsistency verifies that lookup table structures are internally consistent.
// This checks that 2-step codebooks have matching first-step entries and that
// first-step sizes match HCBN values.
func TestCodebookConsistency(t *testing.T) {
	// Test HCBN first-step bit counts
	t.Run("HCBN values", func(t *testing.T) {
		expectedHCBN := [12]uint8{
			0, // 0: reserved
			5, // 1: hcb1_1 uses 5-bit first step (2^5 = 32 entries)
			5, // 2: hcb2_1 uses 5-bit first step
			0, // 3: binary search (no 2-step)
			5, // 4: hcb4_1 uses 5-bit first step
			0, // 5: binary search (no 2-step)
			5, // 6: hcb6_1 uses 5-bit first step
			0, // 7: binary search (no 2-step)
			5, // 8: hcb8_1 uses 5-bit first step
			0, // 9: binary search (no 2-step)
			6, // 10: hcb10_1 uses 6-bit first step (2^6 = 64 entries)
			5, // 11: hcb11_1 uses 5-bit first step
		}
		for i, want := range expectedHCBN {
			if HCBN[i] != want {
				t.Errorf("HCBN[%d]: got %d, want %d", i, HCBN[i], want)
			}
		}
	})

	// Test first-step table sizes match 2^HCBN
	t.Run("first-step sizes match HCBN", func(t *testing.T) {
		// Check 5-bit first step tables have 32 entries
		if len(hcb1_1) != 32 {
			t.Errorf("hcb1_1: HCBN[1]=5 implies 32 entries, got %d", len(hcb1_1))
		}
		if len(hcb2_1) != 32 {
			t.Errorf("hcb2_1: HCBN[2]=5 implies 32 entries, got %d", len(hcb2_1))
		}
		if len(hcb4_1) != 32 {
			t.Errorf("hcb4_1: HCBN[4]=5 implies 32 entries, got %d", len(hcb4_1))
		}
		if len(hcb6_1) != 32 {
			t.Errorf("hcb6_1: HCBN[6]=5 implies 32 entries, got %d", len(hcb6_1))
		}
		if len(hcb8_1) != 32 {
			t.Errorf("hcb8_1: HCBN[8]=5 implies 32 entries, got %d", len(hcb8_1))
		}
		if len(hcb11_1) != 32 {
			t.Errorf("hcb11_1: HCBN[11]=5 implies 32 entries, got %d", len(hcb11_1))
		}

		// Check 6-bit first step table (codebook 10 only)
		if len(hcb10_1) != 64 {
			t.Errorf("hcb10_1: HCBN[10]=6 implies 64 entries, got %d", len(hcb10_1))
		}
	})

	// Test HCBTable pointers are correctly set
	t.Run("HCBTable pointers", func(t *testing.T) {
		// 2-step codebooks should have non-nil first-step tables
		for _, i := range []int{1, 2, 4, 6, 8, 10, 11} {
			if HCBTable[i] == nil {
				t.Errorf("HCBTable[%d]: expected non-nil for 2-step codebook", i)
			}
		}
		// Binary search codebooks should have nil first-step tables
		for _, i := range []int{0, 3, 5, 7, 9} {
			if HCBTable[i] != nil {
				t.Errorf("HCBTable[%d]: expected nil for binary/reserved codebook", i)
			}
		}
	})

	// Test HCB2QuadTable pointers (quad codebooks 1, 2, 4)
	t.Run("HCB2QuadTable pointers", func(t *testing.T) {
		for _, i := range []int{1, 2, 4} {
			if HCB2QuadTable[i] == nil {
				t.Errorf("HCB2QuadTable[%d]: expected non-nil for quad codebook", i)
			}
		}
		for _, i := range []int{0, 3, 5, 6, 7, 8, 9, 10, 11} {
			if HCB2QuadTable[i] != nil {
				t.Errorf("HCB2QuadTable[%d]: expected nil for non-quad codebook", i)
			}
		}
	})

	// Test HCB2PairTable pointers (pair codebooks 6, 8, 10, 11)
	t.Run("HCB2PairTable pointers", func(t *testing.T) {
		for _, i := range []int{6, 8, 10, 11} {
			if HCB2PairTable[i] == nil {
				t.Errorf("HCB2PairTable[%d]: expected non-nil for 2-step pair codebook", i)
			}
		}
		for _, i := range []int{0, 1, 2, 3, 4, 5, 7, 9} {
			if HCB2PairTable[i] != nil {
				t.Errorf("HCB2PairTable[%d]: expected nil for non-pair codebook", i)
			}
		}
	})

	// Test HCBBinPairTable pointers (binary pair codebooks 5, 7, 9)
	t.Run("HCBBinPairTable pointers", func(t *testing.T) {
		for _, i := range []int{5, 7, 9} {
			if HCBBinPairTable[i] == nil {
				t.Errorf("HCBBinPairTable[%d]: expected non-nil for binary pair codebook", i)
			}
		}
		for _, i := range []int{0, 1, 2, 3, 4, 6, 8, 10, 11} {
			if HCBBinPairTable[i] != nil {
				t.Errorf("HCBBinPairTable[%d]: expected nil for non-binary pair", i)
			}
		}
	})

	// Test special tables
	t.Run("HCB3 (binary quad)", func(t *testing.T) {
		if HCB3 == nil {
			t.Error("HCB3: expected non-nil for binary quad codebook 3")
		}
	})

	t.Run("HCBSF (scale factors)", func(t *testing.T) {
		if HCBSF == nil {
			t.Error("HCBSF: expected non-nil for scale factor codebook")
		}
	})

	// Test UnsignedCB flags
	t.Run("UnsignedCB flags", func(t *testing.T) {
		// Signed codebooks: 1, 2, 5, 6
		for _, i := range []int{0, 1, 2, 5, 6} {
			if UnsignedCB[i] {
				t.Errorf("UnsignedCB[%d]: expected false (signed codebook)", i)
			}
		}
		// Unsigned codebooks: 3, 4, 7, 8, 9, 10, 11
		for _, i := range []int{3, 4, 7, 8, 9, 10, 11} {
			if !UnsignedCB[i] {
				t.Errorf("UnsignedCB[%d]: expected true (unsigned codebook)", i)
			}
		}
	})
}

// TestSpotCheckValues verifies critical values from each codebook table.
// Checks first entry, last entry, and a characteristic middle entry.
//
// Reference: ~/dev/faad2/libfaad/codebook/hcb_*.h
func TestSpotCheckValues(t *testing.T) {
	// Codebook 1 - 2-step quad
	t.Run("hcb1_1", func(t *testing.T) {
		// First entry: 1-bit codeword maps to offset 0
		checkHCB(t, "first", hcb1_1[0], HCB{0, 0})
		// Entry 16: 5-bit codeword maps to offset 1
		checkHCB(t, "index 16", hcb1_1[16], HCB{1, 0})
		// Last entry: 9/10/11 bit codewords
		checkHCB(t, "last", hcb1_1[31], HCB{49, 6})
	})

	t.Run("hcb1_2", func(t *testing.T) {
		// First entry: 1-bit codeword (0,0,0,0)
		checkHCB2Quad(t, "first", hcb1_2[0], HCB2Quad{1, 0, 0, 0, 0})
		// Entry 50: middle of table
		checkHCB2Quad(t, "index 50", hcb1_2[50], HCB2Quad{9, -1, 1, 0, -1})
		// Last entry: 11-bit codeword
		checkHCB2Quad(t, "last", hcb1_2[112], HCB2Quad{11, 1, 1, 1, -1})
	})

	// Codebook 2 - 2-step quad
	t.Run("hcb2_1", func(t *testing.T) {
		checkHCB(t, "first", hcb2_1[0], HCB{0, 0})
		checkHCB(t, "index 25", hcb2_1[25], HCB{33, 2})
		checkHCB(t, "last", hcb2_1[31], HCB{69, 4})
	})

	t.Run("hcb2_2", func(t *testing.T) {
		// First: 3-bit codeword (0,0,0,0)
		checkHCB2Quad(t, "first", hcb2_2[0], HCB2Quad{3, 0, 0, 0, 0})
		// Middle: entry 40 is {7, -1, 1, 1, 0}
		checkHCB2Quad(t, "index 40", hcb2_2[40], HCB2Quad{7, -1, 1, 1, 0})
		// Last: 9-bit codeword
		checkHCB2Quad(t, "last", hcb2_2[84], HCB2Quad{9, 1, 1, 1, -1})
	})

	// Codebook 3 - binary quad
	t.Run("hcb3", func(t *testing.T) {
		// First: internal node
		checkHCBBinQuad(t, "first", hcb3[0], HCBBinQuad{0, [4]int8{1, 2, 0, 0}})
		// Entry 1: leaf node (0,0,0,0)
		checkHCBBinQuad(t, "index 1", hcb3[1], HCBBinQuad{1, [4]int8{0, 0, 0, 0}})
		// Middle: entry 80 is {1, [1, 1, 2, 0]}
		checkHCBBinQuad(t, "index 80", hcb3[80], HCBBinQuad{1, [4]int8{1, 1, 2, 0}})
		// Last: leaf node
		checkHCBBinQuad(t, "last", hcb3[160], HCBBinQuad{1, [4]int8{2, 0, 2, 2}})
	})

	// Codebook 4 - 2-step quad (unsigned)
	t.Run("hcb4_1", func(t *testing.T) {
		checkHCB(t, "first", hcb4_1[0], HCB{0, 0})
		checkHCB(t, "index 26", hcb4_1[26], HCB{16, 2})
		checkHCB(t, "last", hcb4_1[31], HCB{56, 7})
	})

	t.Run("hcb4_2", func(t *testing.T) {
		// First: 4-bit codeword (1,1,1,1)
		checkHCB2Quad(t, "first", hcb4_2[0], HCB2Quad{4, 1, 1, 1, 1})
		// Middle: entry 90 is {9, 1, 2, 1, 2}
		checkHCB2Quad(t, "index 90", hcb4_2[90], HCB2Quad{9, 1, 2, 1, 2})
		// Last: 12-bit codeword
		checkHCB2Quad(t, "last", hcb4_2[183], HCB2Quad{12, 2, 0, 2, 2})
	})

	// Codebook 5 - binary pair (signed)
	t.Run("hcb5", func(t *testing.T) {
		// First: internal node
		checkHCBBinPair(t, "first", hcb5[0], HCBBinPair{0, [2]int8{1, 2}})
		// Entry 1: leaf node (0,0)
		checkHCBBinPair(t, "index 1", hcb5[1], HCBBinPair{1, [2]int8{0, 0}})
		// Middle: entry 80
		checkHCBBinPair(t, "index 80", hcb5[80], HCBBinPair{1, [2]int8{-1, 3}})
		// Last: leaf node
		checkHCBBinPair(t, "last", hcb5[160], HCBBinPair{1, [2]int8{-4, -4}})
	})

	// Codebook 6 - 2-step pair
	t.Run("hcb6_1", func(t *testing.T) {
		checkHCB(t, "first", hcb6_1[0], HCB{0, 0})
		checkHCB(t, "index 26", hcb6_1[26], HCB{25, 2})
		checkHCB(t, "last", hcb6_1[31], HCB{61, 6})
	})

	t.Run("hcb6_2", func(t *testing.T) {
		// First: 4-bit codeword (0,0)
		checkHCB2Pair(t, "first", hcb6_2[0], HCB2Pair{4, 0, 0})
		// Middle: entry 60 is {9, 0, -4}
		checkHCB2Pair(t, "index 60", hcb6_2[60], HCB2Pair{9, 0, -4})
		// Last: 11-bit codeword
		checkHCB2Pair(t, "last", hcb6_2[124], HCB2Pair{11, 4, -4})
	})

	// Codebook 7 - binary pair (unsigned)
	t.Run("hcb7", func(t *testing.T) {
		// First: internal node
		checkHCBBinPair(t, "first", hcb7[0], HCBBinPair{0, [2]int8{1, 2}})
		// Entry 1: leaf node (0,0)
		checkHCBBinPair(t, "index 1", hcb7[1], HCBBinPair{1, [2]int8{0, 0}})
		// Middle: entry 60 is {0, [15, 16]}
		checkHCBBinPair(t, "index 60", hcb7[60], HCBBinPair{0, [2]int8{15, 16}})
		// Last: leaf node (7,7)
		checkHCBBinPair(t, "last", hcb7[126], HCBBinPair{1, [2]int8{7, 7}})
	})

	// Codebook 8 - 2-step pair (unsigned)
	t.Run("hcb8_1", func(t *testing.T) {
		checkHCB(t, "first", hcb8_1[0], HCB{0, 0})
		checkHCB(t, "index 26", hcb8_1[26], HCB{23, 2})
		checkHCB(t, "last", hcb8_1[31], HCB{51, 5})
	})

	t.Run("hcb8_2", func(t *testing.T) {
		// First: 3-bit codeword (1,1)
		checkHCB2Pair(t, "first", hcb8_2[0], HCB2Pair{3, 1, 1})
		// Middle: entry 40 is {8, 6, 3}
		checkHCB2Pair(t, "index 40", hcb8_2[40], HCB2Pair{8, 6, 3})
		// Last: 10-bit codeword
		checkHCB2Pair(t, "last", hcb8_2[82], HCB2Pair{10, 7, 7})
	})

	// Codebook 9 - binary pair (unsigned, largest)
	t.Run("hcb9", func(t *testing.T) {
		// First: internal node
		checkHCBBinPair(t, "first", hcb9[0], HCBBinPair{0, [2]int8{1, 2}})
		// Entry 1: leaf node (0,0)
		checkHCBBinPair(t, "index 1", hcb9[1], HCBBinPair{1, [2]int8{0, 0}})
		// Middle: entry 168
		checkHCBBinPair(t, "index 168", hcb9[168], HCBBinPair{1, [2]int8{0, 8}})
		// Last: leaf node (12,12)
		checkHCBBinPair(t, "last", hcb9[336], HCBBinPair{1, [2]int8{12, 12}})
	})

	// Codebook 10 - 2-step pair (6-bit first step)
	t.Run("hcb10_1", func(t *testing.T) {
		checkHCB(t, "first", hcb10_1[0], HCB{0, 0})
		checkHCB(t, "index 32", hcb10_1[32], HCB{15, 0})
		checkHCB(t, "last", hcb10_1[63], HCB{145, 6})
	})

	t.Run("hcb10_2", func(t *testing.T) {
		// First: 4-bit codeword (1,1)
		checkHCB2Pair(t, "first", hcb10_2[0], HCB2Pair{4, 1, 1})
		// Middle: entry 100 is {9, 8, 0}
		checkHCB2Pair(t, "index 100", hcb10_2[100], HCB2Pair{9, 8, 0})
		// Last: 12-bit codeword
		checkHCB2Pair(t, "last", hcb10_2[208], HCB2Pair{12, 12, 12})
	})

	// Codebook 11 - 2-step pair with escape
	t.Run("hcb11_1", func(t *testing.T) {
		checkHCB(t, "first", hcb11_1[0], HCB{0, 0})
		checkHCB(t, "index 16", hcb11_1[16], HCB{26, 2})
		checkHCB(t, "last", hcb11_1[31], HCB{246, 7})
	})

	t.Run("hcb11_2", func(t *testing.T) {
		// First: 4-bit codeword (0,0)
		checkHCB2Pair(t, "first", hcb11_2[0], HCB2Pair{4, 0, 0})
		// Entry with escape (16,16)
		checkHCB2Pair(t, "index 2", hcb11_2[2], HCB2Pair{5, 16, 16})
		// Middle: entry 186 is {10, 7, 11}
		checkHCB2Pair(t, "index 186", hcb11_2[186], HCB2Pair{10, 7, 11})
		// Last: 12-bit codeword
		checkHCB2Pair(t, "last", hcb11_2[373], HCB2Pair{12, 15, 15})
	})

	// Scale factor codebook - binary search
	t.Run("hcbSF", func(t *testing.T) {
		// First: internal node
		checkSF(t, "first", hcbSF[0], [2]uint8{1, 2})
		// Entry 1: leaf node, value 60
		checkSF(t, "index 1", hcbSF[1], [2]uint8{60, 0})
		// Entry 171: leaf node, value 0 (important: 0,0 means value 0)
		checkSF(t, "index 171", hcbSF[171], [2]uint8{0, 0})
		// Last entry: value 13
		checkSF(t, "last", hcbSF[240], [2]uint8{13, 0})
	})
}

// Helper functions for spot-check comparisons

func checkHCB(t *testing.T, name string, got, want HCB) {
	t.Helper()
	if got.Offset != want.Offset || got.ExtraBits != want.ExtraBits {
		t.Errorf("%s: got {%d, %d}, want {%d, %d}",
			name, got.Offset, got.ExtraBits, want.Offset, want.ExtraBits)
	}
}

func checkHCB2Quad(t *testing.T, name string, got, want HCB2Quad) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got {%d, %d, %d, %d, %d}, want {%d, %d, %d, %d, %d}",
			name, got.Bits, got.X, got.Y, got.V, got.W,
			want.Bits, want.X, want.Y, want.W, want.W)
	}
}

func checkHCB2Pair(t *testing.T, name string, got, want HCB2Pair) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got {%d, %d, %d}, want {%d, %d, %d}",
			name, got.Bits, got.X, got.Y, want.Bits, want.X, want.Y)
	}
}

func checkHCBBinQuad(t *testing.T, name string, got, want HCBBinQuad) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got {%d, [%d, %d, %d, %d]}, want {%d, [%d, %d, %d, %d]}",
			name, got.IsLeaf, got.Data[0], got.Data[1], got.Data[2], got.Data[3],
			want.IsLeaf, want.Data[0], want.Data[1], want.Data[2], want.Data[3])
	}
}

func checkHCBBinPair(t *testing.T, name string, got, want HCBBinPair) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got {%d, [%d, %d]}, want {%d, [%d, %d]}",
			name, got.IsLeaf, got.Data[0], got.Data[1],
			want.IsLeaf, want.Data[0], want.Data[1])
	}
}

func checkSF(t *testing.T, name string, got, want [2]uint8) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got [%d, %d], want [%d, %d]",
			name, got[0], got[1], want[0], want[1])
	}
}

// TestBinaryCodebookTreeStructure verifies that binary codebook trees
// are well-formed (all internal nodes have valid branch offsets).
func TestBinaryCodebookTreeStructure(t *testing.T) {
	t.Run("hcb3 tree structure", func(t *testing.T) {
		checkBinQuadTree(t, "hcb3", hcb3[:])
	})

	t.Run("hcb5 tree structure", func(t *testing.T) {
		checkBinPairTree(t, "hcb5", hcb5[:])
	})

	t.Run("hcb7 tree structure", func(t *testing.T) {
		checkBinPairTree(t, "hcb7", hcb7[:])
	})

	t.Run("hcb9 tree structure", func(t *testing.T) {
		checkBinPairTree(t, "hcb9", hcb9[:])
	})

	t.Run("hcbSF tree structure", func(t *testing.T) {
		checkSFTree(t, "hcbSF", hcbSF[:])
	})
}

func checkBinQuadTree(t *testing.T, name string, table []HCBBinQuad) {
	t.Helper()
	for i, entry := range table {
		if entry.IsLeaf == 0 {
			// Internal node - check branch offsets are valid
			offset0 := i + int(entry.Data[0])
			offset1 := i + int(entry.Data[1])
			if offset0 >= len(table) {
				t.Errorf("%s[%d]: branch 0 offset %d exceeds table size %d",
					name, i, offset0, len(table))
			}
			if offset1 >= len(table) {
				t.Errorf("%s[%d]: branch 1 offset %d exceeds table size %d",
					name, i, offset1, len(table))
			}
		}
	}
}

func checkBinPairTree(t *testing.T, name string, table []HCBBinPair) {
	t.Helper()
	for i, entry := range table {
		if entry.IsLeaf == 0 {
			// Internal node - check branch offsets are valid
			offset0 := i + int(entry.Data[0])
			offset1 := i + int(entry.Data[1])
			if offset0 >= len(table) {
				t.Errorf("%s[%d]: branch 0 offset %d exceeds table size %d",
					name, i, offset0, len(table))
			}
			if offset1 >= len(table) {
				t.Errorf("%s[%d]: branch 1 offset %d exceeds table size %d",
					name, i, offset1, len(table))
			}
		}
	}
}

func checkSFTree(t *testing.T, name string, table [][2]uint8) {
	t.Helper()
	for i, entry := range table {
		// If second value is non-zero, this is an internal node
		if entry[1] != 0 {
			offset0 := i + int(entry[0])
			offset1 := i + int(entry[1])
			if offset0 >= len(table) {
				t.Errorf("%s[%d]: branch 0 offset %d exceeds table size %d",
					name, i, offset0, len(table))
			}
			if offset1 >= len(table) {
				t.Errorf("%s[%d]: branch 1 offset %d exceeds table size %d",
					name, i, offset1, len(table))
			}
		}
	}
}

// TestCodebookValueRanges verifies that decoded values fall within expected ranges.
func TestCodebookValueRanges(t *testing.T) {
	// Quad codebooks (1, 2) have signed values -1 to 1
	t.Run("hcb1_2 value range [-1, 1]", func(t *testing.T) {
		for i, entry := range hcb1_2 {
			checkValueRange(t, i, "X", int(entry.X), -1, 1)
			checkValueRange(t, i, "Y", int(entry.Y), -1, 1)
			checkValueRange(t, i, "V", int(entry.V), -1, 1)
			checkValueRange(t, i, "W", int(entry.W), -1, 1)
		}
	})

	t.Run("hcb2_2 value range [-1, 1]", func(t *testing.T) {
		for i, entry := range hcb2_2 {
			checkValueRange(t, i, "X", int(entry.X), -1, 1)
			checkValueRange(t, i, "Y", int(entry.Y), -1, 1)
			checkValueRange(t, i, "V", int(entry.V), -1, 1)
			checkValueRange(t, i, "W", int(entry.W), -1, 1)
		}
	})

	// Codebook 3 has unsigned values 0 to 2 (binary quad)
	t.Run("hcb3 value range [0, 2]", func(t *testing.T) {
		for i, entry := range hcb3 {
			if entry.IsLeaf == 1 {
				for j, v := range entry.Data {
					checkValueRange(t, i, string(rune('A'+j)), int(v), 0, 2)
				}
			}
		}
	})

	// Codebook 4 has unsigned values 0 to 2
	t.Run("hcb4_2 value range [0, 2]", func(t *testing.T) {
		for i, entry := range hcb4_2 {
			checkValueRange(t, i, "X", int(entry.X), 0, 2)
			checkValueRange(t, i, "Y", int(entry.Y), 0, 2)
			checkValueRange(t, i, "V", int(entry.V), 0, 2)
			checkValueRange(t, i, "W", int(entry.W), 0, 2)
		}
	})

	// Codebook 5 has signed values -4 to 4
	t.Run("hcb5 value range [-4, 4]", func(t *testing.T) {
		for i, entry := range hcb5 {
			if entry.IsLeaf == 1 {
				checkValueRange(t, i, "X", int(entry.Data[0]), -4, 4)
				checkValueRange(t, i, "Y", int(entry.Data[1]), -4, 4)
			}
		}
	})

	// Codebook 6 has signed values -4 to 4
	t.Run("hcb6_2 value range [-4, 4]", func(t *testing.T) {
		for i, entry := range hcb6_2 {
			checkValueRange(t, i, "X", int(entry.X), -4, 4)
			checkValueRange(t, i, "Y", int(entry.Y), -4, 4)
		}
	})

	// Codebook 7 has unsigned values 0 to 7
	t.Run("hcb7 value range [0, 7]", func(t *testing.T) {
		for i, entry := range hcb7 {
			if entry.IsLeaf == 1 {
				checkValueRange(t, i, "X", int(entry.Data[0]), 0, 7)
				checkValueRange(t, i, "Y", int(entry.Data[1]), 0, 7)
			}
		}
	})

	// Codebook 8 has unsigned values 0 to 7
	t.Run("hcb8_2 value range [0, 7]", func(t *testing.T) {
		for i, entry := range hcb8_2 {
			checkValueRange(t, i, "X", int(entry.X), 0, 7)
			checkValueRange(t, i, "Y", int(entry.Y), 0, 7)
		}
	})

	// Codebook 9 has unsigned values 0 to 12
	t.Run("hcb9 value range [0, 12]", func(t *testing.T) {
		for i, entry := range hcb9 {
			if entry.IsLeaf == 1 {
				checkValueRange(t, i, "X", int(entry.Data[0]), 0, 12)
				checkValueRange(t, i, "Y", int(entry.Data[1]), 0, 12)
			}
		}
	})

	// Codebook 10 has unsigned values 0 to 12
	t.Run("hcb10_2 value range [0, 12]", func(t *testing.T) {
		for i, entry := range hcb10_2 {
			checkValueRange(t, i, "X", int(entry.X), 0, 12)
			checkValueRange(t, i, "Y", int(entry.Y), 0, 12)
		}
	})

	// Codebook 11 has unsigned values 0 to 16 (16 is escape)
	t.Run("hcb11_2 value range [0, 16]", func(t *testing.T) {
		for i, entry := range hcb11_2 {
			checkValueRange(t, i, "X", int(entry.X), 0, 16)
			checkValueRange(t, i, "Y", int(entry.Y), 0, 16)
		}
	})

	// Scale factors have values 0 to 120 (when leaf node)
	t.Run("hcbSF value range [0, 120]", func(t *testing.T) {
		for i, entry := range hcbSF {
			if entry[1] == 0 {
				// Leaf node - first value is the scale factor
				checkValueRange(t, i, "SF", int(entry[0]), 0, 120)
			}
		}
	})
}

func checkValueRange(t *testing.T, index int, field string, value, min, max int) {
	t.Helper()
	if value < min || value > max {
		t.Errorf("entry %d field %s: value %d not in range [%d, %d]",
			index, field, value, min, max)
	}
}
