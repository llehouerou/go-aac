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

// passf2pos performs a radix-2 butterfly for backward FFT (isign=+1).
//
// Ported from: passf2pos() in ~/dev/faad2/libfaad/cfft.c:70-123
func passf2pos(ido, l1 uint16, cc, ch []Complex, wa []Complex) {
	// Note: ido=1 case is never reached for supported AAC frame lengths
	// according to FAAD2 comments, so we only implement ido > 1 case.

	for k := uint16(0); k < l1; k++ {
		ah := k * ido
		ac := 2 * k * ido

		for i := uint16(0); i < ido; i++ {
			ch[ah+i].Re = cc[ac+i].Re + cc[ac+i+ido].Re
			t2Re := cc[ac+i].Re - cc[ac+i+ido].Re
			ch[ah+i].Im = cc[ac+i].Im + cc[ac+i+ido].Im
			t2Im := cc[ac+i].Im - cc[ac+i+ido].Im

			// Twiddle factor multiplication
			ch[ah+i+l1*ido].Im, ch[ah+i+l1*ido].Re = ComplexMult(t2Im, t2Re, wa[i].Re, wa[i].Im)
		}
	}
}

// passf2neg performs a radix-2 butterfly for forward FFT (isign=-1).
//
// Ported from: passf2neg() in ~/dev/faad2/libfaad/cfft.c:125-178
func passf2neg(ido, l1 uint16, cc, ch []Complex, wa []Complex) {
	// Note: ido=1 case is never reached for supported AAC frame lengths
	// according to FAAD2 comments, so we only implement ido > 1 case.

	for k := uint16(0); k < l1; k++ {
		ah := k * ido
		ac := 2 * k * ido

		for i := uint16(0); i < ido; i++ {
			ch[ah+i].Re = cc[ac+i].Re + cc[ac+i+ido].Re
			t2Re := cc[ac+i].Re - cc[ac+i+ido].Re
			ch[ah+i].Im = cc[ac+i].Im + cc[ac+i+ido].Im
			t2Im := cc[ac+i].Im - cc[ac+i+ido].Im

			// Twiddle factor multiplication (note different order from passf2pos)
			ch[ah+i+l1*ido].Re, ch[ah+i+l1*ido].Im = ComplexMult(t2Re, t2Im, wa[i].Re, wa[i].Im)
		}
	}
}

// passf4pos performs a radix-4 butterfly for backward FFT (isign=+1).
//
// Ported from: passf4pos() in ~/dev/faad2/libfaad/cfft.c:329-413
func passf4pos(ido, l1 uint16, cc, ch []Complex, wa1, wa2, wa3 []Complex) {
	if ido == 1 {
		for k := uint16(0); k < l1; k++ {
			ac := 4 * k
			ah := k

			t2Re := cc[ac].Re + cc[ac+2].Re
			t1Re := cc[ac].Re - cc[ac+2].Re
			t2Im := cc[ac].Im + cc[ac+2].Im
			t1Im := cc[ac].Im - cc[ac+2].Im
			t3Re := cc[ac+1].Re + cc[ac+3].Re
			t4Im := cc[ac+1].Re - cc[ac+3].Re
			t3Im := cc[ac+3].Im + cc[ac+1].Im
			t4Re := cc[ac+3].Im - cc[ac+1].Im

			ch[ah].Re = t2Re + t3Re
			ch[ah+2*l1].Re = t2Re - t3Re
			ch[ah].Im = t2Im + t3Im
			ch[ah+2*l1].Im = t2Im - t3Im
			ch[ah+l1].Re = t1Re + t4Re
			ch[ah+3*l1].Re = t1Re - t4Re
			ch[ah+l1].Im = t1Im + t4Im
			ch[ah+3*l1].Im = t1Im - t4Im
		}
	} else {
		for k := uint16(0); k < l1; k++ {
			ac := 4 * k * ido
			ah := k * ido

			for i := uint16(0); i < ido; i++ {
				t2Re := cc[ac+i].Re + cc[ac+i+2*ido].Re
				t1Re := cc[ac+i].Re - cc[ac+i+2*ido].Re
				t2Im := cc[ac+i].Im + cc[ac+i+2*ido].Im
				t1Im := cc[ac+i].Im - cc[ac+i+2*ido].Im
				t3Re := cc[ac+i+ido].Re + cc[ac+i+3*ido].Re
				t4Im := cc[ac+i+ido].Re - cc[ac+i+3*ido].Re
				t3Im := cc[ac+i+3*ido].Im + cc[ac+i+ido].Im
				t4Re := cc[ac+i+3*ido].Im - cc[ac+i+ido].Im

				c2Re := t1Re + t4Re
				c4Re := t1Re - t4Re
				c2Im := t1Im + t4Im
				c4Im := t1Im - t4Im

				ch[ah+i].Re = t2Re + t3Re
				c3Re := t2Re - t3Re
				ch[ah+i].Im = t2Im + t3Im
				c3Im := t2Im - t3Im

				// Twiddle factor multiplication
				ch[ah+i+l1*ido].Im, ch[ah+i+l1*ido].Re = ComplexMult(c2Im, c2Re, wa1[i].Re, wa1[i].Im)
				ch[ah+i+2*l1*ido].Im, ch[ah+i+2*l1*ido].Re = ComplexMult(c3Im, c3Re, wa2[i].Re, wa2[i].Im)
				ch[ah+i+3*l1*ido].Im, ch[ah+i+3*l1*ido].Re = ComplexMult(c4Im, c4Re, wa3[i].Re, wa3[i].Im)
			}
		}
	}
}

