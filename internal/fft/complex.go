package fft

// Complex represents a complex number with real and imaginary parts.
//
// Ported from: complex_t typedef in ~/dev/faad2/libfaad/common.h:390
// In FAAD2: typedef real_t complex_t[2]; with RE(A)=(A)[0], IM(A)=(A)[1]
type Complex struct {
	Re float32
	Im float32
}

// ComplexMult performs the complex multiplication used in FFT twiddle operations.
// It computes:
//
//	y1 = x1*c1 + x2*c2
//	y2 = x2*c1 - x1*c2
//
// This is NOT a standard complex multiplication but a specialized operation
// used in the FFT algorithm for combining twiddle factors.
//
// Ported from: ComplexMult() in ~/dev/faad2/libfaad/common.h:294-299
func ComplexMult(x1, x2, c1, c2 float32) (y1, y2 float32) {
	y1 = x1*c1 + x2*c2
	y2 = x2*c1 - x1*c2
	return
}
