// Package huffman implements AAC Huffman decoding.
package huffman

// HCBN contains the first-step bit count for each codebook.
// Index 0 is reserved (no codebook), indices 1-11 are spectral codebooks.
// Value of 0 means the codebook uses binary search instead of 2-step lookup.
//
// Ported from: ~/dev/faad2/libfaad/huffman.c:75-76 (hcbN)
var HCBN = [12]uint8{
	0, // 0: reserved
	5, // 1: hcb1_1 uses 5-bit first step
	5, // 2: hcb2_1 uses 5-bit first step
	0, // 3: binary search (no 2-step)
	5, // 4: hcb4_1 uses 5-bit first step
	0, // 5: binary search (no 2-step)
	5, // 6: hcb6_1 uses 5-bit first step
	0, // 7: binary search (no 2-step)
	5, // 8: hcb8_1 uses 5-bit first step
	0, // 9: binary search (no 2-step)
	6, // 10: hcb10_1 uses 6-bit first step (unique!)
	5, // 11: hcb11_1 uses 5-bit first step
}

// HCBTable maps codebook index to first-step lookup table.
// Non-nil for 2-step codebooks (1, 2, 4, 6, 8, 10, 11).
// Nil for binary search codebooks (0, 3, 5, 7, 9).
//
// Ported from: ~/dev/faad2/libfaad/huffman.c:77-78 (hcb_table)
var HCBTable = [12]*[]HCB{
	nil,        // 0: reserved
	sliceHCB1,  // 1
	sliceHCB2,  // 2
	nil,        // 3: binary search
	sliceHCB4,  // 4
	nil,        // 5: binary search
	sliceHCB6,  // 6
	nil,        // 7: binary search
	sliceHCB8,  // 8
	nil,        // 9: binary search
	sliceHCB10, // 10
	sliceHCB11, // 11
}

// HCB2QuadTable maps codebook index to second-step quad table.
// Non-nil for quad codebooks with 2-step lookup (1, 2, 4).
// Nil for all others.
//
// Ported from: ~/dev/faad2/libfaad/huffman.c:79-80 (hcb_2_quad_table)
var HCB2QuadTable = [12]*[]HCB2Quad{
	nil,            // 0: reserved
	sliceHCB2Quad1, // 1
	sliceHCB2Quad2, // 2
	nil,            // 3: binary search
	sliceHCB2Quad4, // 4
	nil,            // 5: binary search
	nil,            // 6: pair codebook
	nil,            // 7: binary search
	nil,            // 8: pair codebook
	nil,            // 9: binary search
	nil,            // 10: pair codebook
	nil,            // 11: pair codebook
}

// HCB2PairTable maps codebook index to second-step pair table.
// Non-nil for pair codebooks with 2-step lookup (6, 8, 10, 11).
// Nil for all others.
//
// Ported from: ~/dev/faad2/libfaad/huffman.c:81-82 (hcb_2_pair_table)
var HCB2PairTable = [12]*[]HCB2Pair{
	nil,             // 0: reserved
	nil,             // 1: quad codebook
	nil,             // 2: quad codebook
	nil,             // 3: binary search
	nil,             // 4: quad codebook
	nil,             // 5: binary search
	sliceHCB2Pair6,  // 6
	nil,             // 7: binary search
	sliceHCB2Pair8,  // 8
	nil,             // 9: binary search
	sliceHCB2Pair10, // 10
	sliceHCB2Pair11, // 11
}

// HCBBinPairTable maps codebook index to binary pair table.
// Non-nil for pair codebooks with binary search (5, 7, 9).
// Nil for all others.
//
// Ported from: ~/dev/faad2/libfaad/huffman.c:83-84 (hcb_bin_table)
var HCBBinPairTable = [12]*[]HCBBinPair{
	nil,           // 0: reserved
	nil,           // 1: 2-step quad
	nil,           // 2: 2-step quad
	nil,           // 3: binary quad (handled separately)
	nil,           // 4: 2-step quad
	sliceBinPair5, // 5
	nil,           // 6: 2-step pair
	sliceBinPair7, // 7
	nil,           // 8: 2-step pair
	sliceBinPair9, // 9
	nil,           // 10: 2-step pair
	nil,           // 11: 2-step pair
}