// Trigonometric constants for radix-3 FFT.
// Ported from: passf3() in ~/dev/faad2/libfaad/cfft.c:185-186
const (
	taur3 = float32(-0.5)
	taui3 = float32(0.866025403784439) // sqrt(3)/2
)

// Trigonometric constants for radix-5 FFT.
// Ported from: passf5() in ~/dev/faad2/libfaad/cfft.c:507-510
const (
	tr11 = float32(0.309016994374947)  // cos(2*pi/5)
	ti11 = float32(0.951056516295154)  // sin(2*pi/5)
	tr12 = float32(-0.809016994374947) // cos(4*pi/5)
	ti12 = float32(0.587785252292473)  // sin(4*pi/5)
)

// passf4neg performs a radix-4 butterfly for forward FFT (isign=-1).
//
// Ported from: passf4neg() in ~/dev/faad2/libfaad/cfft.c:416-501
func passf4neg(ido, l1 uint16, cc, ch []Complex, wa1, wa2, wa3 []Complex) {
	if ido == 1 {
		for k := uint16(0); k < l1; k++ {
			ac := 4 * k
			ah := k

			t2Re := cc[ac].Re + cc[ac+2].Re
			t1Re := cc[ac].Re - cc[ac+2].Re
			t2Im := cc[ac].Im + cc[ac+2].Im
			t1Im := cc[ac].Im - cc[ac+2].Im
			t3Re := cc[ac+1].Re + cc[ac+3].Re
			t4Im := cc[ac+1].Re - cc[ac+3].Re
			t3Im := cc[ac+3].Im + cc[ac+1].Im
			t4Re := cc[ac+3].Im - cc[ac+1].Im

			ch[ah].Re = t2Re + t3Re
			ch[ah+2*l1].Re = t2Re - t3Re
			ch[ah].Im = t2Im + t3Im
			ch[ah+2*l1].Im = t2Im - t3Im
			// Note: signs differ from passf4pos
			ch[ah+l1].Re = t1Re - t4Re
			ch[ah+3*l1].Re = t1Re + t4Re
			ch[ah+l1].Im = t1Im - t4Im
			ch[ah+3*l1].Im = t1Im + t4Im
		}
	} else {
		for k := uint16(0); k < l1; k++ {
			ac := 4 * k * ido
			ah := k * ido

			for i := uint16(0); i < ido; i++ {
				t2Re := cc[ac+i].Re + cc[ac+i+2*ido].Re
				t1Re := cc[ac+i].Re - cc[ac+i+2*ido].Re
				t2Im := cc[ac+i].Im + cc[ac+i+2*ido].Im
				t1Im := cc[ac+i].Im - cc[ac+i+2*ido].Im
				t3Re := cc[ac+i+ido].Re + cc[ac+i+3*ido].Re
				t4Im := cc[ac+i+ido].Re - cc[ac+i+3*ido].Re
				t3Im := cc[ac+i+3*ido].Im + cc[ac+i+ido].Im
				t4Re := cc[ac+i+3*ido].Im - cc[ac+i+ido].Im

				// Note: signs differ from passf4pos
				c2Re := t1Re - t4Re
				c4Re := t1Re + t4Re
				c2Im := t1Im - t4Im
				c4Im := t1Im + t4Im

				ch[ah+i].Re = t2Re + t3Re
				c3Re := t2Re - t3Re
				ch[ah+i].Im = t2Im + t3Im
				c3Im := t2Im - t3Im

				// Twiddle factor multiplication (note different order from passf4pos)
				ch[ah+i+l1*ido].Re, ch[ah+i+l1*ido].Im = ComplexMult(c2Re, c2Im, wa1[i].Re, wa1[i].Im)
				ch[ah+i+2*l1*ido].Re, ch[ah+i+2*l1*ido].Im = ComplexMult(c3Re, c3Im, wa2[i].Re, wa2[i].Im)
				ch[ah+i+3*l1*ido].Re, ch[ah+i+3*l1*ido].Im = ComplexMult(c4Re, c4Im, wa3[i].Re, wa3[i].Im)
			}
		}
	}
}

