package fft

import "math"

// CFFT holds state for a complex FFT of a fixed size.
//
// Ported from: cfft_info struct in ~/dev/faad2/libfaad/cfft.h:38-44
type CFFT struct {
	N    uint16     // FFT size
	IFac [15]uint16 // Factorization of N
	Work []Complex  // Work buffer for intermediate results
	Tab  []Complex  // Twiddle factor table
}

// NewCFFT creates and initializes a new CFFT for size n.
// n must be divisible by 8 and only contain factors 2, 3, 4, 5.
//
// Ported from: cffti() in ~/dev/faad2/libfaad/cfft.c:1005-1039
func NewCFFT(n uint16) *CFFT {
	cfft := &CFFT{
		N:    n,
		Work: make([]Complex, n),
		Tab:  make([]Complex, n),
	}

	// Factorize n and compute twiddle factors
	factorize(n, cfft.IFac[:])
	computeTwiddle(n, cfft.Tab, cfft.IFac[:])

	return cfft
}

// factorize computes the factorization of n into factors 2, 3, 4, 5.
// Results are stored in ifac where:
//   - ifac[0] = n
//   - ifac[1] = number of factors
//   - ifac[2..] = the factors
//
// Ported from: cffti1() factorization in ~/dev/faad2/libfaad/cfft.c:906-956
func factorize(n uint16, ifac []uint16) {
	// Factor order: try 3, 4, 2, 5
	ntryh := [4]uint16{3, 4, 2, 5}

	nl := n
	nf := uint16(0)
	j := uint16(0)
	ntry := uint16(0)

startloop:
	j++
	if j <= 4 {
		ntry = ntryh[j-1]
	} else {
		ntry += 2
	}

	for {
		nq := nl / ntry
		nr := nl - ntry*nq

		if nr != 0 {
			goto startloop
		}

		nf++
		ifac[nf+1] = ntry
		nl = nq

		// If we found a factor of 2 and it's not the first factor,
		// move it to the front (after any existing 2s)
		if ntry == 2 && nf != 1 {
			for i := uint16(2); i <= nf; i++ {
				ib := nf - i + 2
				ifac[ib+1] = ifac[ib]
			}
			ifac[2] = 2
		}

		if nl == 1 {
			break
		}
	}

	ifac[0] = n
	ifac[1] = nf
}

// computeTwiddle computes the twiddle factor table.
//
// Ported from: cffti1() twiddle computation in ~/dev/faad2/libfaad/cfft.c:957-999
func computeTwiddle(n uint16, wa []Complex, ifac []uint16) {
	nf := ifac[1]
	argh := float64(2.0 * math.Pi / float64(n))

	i := uint16(0)
	l1 := uint16(1)

	for k1 := uint16(1); k1 <= nf; k1++ {
		ip := ifac[k1+1]
		l2 := l1 * ip
		ido := n / l2
		ipm := ip - 1

		for j := uint16(0); j < ipm; j++ {
			i1 := i
			wa[i].Re = 1.0
			wa[i].Im = 0.0

			ld := l1 * (j + 1)
			fi := float64(0)
			argld := float64(ld) * argh

			for ii := uint16(0); ii < ido; ii++ {
				i++
				fi++
				arg := fi * argld
				wa[i].Re = float32(math.Cos(arg))
				wa[i].Im = float32(math.Sin(arg))
			}

			if ip > 5 {
				wa[i1].Re = wa[i].Re
				wa[i1].Im = wa[i].Im
			}
		}
		l1 = l2
	}
}
