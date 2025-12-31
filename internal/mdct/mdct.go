package mdct

import "github.com/llehouerou/go-aac/internal/fft"

// MDCT holds state for Modified Discrete Cosine Transform operations.
//
// Ported from: mdct_info struct in ~/dev/faad2/libfaad/structs.h:57-65
type MDCT struct {
	N      uint16        // Transform size (256 or 2048 for AAC)
	N2     uint16        // N/2
	N4     uint16        // N/4
	N8     uint16        // N/8
	cfft   *fft.CFFT     // Complex FFT of size N/4
	sincos []fft.Complex // Pre/post twiddle factors (N/4 entries)
}

// NewMDCT creates and initializes an MDCT for the given transform size.
// N must be divisible by 8.
//
// Ported from: faad_mdct_init() in ~/dev/faad2/libfaad/mdct.c:62-105
func NewMDCT(n uint16) *MDCT {
	if n%8 != 0 {
		panic("MDCT size must be divisible by 8")
	}

	m := &MDCT{
		N:  n,
		N2: n >> 1,
		N4: n >> 2,
		N8: n >> 3,
	}

	// Initialize FFT for size N/4
	m.cfft = fft.NewCFFT(m.N4)

	// Get precomputed sincos table
	m.sincos = getSinCosTable(n)

	return m
}

// getSinCosTable returns the precomputed sincos twiddle table for the given size.
// Returns nil for unsupported sizes.
func getSinCosTable(n uint16) []fft.Complex {
	switch n {
	case 2048:
		return mdctTab2048[:]
	case 256:
		return mdctTab256[:]
	default:
		return nil
	}
}
