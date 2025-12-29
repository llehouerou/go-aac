// Package spectrum implements spectral processing for AAC decoding.
// This file contains the random number generator for PNS.

package spectrum

// parity contains precomputed parity (number of 1-bits mod 2) for bytes 0-255.
// Ported from: Parity[] in ~/dev/faad2/libfaad/common.c:193-202
var parity = [256]uint8{
	0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
	1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
	1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
	0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
	1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
	0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
	0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
	1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
}

// RNG generates a pseudo-random 32-bit value using dual polycounters.
// This is a deterministic RNG suitable for audio purposes with a very long period.
// The state is updated in-place through the r1 and r2 pointers.
//
// Algorithm: Two LFSRs with opposite rotation direction and coprime periods.
// Period = 3*5*17*257*65537 * 7*47*73*178481 = 18,410,713,077,675,721,215
//
// Ported from: ne_rng() in ~/dev/faad2/libfaad/common.c:231-241
func RNG(r1, r2 *uint32) uint32 {
	t1 := *r1
	t2 := *r2
	t3 := t1
	t4 := t2

	// First polycounter: LFSR with taps at bits 0,2,4,5,6,7
	t1 &= 0xF5
	t1 = uint32(parity[t1])
	t1 <<= 31

	// Second polycounter: LFSR with taps at bits 25,26,29,30
	t2 >>= 25
	t2 &= 0x63
	t2 = uint32(parity[t2])

	// Update states
	*r1 = (t3 >> 1) | t1
	*r2 = (t4 + t4) | t2

	// Return XOR of both states
	return *r1 ^ *r2
}
