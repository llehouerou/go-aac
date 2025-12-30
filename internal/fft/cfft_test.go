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
