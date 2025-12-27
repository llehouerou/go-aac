// Package huffman implements AAC Huffman decoding.
package huffman

import "testing"

// TestHCBN verifies the first-step bit count table.
// This table indicates how many bits are used for the first lookup step.
// Ported from: ~/dev/faad2/libfaad/huffman.c:75-76 (hcbN)
func TestHCBN(t *testing.T) {
	// Expected values from FAAD2
	// Index:    0   1   2   3   4   5   6   7   8   9  10  11
	expected := [12]uint8{0, 5, 5, 0, 5, 0, 5, 0, 5, 0, 6, 5}

	if len(HCBN) != len(expected) {
		t.Fatalf("HCBN size = %d, want %d", len(HCBN), len(expected))
	}

	for i, want := range expected {
		if HCBN[i] != want {
			t.Errorf("HCBN[%d] = %d, want %d", i, HCBN[i], want)
		}
	}
}

// TestHCBTable verifies the first-step table lookup.
// Maps codebook index to first-step lookup table for 2-step codebooks.
// Ported from: ~/dev/faad2/libfaad/huffman.c:77-78 (hcb_table)
func TestHCBTable(t *testing.T) {
	// Non-nil entries (codebooks with 2-step lookup)
	nonNilEntries := map[int]int{
		1:  32, // hcb1_1[32]
		2:  32, // hcb2_1[32]
		4:  32, // hcb4_1[32]
		6:  32, // hcb6_1[32]
		8:  32, // hcb8_1[32]
		10: 64, // hcb10_1[64]
		11: 32, // hcb11_1[32]
	}

	// Nil entries (codebooks with binary search)
	nilEntries := []int{0, 3, 5, 7, 9}

	// Verify non-nil entries have correct size
	for idx, expectedLen := range nonNilEntries {
		if HCBTable[idx] == nil {
			t.Errorf("HCBTable[%d] is nil, expected non-nil table", idx)
			continue
		}
		if len(*HCBTable[idx]) != expectedLen {
			t.Errorf("HCBTable[%d] len = %d, want %d", idx, len(*HCBTable[idx]), expectedLen)
		}
	}

	// Verify nil entries
	for _, idx := range nilEntries {
		if HCBTable[idx] != nil {
			t.Errorf("HCBTable[%d] should be nil, got non-nil", idx)
		}
	}
}

// TestHCB2QuadTable verifies the second-step quad table lookup.
// Maps codebook index to second-step quad table for quad codebooks.
// Ported from: ~/dev/faad2/libfaad/huffman.c:79-80 (hcb_2_quad_table)
func TestHCB2QuadTable(t *testing.T) {
	// Non-nil entries (quad codebooks with 2-step lookup)
	nonNilEntries := map[int]int{
		1: 113, // hcb1_2[113]
		2: 85,  // hcb2_2[85]
		4: 184, // hcb4_2[184]
	}

	// Nil entries
	nilEntries := []int{0, 3, 5, 6, 7, 8, 9, 10, 11}

	for idx, expectedLen := range nonNilEntries {
		if HCB2QuadTable[idx] == nil {
			t.Errorf("HCB2QuadTable[%d] is nil, expected non-nil table", idx)
			continue
		}
		if len(*HCB2QuadTable[idx]) != expectedLen {
			t.Errorf("HCB2QuadTable[%d] len = %d, want %d", idx, len(*HCB2QuadTable[idx]), expectedLen)
		}
	}

	for _, idx := range nilEntries {
		if HCB2QuadTable[idx] != nil {
			t.Errorf("HCB2QuadTable[%d] should be nil, got non-nil", idx)
		}
	}
}

