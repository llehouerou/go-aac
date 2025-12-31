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

func TestPassf2pos_Simple(t *testing.T) {
	// Test radix-2 butterfly with ido > 1 case
	// (ido=1 case is never used according to FAAD2 comments)

	cc := []Complex{
		{Re: 1, Im: 0},
		{Re: 2, Im: 0},
		{Re: 3, Im: 0},
		{Re: 4, Im: 0},
	}

	ch := make([]Complex, 4)
	wa := []Complex{{Re: 1, Im: 0}, {Re: 1, Im: 0}}

	passf2pos(2, 1, cc, ch, wa)

	// After radix-2: ch[0,1] = sum, ch[2,3] = diff (with twiddle)
	// Sum: cc[0]+cc[2]=1+3=4, cc[1]+cc[3]=2+4=6
	if ch[0].Re != 4.0 {
		t.Errorf("ch[0].Re = %v, want 4.0", ch[0].Re)
	}
	if ch[1].Re != 6.0 {
		t.Errorf("ch[1].Re = %v, want 6.0", ch[1].Re)
	}
}

func TestPassf2neg_Simple(t *testing.T) {
	// Test radix-2 butterfly for forward FFT (isign=-1)

	cc := []Complex{
		{Re: 1, Im: 0},
		{Re: 2, Im: 0},
		{Re: 3, Im: 0},
		{Re: 4, Im: 0},
	}

	ch := make([]Complex, 4)
	wa := []Complex{{Re: 1, Im: 0}, {Re: 1, Im: 0}}

	passf2neg(2, 1, cc, ch, wa)

	// After radix-2: ch[0,1] = sum, ch[2,3] = diff (with twiddle)
	// Sum: cc[0]+cc[2]=1+3=4, cc[1]+cc[3]=2+4=6
	if ch[0].Re != 4.0 {
		t.Errorf("ch[0].Re = %v, want 4.0", ch[0].Re)
	}
	if ch[1].Re != 6.0 {
		t.Errorf("ch[1].Re = %v, want 6.0", ch[1].Re)
	}
}

func TestPassf3_Constants(t *testing.T) {
	// Verify the trigonometric constants used in radix-3
	// From FAAD2: ~/dev/faad2/libfaad/cfft.c:185-186
	taur := float32(-0.5)
	taui := float32(0.866025403784439) // sqrt(3)/2

	if taur != -0.5 {
		t.Errorf("taur = %v, want -0.5", taur)
	}
	if taui != 0.866025403784439 {
		t.Errorf("taui = %v, want 0.866025403784439", taui)
	}

	// Verify the constants are defined correctly in the implementation
	if taur3 != -0.5 {
		t.Errorf("taur3 = %v, want -0.5", taur3)
	}
	if taui3 != 0.866025403784439 {
		t.Errorf("taui3 = %v, want 0.866025403784439", taui3)
	}
}

func TestPassf5_Constants(t *testing.T) {
	// Verify the trigonometric constants used in radix-5
	// From FAAD2: ~/dev/faad2/libfaad/cfft.c:507-510
	expectedTr11 := float32(0.309016994374947)  // cos(2*pi/5)
	expectedTi11 := float32(0.951056516295154)  // sin(2*pi/5)
	expectedTr12 := float32(-0.809016994374947) // cos(4*pi/5)
	expectedTi12 := float32(0.587785252292473)  // sin(4*pi/5)

	if tr11 != expectedTr11 {
		t.Errorf("tr11 = %v, want %v", tr11, expectedTr11)
	}
	if ti11 != expectedTi11 {
		t.Errorf("ti11 = %v, want %v", ti11, expectedTi11)
	}
	if tr12 != expectedTr12 {
		t.Errorf("tr12 = %v, want %v", tr12, expectedTr12)
	}
	if ti12 != expectedTi12 {
		t.Errorf("ti12 = %v, want %v", ti12, expectedTi12)
	}
}

func TestPassf3_BackwardFFT(t *testing.T) {
	// Test radix-3 backward FFT (isign=1)
	// With ido=2, l1=1, we process a 3-point transform

	// Input: 6 complex numbers (3 * ido=2)
	cc := []Complex{
		{Re: 1, Im: 0}, {Re: 2, Im: 0}, // first group
		{Re: 3, Im: 0}, {Re: 4, Im: 0}, // second group
		{Re: 5, Im: 0}, {Re: 6, Im: 0}, // third group
	}

	ch := make([]Complex, 6)
	wa1 := []Complex{{Re: 1, Im: 0}, {Re: 1, Im: 0}}
	wa2 := []Complex{{Re: 1, Im: 0}, {Re: 1, Im: 0}}

	passf3(2, 1, cc, ch, wa1, wa2, 1)

	// For backward FFT with identity twiddles, the DC component should be sum
	// ch[0] = cc[0] + cc[2] + cc[4] = 1 + 3 + 5 = 9
	if ch[0].Re != 9.0 {
		t.Errorf("ch[0].Re = %v, want 9.0", ch[0].Re)
	}
}

func TestPassf3_ForwardFFT(t *testing.T) {
	// Test radix-3 forward FFT (isign=-1)
	cc := []Complex{
		{Re: 1, Im: 0}, {Re: 2, Im: 0},
		{Re: 3, Im: 0}, {Re: 4, Im: 0},
		{Re: 5, Im: 0}, {Re: 6, Im: 0},
	}

	ch := make([]Complex, 6)
	wa1 := []Complex{{Re: 1, Im: 0}, {Re: 1, Im: 0}}
	wa2 := []Complex{{Re: 1, Im: 0}, {Re: 1, Im: 0}}

	passf3(2, 1, cc, ch, wa1, wa2, -1)

	// DC component should be the same: sum of all inputs
	if ch[0].Re != 9.0 {
		t.Errorf("ch[0].Re = %v, want 9.0", ch[0].Re)
	}
}

func TestPassf5_BackwardFFT_ido1(t *testing.T) {
	// Test radix-5 backward FFT (isign=1) with ido=1
	// This is the common case for AAC as noted in FAAD2

	// Input: 5 complex numbers
	cc := []Complex{
		{Re: 1, Im: 0},
		{Re: 2, Im: 0},
		{Re: 3, Im: 0},
		{Re: 4, Im: 0},
		{Re: 5, Im: 0},
	}

	ch := make([]Complex, 5)

	passf5(1, 1, cc, ch, nil, nil, nil, nil, 1)

	// DC component should be sum of all inputs: 1+2+3+4+5 = 15
	if ch[0].Re != 15.0 {
		t.Errorf("ch[0].Re = %v, want 15.0", ch[0].Re)
	}
	if ch[0].Im != 0.0 {
		t.Errorf("ch[0].Im = %v, want 0.0", ch[0].Im)
	}
}

func TestPassf5_ForwardFFT_ido1(t *testing.T) {
	// Test radix-5 forward FFT (isign=-1) with ido=1

	cc := []Complex{
		{Re: 1, Im: 0},
		{Re: 2, Im: 0},
		{Re: 3, Im: 0},
		{Re: 4, Im: 0},
		{Re: 5, Im: 0},
	}

	ch := make([]Complex, 5)

	passf5(1, 1, cc, ch, nil, nil, nil, nil, -1)

	// DC component should be sum of all inputs: 1+2+3+4+5 = 15
	if ch[0].Re != 15.0 {
		t.Errorf("ch[0].Re = %v, want 15.0", ch[0].Re)
	}
	if ch[0].Im != 0.0 {
		t.Errorf("ch[0].Im = %v, want 0.0", ch[0].Im)
	}
}
