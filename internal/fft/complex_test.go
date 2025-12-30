// internal/fft/complex_test.go
package fft

import (
	"math"
	"testing"
)

func TestComplex_Basic(t *testing.T) {
	c := Complex{Re: 3.0, Im: 4.0}
	if c.Re != 3.0 || c.Im != 4.0 {
		t.Errorf("Complex{3, 4} = %v, want {3, 4}", c)
	}
}

func TestComplexMult(t *testing.T) {
	// ComplexMult computes:
	// y1 = x1*c1 + x2*c2
	// y2 = x2*c1 - x1*c2
	//
	// This is used for twiddle factor multiplication in FFT.
	// Ported from: ComplexMult() in ~/dev/faad2/libfaad/common.h:294-299

	tests := []struct {
		name   string
		x1, x2 float32
		c1, c2 float32
		wantY1 float32
		wantY2 float32
	}{
		{
			name: "identity c1=1 c2=0",
			x1:   2.0, x2: 3.0,
			c1: 1.0, c2: 0.0,
			wantY1: 2.0, // 2*1 + 3*0
			wantY2: 3.0, // 3*1 - 2*0
		},
		{
			name: "swap c1=0 c2=1",
			x1:   2.0, x2: 3.0,
			c1: 0.0, c2: 1.0,
			wantY1: 3.0,  // 2*0 + 3*1
			wantY2: -2.0, // 3*0 - 2*1
		},
		{
			name: "general case",
			x1:   1.0, x2: 2.0,
			c1: 0.5, c2: 0.5,
			wantY1: 1.5, // 1*0.5 + 2*0.5
			wantY2: 0.5, // 2*0.5 - 1*0.5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			y1, y2 := ComplexMult(tt.x1, tt.x2, tt.c1, tt.c2)
			if math.Abs(float64(y1-tt.wantY1)) > 1e-6 {
				t.Errorf("y1 = %v, want %v", y1, tt.wantY1)
			}
			if math.Abs(float64(y2-tt.wantY2)) > 1e-6 {
				t.Errorf("y2 = %v, want %v", y2, tt.wantY2)
			}
		})
	}
}