// TestHCB2PairTable verifies the second-step pair table lookup.
// Maps codebook index to second-step pair table for pair codebooks.
// Ported from: ~/dev/faad2/libfaad/huffman.c:81-82 (hcb_2_pair_table)
func TestHCB2PairTable(t *testing.T) {
	// Non-nil entries (pair codebooks with 2-step lookup)
	nonNilEntries := map[int]int{
		6:  125, // hcb6_2[125]
		8:  83,  // hcb8_2[83]
		10: 209, // hcb10_2[209]
		11: 374, // hcb11_2[374]
	}

	// Nil entries
	nilEntries := []int{0, 1, 2, 3, 4, 5, 7, 9}

	for idx, expectedLen := range nonNilEntries {
		if HCB2PairTable[idx] == nil {
			t.Errorf("HCB2PairTable[%d] is nil, expected non-nil table", idx)
			continue
		}
		if len(*HCB2PairTable[idx]) != expectedLen {
			t.Errorf("HCB2PairTable[%d] len = %d, want %d", idx, len(*HCB2PairTable[idx]), expectedLen)
		}
	}

	for _, idx := range nilEntries {
		if HCB2PairTable[idx] != nil {
			t.Errorf("HCB2PairTable[%d] should be nil, got non-nil", idx)
		}
	}
}

// TestHCBBinPairTable verifies the binary search pair table lookup.
// Maps codebook index to binary pair table for binary-search pair codebooks.
// Ported from: ~/dev/faad2/libfaad/huffman.c:83-84 (hcb_bin_table)
func TestHCBBinPairTable(t *testing.T) {
	// Non-nil entries (pair codebooks with binary search)
	nonNilEntries := map[int]int{
		5: 161, // hcb5[161]
		7: 127, // hcb7[127]
		9: 337, // hcb9[337]
	}

	// Nil entries
	nilEntries := []int{0, 1, 2, 3, 4, 6, 8, 10, 11}

	for idx, expectedLen := range nonNilEntries {
		if HCBBinPairTable[idx] == nil {
			t.Errorf("HCBBinPairTable[%d] is nil, expected non-nil table", idx)
			continue
		}
		if len(*HCBBinPairTable[idx]) != expectedLen {
			t.Errorf("HCBBinPairTable[%d] len = %d, want %d", idx, len(*HCBBinPairTable[idx]), expectedLen)
		}
	}

	for _, idx := range nilEntries {
		if HCBBinPairTable[idx] != nil {
			t.Errorf("HCBBinPairTable[%d] should be nil, got non-nil", idx)
		}
	}
}

// TestHCB3 verifies that hcb3 (binary quad) is directly accessible.
// Codebook 3 is the only binary quad codebook and is handled specially.
func TestHCB3(t *testing.T) {
	if HCB3 == nil {
		t.Fatal("HCB3 is nil, expected non-nil table")
	}
	if len(*HCB3) != 161 {
		t.Errorf("HCB3 len = %d, want 161", len(*HCB3))
	}
}

// TestHCBSF verifies that hcbSF (scale factor) is directly accessible.
// Scale factor codebook uses binary search.
func TestHCBSF(t *testing.T) {
	if HCBSF == nil {
		t.Fatal("HCBSF is nil, expected non-nil table")
	}
	if len(*HCBSF) != 241 {
		t.Errorf("HCBSF len = %d, want 241", len(*HCBSF))
	}
}

// TestUnsignedCB verifies the unsigned codebook flags.
// Indicates whether a codebook uses unsigned values.
// Ported from: ~/dev/faad2/libfaad/huffman.c:89-91 (unsigned_cb)
func TestUnsignedCB(t *testing.T) {
	// Expected values from FAAD2 Table 4.6.2
	// Codebooks 3, 4, 7, 8, 9, 10, 11 are unsigned
	// Codebooks 16-31 are unsigned (virtual signed codebooks)
	expected := [32]bool{
		false, false, false, true, true, false, false, true,
		true, true, true, true, false, false, false, false,
		true, true, true, true, true, true, true, true,
		true, true, true, true, true, true, true, true,
	}

	if len(UnsignedCB) != len(expected) {
		t.Fatalf("UnsignedCB size = %d, want %d", len(UnsignedCB), len(expected))
	}

	for i, want := range expected {
		if UnsignedCB[i] != want {
			t.Errorf("UnsignedCB[%d] = %v, want %v", i, UnsignedCB[i], want)
		}
	}
}

