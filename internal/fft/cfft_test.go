package fft

import (
	"fmt"
	"testing"
)

func TestFactorize(t *testing.T) {
	// FAAD2 factorizes using factors 3, 4, 2, 5 (in that order of preference)
	// The resulting ifac array contains:
	// ifac[0] = n
	// ifac[1] = number of factors
	// ifac[2..] = the factors
	//
	// Ported from: cffti1() factorization in ~/dev/faad2/libfaad/cfft.c:906-952

	tests := []struct {
		n           uint16
		wantNF      uint16
		wantFactors []uint16
	}{
		{
			n:           64,
			wantNF:      3,
			wantFactors: []uint16{4, 4, 4}, // 64 = 4*4*4
		},
		{
			n:           512,
			wantNF:      5,
			wantFactors: []uint16{2, 4, 4, 4, 4}, // 512 = 4*4*4*4*2 with 2 moved to front
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("n=%d", tt.n), func(t *testing.T) {
			var ifac [15]uint16
			factorize(tt.n, ifac[:])

			if ifac[0] != tt.n {
				t.Errorf("ifac[0] = %d, want %d", ifac[0], tt.n)
			}
			if ifac[1] != tt.wantNF {
				t.Errorf("ifac[1] (nf) = %d, want %d", ifac[1], tt.wantNF)
			}
			for i, wantF := range tt.wantFactors {
				if ifac[i+2] != wantF {
					t.Errorf("ifac[%d] = %d, want %d", i+2, ifac[i+2], wantF)
				}
			}
		})
	}
}

func TestNewCFFT(t *testing.T) {
	tests := []uint16{64, 512}

	for _, n := range tests {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			cfft := NewCFFT(n)
			if cfft == nil {
				t.Fatal("NewCFFT returned nil")
			}
			if cfft.N != n {
				t.Errorf("cfft.N = %d, want %d", cfft.N, n)
			}
			if len(cfft.Work) != int(n) {
				t.Errorf("len(Work) = %d, want %d", len(cfft.Work), n)
			}
			if len(cfft.Tab) != int(n) {
				t.Errorf("len(Tab) = %d, want %d", len(cfft.Tab), n)
			}
		})
	}
}

func TestPassf4pos_Simple(t *testing.T) {
	// Test radix-4 butterfly with a simple 4-point case
	// This validates the core butterfly computation.
	//
	// For ido=1, l1=1, the radix-4 butterfly computes a 4-point DFT.

	// Input: 4 complex numbers
	cc := []Complex{
		{Re: 1, Im: 0},
		{Re: 1, Im: 0},
		{Re: 1, Im: 0},
		{Re: 1, Im: 0},
	}

	ch := make([]Complex, 4)

	// For a 4-point FFT of all 1s:
	// Forward: [4, 0, 0, 0]
	// Backward: [4, 0, 0, 0]

	passf4pos(1, 1, cc, ch, nil, nil, nil)

	// DC component should be sum of all inputs = 4
	if ch[0].Re != 4.0 {
		t.Errorf("ch[0].Re = %v, want 4.0", ch[0].Re)
	}
	if ch[0].Im != 0.0 {
		t.Errorf("ch[0].Im = %v, want 0.0", ch[0].Im)
	}
}

func TestPassf4neg_Simple(t *testing.T) {
	// Test radix-4 butterfly for forward FFT (isign=-1)
	// For ido=1, l1=1, this computes a 4-point forward DFT.

	// Input: 4 complex numbers
	cc := []Complex{
		{Re: 1, Im: 0},
		{Re: 1, Im: 0},
		{Re: 1, Im: 0},
		{Re: 1, Im: 0},
	}

	ch := make([]Complex, 4)

	passf4neg(1, 1, cc, ch, nil, nil, nil)

	// DC component should be sum of all inputs = 4
	if ch[0].Re != 4.0 {
		t.Errorf("ch[0].Re = %v, want 4.0", ch[0].Re)
	}
	if ch[0].Im != 0.0 {
		t.Errorf("ch[0].Im = %v, want 0.0", ch[0].Im)
	}
}