// HCB3 is the binary quad table for codebook 3.
// This is the only binary quad codebook and is handled specially.
//
// Ported from: ~/dev/faad2/libfaad/huffman.c:85 comment
var HCB3 = sliceBinQuad3

// HCBSF is the binary search table for scale factors.
// Uses a different structure than spectral codebooks.
//
// Ported from: ~/dev/faad2/libfaad/huffman.c:64 (hcb_sf)
var HCBSF = sliceSF

// UnsignedCB indicates whether a codebook uses unsigned values.
// If true, sign bits must be read separately for non-zero values.
// Codebooks 3, 4, 7, 8, 9, 10, 11 are unsigned.
// Codebooks 16-31 are virtual signed codebooks (mapped to unsigned).
//
// Ported from: ~/dev/faad2/libfaad/huffman.c:89-91 (unsigned_cb)
var UnsignedCB = [32]bool{
	false, false, false, true, true, false, false, true, // 0-7
	true, true, true, true, false, false, false, false, // 8-15
	true, true, true, true, true, true, true, true, // 16-23
	true, true, true, true, true, true, true, true, // 24-31
}

// Helper slices - these point to the underlying arrays
// Using slices allows us to use pointer-to-slice for the lookup tables

var (
	// First-step tables (2-step codebooks)
	sliceHCB1  = toHCBSlice(hcb1_1[:])
	sliceHCB2  = toHCBSlice(hcb2_1[:])
	sliceHCB4  = toHCBSlice(hcb4_1[:])
	sliceHCB6  = toHCBSlice(hcb6_1[:])
	sliceHCB8  = toHCBSlice(hcb8_1[:])
	sliceHCB10 = toHCBSlice(hcb10_1[:])
	sliceHCB11 = toHCBSlice(hcb11_1[:])

	// Second-step quad tables
	sliceHCB2Quad1 = toHCB2QuadSlice(hcb1_2[:])
	sliceHCB2Quad2 = toHCB2QuadSlice(hcb2_2[:])
	sliceHCB2Quad4 = toHCB2QuadSlice(hcb4_2[:])

	// Second-step pair tables
	sliceHCB2Pair6  = toHCB2PairSlice(hcb6_2[:])
	sliceHCB2Pair8  = toHCB2PairSlice(hcb8_2[:])
	sliceHCB2Pair10 = toHCB2PairSlice(hcb10_2[:])
	sliceHCB2Pair11 = toHCB2PairSlice(hcb11_2[:])

	// Binary pair tables
	sliceBinPair5 = toHCBBinPairSlice(hcb5[:])
	sliceBinPair7 = toHCBBinPairSlice(hcb7[:])
	sliceBinPair9 = toHCBBinPairSlice(hcb9[:])

	// Binary quad table (only codebook 3)
	sliceBinQuad3 = toHCBBinQuadSlice(hcb3[:])

	// Scale factor table
	sliceSF = toSFSlice(hcbSF[:])
)

// Helper functions to create slice pointers

func toHCBSlice(s []HCB) *[]HCB {
	return &s
}

func toHCB2QuadSlice(s []HCB2Quad) *[]HCB2Quad {
	return &s
}

func toHCB2PairSlice(s []HCB2Pair) *[]HCB2Pair {
	return &s
}

func toHCBBinPairSlice(s []HCBBinPair) *[]HCBBinPair {
	return &s
}

func toHCBBinQuadSlice(s []HCBBinQuad) *[]HCBBinQuad {
	return &s
}

func toSFSlice(s [][2]uint8) *[][2]uint8 {
	return &s
}