// TestTablePointerIntegrity verifies that table pointers point to the correct tables.
func TestTablePointerIntegrity(t *testing.T) {
	// Verify first-step tables point to correct arrays
	if HCBTable[1] != nil && (*HCBTable[1])[0] != hcb1_1[0] {
		t.Error("HCBTable[1] does not point to hcb1_1")
	}
	if HCBTable[2] != nil && (*HCBTable[2])[0] != hcb2_1[0] {
		t.Error("HCBTable[2] does not point to hcb2_1")
	}
	if HCBTable[4] != nil && (*HCBTable[4])[0] != hcb4_1[0] {
		t.Error("HCBTable[4] does not point to hcb4_1")
	}
	if HCBTable[6] != nil && (*HCBTable[6])[0] != hcb6_1[0] {
		t.Error("HCBTable[6] does not point to hcb6_1")
	}
	if HCBTable[8] != nil && (*HCBTable[8])[0] != hcb8_1[0] {
		t.Error("HCBTable[8] does not point to hcb8_1")
	}
	if HCBTable[10] != nil && (*HCBTable[10])[0] != hcb10_1[0] {
		t.Error("HCBTable[10] does not point to hcb10_1")
	}
	if HCBTable[11] != nil && (*HCBTable[11])[0] != hcb11_1[0] {
		t.Error("HCBTable[11] does not point to hcb11_1")
	}

	// Verify second-step quad tables
	if HCB2QuadTable[1] != nil && (*HCB2QuadTable[1])[0] != hcb1_2[0] {
		t.Error("HCB2QuadTable[1] does not point to hcb1_2")
	}
	if HCB2QuadTable[2] != nil && (*HCB2QuadTable[2])[0] != hcb2_2[0] {
		t.Error("HCB2QuadTable[2] does not point to hcb2_2")
	}
	if HCB2QuadTable[4] != nil && (*HCB2QuadTable[4])[0] != hcb4_2[0] {
		t.Error("HCB2QuadTable[4] does not point to hcb4_2")
	}

	// Verify second-step pair tables
	if HCB2PairTable[6] != nil && (*HCB2PairTable[6])[0] != hcb6_2[0] {
		t.Error("HCB2PairTable[6] does not point to hcb6_2")
	}
	if HCB2PairTable[8] != nil && (*HCB2PairTable[8])[0] != hcb8_2[0] {
		t.Error("HCB2PairTable[8] does not point to hcb8_2")
	}
	if HCB2PairTable[10] != nil && (*HCB2PairTable[10])[0] != hcb10_2[0] {
		t.Error("HCB2PairTable[10] does not point to hcb10_2")
	}
	if HCB2PairTable[11] != nil && (*HCB2PairTable[11])[0] != hcb11_2[0] {
		t.Error("HCB2PairTable[11] does not point to hcb11_2")
	}

	// Verify binary pair tables
	if HCBBinPairTable[5] != nil && (*HCBBinPairTable[5])[0] != hcb5[0] {
		t.Error("HCBBinPairTable[5] does not point to hcb5")
	}
	if HCBBinPairTable[7] != nil && (*HCBBinPairTable[7])[0] != hcb7[0] {
		t.Error("HCBBinPairTable[7] does not point to hcb7")
	}
	if HCBBinPairTable[9] != nil && (*HCBBinPairTable[9])[0] != hcb9[0] {
		t.Error("HCBBinPairTable[9] does not point to hcb9")
	}

	// Verify HCB3 and HCBSF
	if HCB3 != nil && (*HCB3)[0] != hcb3[0] {
		t.Error("HCB3 does not point to hcb3")
	}
	if HCBSF != nil && (*HCBSF)[0] != hcbSF[0] {
		t.Error("HCBSF does not point to hcbSF")
	}
}
