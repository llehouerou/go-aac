package mdct

import "testing"

func TestNewMDCT_CreatesValidInstance(t *testing.T) {
	tests := []struct {
		n       uint16
		fftSize uint16
	}{
		{256, 64},   // short blocks
		{2048, 512}, // long blocks
	}

	for _, tt := range tests {
		m := NewMDCT(tt.n)
		if m == nil {
			t.Fatalf("NewMDCT(%d) returned nil", tt.n)
		}
		if m.N != tt.n {
			t.Errorf("N = %d, want %d", m.N, tt.n)
		}
		if m.N4 != tt.n/4 {
			t.Errorf("N4 = %d, want %d", m.N4, tt.n/4)
		}
		if m.cfft == nil {
			t.Error("cfft is nil")
		}
		if len(m.sincos) != int(tt.fftSize) {
			t.Errorf("sincos length = %d, want %d", len(m.sincos), tt.fftSize)
		}
	}
}
