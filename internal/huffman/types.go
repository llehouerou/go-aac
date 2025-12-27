// Package huffman implements AAC Huffman decoding.
package huffman

// HCB is a first-step table entry for 2-step Huffman decoding.
// It maps initial bits to an offset in the second-step table and
// indicates how many additional bits to read.
//
// Ported from: hcb struct in ~/dev/faad2/libfaad/codebook/hcb.h:85-89
type HCB struct {
	Offset    uint8 // Index into second-step table
	ExtraBits uint8 // Number of additional bits to read
}

// HCB2Quad is a second-step table entry for quadruple codebooks.
// Used by codebooks 1, 2, 4 which output 4 spectral coefficients.
//
// Ported from: hcb_2_quad struct in ~/dev/faad2/libfaad/codebook/hcb.h:99-106
type HCB2Quad struct {
	Bits uint8 // Total codeword length
	X    int8  // First coefficient
	Y    int8  // Second coefficient
	V    int8  // Third coefficient
	W    int8  // Fourth coefficient
}

// HCB2Pair is a second-step table entry for pair codebooks.
// Used by codebooks 6, 8, 10, 11 which output 2 spectral coefficients.
//
// Ported from: hcb_2_pair struct in ~/dev/faad2/libfaad/codebook/hcb.h:92-97
type HCB2Pair struct {
	Bits uint8 // Total codeword length
	X    int8  // First coefficient
	Y    int8  // Second coefficient
}

// HCBBinQuad is a binary search tree node for quadruple codebooks.
// Used by codebook 3 which outputs 4 spectral coefficients.
//
// Ported from: hcb_bin_quad struct in ~/dev/faad2/libfaad/codebook/hcb.h:109-113
type HCBBinQuad struct {
	IsLeaf uint8   // 1 if leaf node with data, 0 if internal with branch offsets
	Data   [4]int8 // Leaf: output values; Internal: branch offsets in data[0], data[1]
}

// HCBBinPair is a binary search tree node for pair codebooks.
// Used by codebooks 5, 7, 9 which output 2 spectral coefficients.
//
// Ported from: hcb_bin_pair struct in ~/dev/faad2/libfaad/codebook/hcb.h:115-119
type HCBBinPair struct {
	IsLeaf uint8   // 1 if leaf node with data, 0 if internal with branch offsets
	Data   [2]int8 // Leaf: output values; Internal: branch offsets
}
