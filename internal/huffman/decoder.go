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

// decodeBinaryQuad decodes a quadruple using binary search.
// Used only for codebook 3.
//
// Traverse the binary tree by reading one bit at a time until
// reaching a leaf node (IsLeaf = 1), then extract the 4 values.
//
// Ported from: huffman_binary_quad() in ~/dev/faad2/libfaad/huffman.c:244-266
func decodeBinaryQuad(r *bits.Reader, sp []int16) error {
	offset := uint16(0)
	table := *HCB3

	// Traverse until we hit a leaf
	for table[offset].IsLeaf == 0 {
		b := r.Get1Bit()
		offset += uint16(table[offset].Data[b])
	}

	// Extract the four values from the leaf
	sp[0] = int16(table[offset].Data[0])
	sp[1] = int16(table[offset].Data[1])
	sp[2] = int16(table[offset].Data[2])
	sp[3] = int16(table[offset].Data[3])

	return nil
}

// decodeBinaryPair decodes a pair using binary search.
// Used for codebooks 5, 7, and 9.
//
// Ported from: huffman_binary_pair() in ~/dev/faad2/libfaad/huffman.c:276-298
func decodeBinaryPair(cb uint8, r *bits.Reader, sp []int16) error {
	offset := uint16(0)
	table := *HCBBinPairTable[cb]

	// Traverse until we hit a leaf
	for table[offset].IsLeaf == 0 {
		b := r.Get1Bit()
		offset += uint16(table[offset].Data[b])
	}

	// Extract the two values from the leaf
	sp[0] = int16(table[offset].Data[0])
	sp[1] = int16(table[offset].Data[1])

	return nil
}

// vcb11LAV is the Largest Absolute Value table for virtual codebook 11.
// Index is (cb - 16), valid for codebooks 16-31.
//
// Ported from: vcb11_LAV_tab in ~/dev/faad2/libfaad/huffman.c:319-322
var vcb11LAV = [16]int16{
	16, 31, 47, 63, 95, 127, 159, 191,
	223, 255, 319, 383, 511, 767, 1023, 2047,
}

// vcb11CheckLAV checks if values exceed the Largest Absolute Value
// for virtual codebook 11 (codebooks 16-31). If exceeded, zeros them.
//
// This catches errors in escape sequences for VCB11.
//
// Ported from: vcb11_check_LAV() in ~/dev/faad2/libfaad/huffman.c:317-335
func vcb11CheckLAV(cb uint8, sp []int16) {
	if cb < 16 || cb > 31 {
		return
	}

	maxVal := vcb11LAV[cb-16]

	if abs16(sp[0]) > maxVal || abs16(sp[1]) > maxVal {
		sp[0] = 0
		sp[1] = 0
	}
}

// abs16 returns the absolute value of an int16.
func abs16(x int16) int16 {
	if x < 0 {
		return -x
	}
	return x
}

// ErrInvalidCodebook indicates an invalid Huffman codebook index.
var ErrInvalidCodebook = errors.New("huffman: invalid codebook")

// SpectralData decodes spectral coefficients using the specified codebook.
// For quad codebooks (1-4), sp must have at least 4 elements.
// For pair codebooks (5-11), sp must have at least 2 elements.
//
// Returns the decoded values in sp. For unsigned codebooks, sign bits
// are read separately. For escape codebook (11), escape sequences
// are decoded for values of +/-16.
//
// Ported from: huffman_spectral_data() in ~/dev/faad2/libfaad/huffman.c:337-398
func SpectralData(cb uint8, r *bits.Reader, sp []int16) error {
	switch cb {
	case 1, 2: // 2-step quad, signed
		return decode2StepQuad(cb, r, sp)

	case 3: // Binary quad, unsigned
		if err := decodeBinaryQuad(r, sp); err != nil {
			return err
		}
		signBits(r, sp[:QuadLen])
		return nil

	case 4: // 2-step quad, unsigned
		if err := decode2StepQuad(cb, r, sp); err != nil {
			return err
		}
		signBits(r, sp[:QuadLen])
		return nil

	case 5: // Binary pair, signed
		return decodeBinaryPair(cb, r, sp)

	case 6: // 2-step pair, signed
		return decode2StepPair(cb, r, sp)

	case 7, 9: // Binary pair, unsigned
		if err := decodeBinaryPair(cb, r, sp); err != nil {
			return err
		}
		signBits(r, sp[:PairLen])
		return nil

	case 8, 10: // 2-step pair, unsigned
		if err := decode2StepPair(cb, r, sp); err != nil {
			return err
		}
		signBits(r, sp[:PairLen])
		return nil

	case 11: // Escape codebook
		if err := decode2StepPair(11, r, sp); err != nil {
			return err
		}
		signBits(r, sp[:PairLen])
		if err := getEscape(r, &sp[0]); err != nil {
			return err
		}
		return getEscape(r, &sp[1])

	case 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31:
		// VCB11: virtual codebooks using codebook 11
		if err := decode2StepPair(11, r, sp); err != nil {
			return err
		}
		signBits(r, sp[:PairLen])
		if err := getEscape(r, &sp[0]); err != nil {
			return err
		}
		if err := getEscape(r, &sp[1]); err != nil {
			return err
		}
		vcb11CheckLAV(cb, sp)
		return nil

	default:
		return ErrInvalidCodebook
	}
}