// passf3 performs a radix-3 butterfly for both forward and backward FFT.
//
// Ported from: passf3() in ~/dev/faad2/libfaad/cfft.c:181-326
func passf3(ido, l1 uint16, cc, ch []Complex, wa1, wa2 []Complex, isign int8) {
	// Note: ido=1 case is never reached for supported AAC frame lengths
	// according to FAAD2 comments.

	if isign == 1 {
		// Backward FFT
		for k := uint16(0); k < l1; k++ {
			for i := uint16(0); i < ido; i++ {
				ac := i + (3*k+1)*ido
				ah := i + k*ido

				t2Re := cc[ac].Re + cc[ac+ido].Re
				c2Re := cc[ac-ido].Re + taur3*t2Re
				t2Im := cc[ac].Im + cc[ac+ido].Im
				c2Im := cc[ac-ido].Im + taur3*t2Im

				ch[ah].Re = cc[ac-ido].Re + t2Re
				ch[ah].Im = cc[ac-ido].Im + t2Im

				c3Re := taui3 * (cc[ac].Re - cc[ac+ido].Re)
				c3Im := taui3 * (cc[ac].Im - cc[ac+ido].Im)

				d2Re := c2Re - c3Im
				d3Im := c2Im - c3Re
				d3Re := c2Re + c3Im
				d2Im := c2Im + c3Re

				ch[ah+l1*ido].Im, ch[ah+l1*ido].Re = ComplexMult(d2Im, d2Re, wa1[i].Re, wa1[i].Im)
				ch[ah+2*l1*ido].Im, ch[ah+2*l1*ido].Re = ComplexMult(d3Im, d3Re, wa2[i].Re, wa2[i].Im)
			}
		}
	} else {
		// Forward FFT
		for k := uint16(0); k < l1; k++ {
			for i := uint16(0); i < ido; i++ {
				ac := i + (3*k+1)*ido
				ah := i + k*ido

				t2Re := cc[ac].Re + cc[ac+ido].Re
				c2Re := cc[ac-ido].Re + taur3*t2Re
				t2Im := cc[ac].Im + cc[ac+ido].Im
				c2Im := cc[ac-ido].Im + taur3*t2Im

				ch[ah].Re = cc[ac-ido].Re + t2Re
				ch[ah].Im = cc[ac-ido].Im + t2Im

				c3Re := taui3 * (cc[ac].Re - cc[ac+ido].Re)
				c3Im := taui3 * (cc[ac].Im - cc[ac+ido].Im)

				d2Re := c2Re + c3Im
				d3Im := c2Im + c3Re
				d3Re := c2Re - c3Im
				d2Im := c2Im - c3Re

				ch[ah+l1*ido].Re, ch[ah+l1*ido].Im = ComplexMult(d2Re, d2Im, wa1[i].Re, wa1[i].Im)
				ch[ah+2*l1*ido].Re, ch[ah+2*l1*ido].Im = ComplexMult(d3Re, d3Im, wa2[i].Re, wa2[i].Im)
			}
		}
	}
}

