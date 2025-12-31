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
	if m.sincos == nil {
		panic("MDCT size not supported: no twiddle table available")
	}

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

// IMDCT performs the Inverse Modified Discrete Cosine Transform.
// Input: N/2 frequency coefficients (xIn)
// Output: N time samples (xOut)
//
// The algorithm uses FFT-based computation:
// 1. Pre-IFFT complex multiplication
// 2. Complex IFFT of size N/4
// 3. Post-IFFT complex multiplication
// 4. Reordering
//
// Ported from: faad_imdct() in ~/dev/faad2/libfaad/mdct.c:122-228
func (m *MDCT) IMDCT(xIn []float32, xOut []float32) {
	n2 := m.N2
	n4 := m.N4
	n8 := m.N8
	sincos := m.sincos

	// Allocate work buffer for complex FFT
	z1 := make([]fft.Complex, n4)

	// Pre-IFFT complex multiplication
	// Z1[k] = ComplexMult(X_in[2*k], X_in[N/2-1-2*k], sincos[k])
	for k := uint16(0); k < n4; k++ {
		// ComplexMult computes:
		//   y1 = x1*c1 + x2*c2
		//   y2 = x2*c1 - x1*c2
		x1 := xIn[2*k]
		x2 := xIn[n2-1-2*k]
		c1 := sincos[k].Re
		c2 := sincos[k].Im

		z1[k].Im, z1[k].Re = fft.ComplexMult(x1, x2, c1, c2)
	}

	// Complex IFFT
	m.cfft.Backward(z1)

	// Post-IFFT complex multiplication
	for k := uint16(0); k < n4; k++ {
		re := z1[k].Re
		im := z1[k].Im
		c1 := sincos[k].Re
		c2 := sincos[k].Im

		z1[k].Im, z1[k].Re = fft.ComplexMult(im, re, c1, c2)
	}

	// Reordering
	// Ported from: faad_imdct() reordering in mdct.c:196-221
	for k := uint16(0); k < n8; k += 2 {
		xOut[2*k] = z1[n8+k].Im
		xOut[2+2*k] = z1[n8+1+k].Im

		xOut[1+2*k] = -z1[n8-1-k].Re
		xOut[3+2*k] = -z1[n8-2-k].Re

		xOut[n4+2*k] = z1[k].Re
		xOut[n4+2+2*k] = z1[1+k].Re

		xOut[n4+1+2*k] = -z1[n4-1-k].Im
		xOut[n4+3+2*k] = -z1[n4-2-k].Im

		xOut[n2+2*k] = z1[n8+k].Re
		xOut[n2+2+2*k] = z1[n8+1+k].Re

		xOut[n2+1+2*k] = -z1[n8-1-k].Im
		xOut[n2+3+2*k] = -z1[n8-2-k].Im

		xOut[n2+n4+2*k] = -z1[k].Im
		xOut[n2+n4+2+2*k] = -z1[1+k].Im

		xOut[n2+n4+1+2*k] = z1[n4-1-k].Re
		xOut[n2+n4+3+2*k] = z1[n4-2-k].Re
	}
}
