// Package huffman implements AAC Huffman decoding.
package huffman

import (
	"errors"

	"github.com/llehouerou/go-aac/internal/bits"
)

// ErrEscapeSequence indicates a malformed escape sequence in spectral data.
var ErrEscapeSequence = errors.New("huffman: invalid escape sequence")

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

// signBits reads sign bits for non-zero spectral coefficients.
// For each non-zero value in sp, reads 1 bit: if 1, negates the value.
//
// Ported from: huffman_sign_bits() in ~/dev/faad2/libfaad/huffman.c:93-108
func signBits(r *bits.Reader, sp []int16) {
	for i := range sp {
		if sp[i] != 0 {
			if r.Get1Bit()&1 != 0 {
				sp[i] = -sp[i]
			}
		}
	}
}

// getEscape decodes an escape code if the value is ±16.
// For escape codebook (11), values of ±16 indicate more bits follow.
// Returns error if escape sequence is malformed.
//
// Format: N ones followed by a zero (N >= 4), then N bits of magnitude.
// Final value = (1 << N) | magnitude_bits
//
// Ported from: huffman_getescape() in ~/dev/faad2/libfaad/huffman.c:110-148
func getEscape(r *bits.Reader, sp *int16) error {
	x := *sp

	// Check if this is an escape value
	var neg bool
	if x < 0 {
		if x != -16 {
			return nil // Not an escape
		}
		neg = true
	} else {
		if x != 16 {
			return nil // Not an escape
		}
		neg = false
	}

	// Count leading ones (starting from i=4, since 16 = 2^4)
	var i uint
	for i = 4; i < 16; i++ {
		if r.Get1Bit() == 0 {
			break
		}
	}
	if i >= 16 {
		return ErrEscapeSequence
	}

	// Read i bits for the offset
	off := int16(r.GetBits(i))
	j := off | (1 << i)

	if neg {
		j = -j
	}

	*sp = j
	return nil
}

// decode2StepQuad decodes a quadruple (4 values) using 2-step table lookup.
// Used for codebooks 1, 2, and 4.
//
// Step 1: Read root_bits (5) and lookup in first table
// Step 2: If extra_bits > 0, read more bits and lookup in second table
//
// Ported from: huffman_2step_quad() in ~/dev/faad2/libfaad/huffman.c:150-188
func decode2StepQuad(cb uint8, r *bits.Reader, sp []int16) error {
	root := HCBTable[cb]
	rootBits := HCBN[cb]
	table := HCB2QuadTable[cb]

	// Read first-step bits and lookup
	cw := r.ShowBits(uint(rootBits))
	offset := uint16((*root)[cw].Offset)
	extraBits := (*root)[cw].ExtraBits

	if extraBits != 0 {
		// Need more bits - flush root bits, read extra, adjust offset
		r.FlushBits(uint(rootBits))
		offset += uint16(r.ShowBits(uint(extraBits)))
		r.FlushBits(uint((*table)[offset].Bits) - uint(rootBits))
	} else {
		// All bits in root lookup
		r.FlushBits(uint((*table)[offset].Bits))
	}

	// Extract the four values
	sp[0] = int16((*table)[offset].X)
	sp[1] = int16((*table)[offset].Y)
	sp[2] = int16((*table)[offset].V)
	sp[3] = int16((*table)[offset].W)

	return nil
}

// decode2StepPair decodes a pair (2 values) using 2-step table lookup.
// Used for codebooks 6, 8, 10, and 11.
//
// Step 1: Read root_bits (5 or 6) and lookup in first table
// Step 2: If extra_bits > 0, read more bits and lookup in second table
//
// Ported from: huffman_2step_pair() in ~/dev/faad2/libfaad/huffman.c:198-234
func decode2StepPair(cb uint8, r *bits.Reader, sp []int16) error {
	root := HCBTable[cb]
	rootBits := HCBN[cb]
	table := HCB2PairTable[cb]

	// Read first-step bits and lookup
	cw := r.ShowBits(uint(rootBits))
	offset := uint16((*root)[cw].Offset)
	extraBits := (*root)[cw].ExtraBits

	if extraBits != 0 {
		// Need more bits - flush root bits, read extra, adjust offset
		r.FlushBits(uint(rootBits))
		offset += uint16(r.ShowBits(uint(extraBits)))
		r.FlushBits(uint((*table)[offset].Bits) - uint(rootBits))
	} else {
		// All bits in root lookup
		r.FlushBits(uint((*table)[offset].Bits))
	}

	// Extract the two values
	sp[0] = int16((*table)[offset].X)
	sp[1] = int16((*table)[offset].Y)

	return nil
}