// passf5 performs a radix-5 butterfly for both forward and backward FFT.
//
// Ported from: passf5() in ~/dev/faad2/libfaad/cfft.c:503-733
func passf5(ido, l1 uint16, cc, ch []Complex, wa1, wa2, wa3, wa4 []Complex, isign int8) {
	// Note: For AAC, radix-5 with ido=1 is the common case (5 is always the largest factor)

	if ido == 1 {
		if isign == 1 {
			// Backward FFT
			for k := uint16(0); k < l1; k++ {
				ac := 5*k + 1
				ah := k

				t2Re := cc[ac].Re + cc[ac+3].Re
				t2Im := cc[ac].Im + cc[ac+3].Im
				t3Re := cc[ac+1].Re + cc[ac+2].Re
				t3Im := cc[ac+1].Im + cc[ac+2].Im
				t4Re := cc[ac+1].Re - cc[ac+2].Re
				t4Im := cc[ac+1].Im - cc[ac+2].Im
				t5Re := cc[ac].Re - cc[ac+3].Re
				t5Im := cc[ac].Im - cc[ac+3].Im

				ch[ah].Re = cc[ac-1].Re + t2Re + t3Re
				ch[ah].Im = cc[ac-1].Im + t2Im + t3Im

				c2Re := cc[ac-1].Re + tr11*t2Re + tr12*t3Re
				c2Im := cc[ac-1].Im + tr11*t2Im + tr12*t3Im
				c3Re := cc[ac-1].Re + tr12*t2Re + tr11*t3Re
				c3Im := cc[ac-1].Im + tr12*t2Im + tr11*t3Im

				c5Re, c4Re := ComplexMult(ti11, ti12, t5Re, t4Re)
				c5Im, c4Im := ComplexMult(ti11, ti12, t5Im, t4Im)

				ch[ah+l1].Re = c2Re - c5Im
				ch[ah+l1].Im = c2Im + c5Re
				ch[ah+2*l1].Re = c3Re - c4Im
				ch[ah+2*l1].Im = c3Im + c4Re
				ch[ah+3*l1].Re = c3Re + c4Im
				ch[ah+3*l1].Im = c3Im - c4Re
				ch[ah+4*l1].Re = c2Re + c5Im
				ch[ah+4*l1].Im = c2Im - c5Re
			}
		} else {
			// Forward FFT
			for k := uint16(0); k < l1; k++ {
				ac := 5*k + 1
				ah := k

				t2Re := cc[ac].Re + cc[ac+3].Re
				t2Im := cc[ac].Im + cc[ac+3].Im
				t3Re := cc[ac+1].Re + cc[ac+2].Re
				t3Im := cc[ac+1].Im + cc[ac+2].Im
				t4Re := cc[ac+1].Re - cc[ac+2].Re
				t4Im := cc[ac+1].Im - cc[ac+2].Im
				t5Re := cc[ac].Re - cc[ac+3].Re
				t5Im := cc[ac].Im - cc[ac+3].Im

				ch[ah].Re = cc[ac-1].Re + t2Re + t3Re
				ch[ah].Im = cc[ac-1].Im + t2Im + t3Im

				c2Re := cc[ac-1].Re + tr11*t2Re + tr12*t3Re
				c2Im := cc[ac-1].Im + tr11*t2Im + tr12*t3Im
				c3Re := cc[ac-1].Re + tr12*t2Re + tr11*t3Re
				c3Im := cc[ac-1].Im + tr12*t2Im + tr11*t3Im

				c4Re, c5Re := ComplexMult(ti12, ti11, t5Re, t4Re)
				c4Im, c5Im := ComplexMult(ti12, ti11, t5Im, t4Im)

				ch[ah+l1].Re = c2Re + c5Im
				ch[ah+l1].Im = c2Im - c5Re
				ch[ah+2*l1].Re = c3Re + c4Im
				ch[ah+2*l1].Im = c3Im - c4Re
				ch[ah+3*l1].Re = c3Re - c4Im
				ch[ah+3*l1].Im = c3Im + c4Re
				ch[ah+4*l1].Re = c2Re - c5Im
				ch[ah+4*l1].Im = c2Im + c5Re
			}
		}
	}
	// Note: ido > 1 case exists in FAAD2 but is marked as unreachable for AAC
}

// Forward performs the forward FFT (frequency analysis).
//
// Ported from: cfftf() in ~/dev/faad2/libfaad/cfft.c:896-899
func (cfft *CFFT) Forward(c []Complex) {
	cfft.cfftf1neg(c, -1)
}

// Backward performs the backward FFT (synthesis).
//
// Ported from: cfftb() in ~/dev/faad2/libfaad/cfft.c:901-904
func (cfft *CFFT) Backward(c []Complex) {
	cfft.cfftf1pos(c, +1)
}

