// Package huffman implements AAC Huffman decoding.
package huffman

import "github.com/llehouerou/go-aac/internal/bits"

// ScaleFactor decodes a scale factor delta from the bitstream.
// Returns a value in the range [-60, 60] representing the delta
// from the previous scale factor.
//
// The scale factor codebook uses binary search: start at offset 0,
// read one bit at a time, and follow branches until reaching a leaf.
//
// Ported from: huffman_scale_factor() in ~/dev/faad2/libfaad/huffman.c:60-72
func ScaleFactor(r *bits.Reader) int8 {
	offset := uint16(0)
	sf := *HCBSF

	// Traverse binary tree until we hit a leaf (branch offset = 0)
	for sf[offset][1] != 0 {
		b := r.Get1Bit()
		offset += uint16(sf[offset][b])
	}

	// Return the value at the leaf node, adjusted to signed delta
	return int8(sf[offset][0]) - 60
}