// cfftf1pos is the main FFT computation for backward transform.
//
// Ported from: cfftf1pos() in ~/dev/faad2/libfaad/cfft.c:740-816
func (cfft *CFFT) cfftf1pos(c []Complex, isign int8) {
	n := cfft.N
	ch := cfft.Work
	ifac := cfft.IFac[:]
	wa := cfft.Tab

	nf := ifac[1]
	na := uint16(0)
	l1 := uint16(1)
	iw := uint16(0)

	for k1 := uint16(2); k1 <= nf+1; k1++ {
		ip := ifac[k1]
		l2 := ip * l1
		ido := n / l2

		switch ip {
		case 4:
			ix2 := iw + ido
			ix3 := ix2 + ido
			if na == 0 {
				passf4pos(ido, l1, c, ch, wa[iw:], wa[ix2:], wa[ix3:])
			} else {
				passf4pos(ido, l1, ch, c, wa[iw:], wa[ix2:], wa[ix3:])
			}
			na = 1 - na

		case 2:
			if na == 0 {
				passf2pos(ido, l1, c, ch, wa[iw:])
			} else {
				passf2pos(ido, l1, ch, c, wa[iw:])
			}
			na = 1 - na

		case 3:
			ix2 := iw + ido
			if na == 0 {
				passf3(ido, l1, c, ch, wa[iw:], wa[ix2:], isign)
			} else {
				passf3(ido, l1, ch, c, wa[iw:], wa[ix2:], isign)
			}
			na = 1 - na

		case 5:
			ix2 := iw + ido
			ix3 := ix2 + ido
			ix4 := ix3 + ido
			if na == 0 {
				passf5(ido, l1, c, ch, wa[iw:], wa[ix2:], wa[ix3:], wa[ix4:], isign)
			} else {
				passf5(ido, l1, ch, c, wa[iw:], wa[ix2:], wa[ix3:], wa[ix4:], isign)
			}
			na = 1 - na
		}

		l1 = l2
		iw += (ip - 1) * ido
	}

	if na == 0 {
		return
	}

	// Copy result back to c
	copy(c, ch[:n])
}

// cfftf1neg is the main FFT computation for forward transform.
//
// Ported from: cfftf1neg() in ~/dev/faad2/libfaad/cfft.c:818-894
func (cfft *CFFT) cfftf1neg(c []Complex, isign int8) {
	n := cfft.N
	ch := cfft.Work
	ifac := cfft.IFac[:]
	wa := cfft.Tab

	nf := ifac[1]
	na := uint16(0)
	l1 := uint16(1)
	iw := uint16(0)

	for k1 := uint16(2); k1 <= nf+1; k1++ {
		ip := ifac[k1]
		l2 := ip * l1
		ido := n / l2

		switch ip {
		case 4:
			ix2 := iw + ido
			ix3 := ix2 + ido
			if na == 0 {
				passf4neg(ido, l1, c, ch, wa[iw:], wa[ix2:], wa[ix3:])
			} else {
				passf4neg(ido, l1, ch, c, wa[iw:], wa[ix2:], wa[ix3:])
			}
			na = 1 - na

		case 2:
			if na == 0 {
				passf2neg(ido, l1, c, ch, wa[iw:])
			} else {
				passf2neg(ido, l1, ch, c, wa[iw:])
			}
			na = 1 - na

		case 3:
			ix2 := iw + ido
			if na == 0 {
				passf3(ido, l1, c, ch, wa[iw:], wa[ix2:], isign)
			} else {
				passf3(ido, l1, ch, c, wa[iw:], wa[ix2:], isign)
			}
			na = 1 - na

		case 5:
			ix2 := iw + ido
			ix3 := ix2 + ido
			ix4 := ix3 + ido
			if na == 0 {
				passf5(ido, l1, c, ch, wa[iw:], wa[ix2:], wa[ix3:], wa[ix4:], isign)
			} else {
				passf5(ido, l1, ch, c, wa[iw:], wa[ix2:], wa[ix3:], wa[ix4:], isign)
			}
			na = 1 - na
		}

		l1 = l2
		iw += (ip - 1) * ido
	}

	if na == 0 {
		return
	}

	// Copy result back to c
	copy(c, ch[:n])
}
