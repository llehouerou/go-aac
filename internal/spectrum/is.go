package spectrum

import "github.com/llehouerou/go-aac/internal/syntax"

// ISDecodeConfig holds configuration for intensity stereo decoding.
type ISDecodeConfig struct {
	// ICSL is the left channel's individual channel stream
	ICSL *syntax.ICStream

	// ICSR is the right channel's individual channel stream (contains IS scale factors)
	ICSR *syntax.ICStream

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16
}

// ISDecode applies intensity stereo decoding to spectral coefficients.
// The right channel spectrum is reconstructed from the left channel
// for bands coded with intensity stereo (INTENSITY_HCB or INTENSITY_HCB2).
//
// The left channel is NOT modified; only the right channel is written.
//
// Ported from: is_decode() in ~/dev/faad2/libfaad/is.c:46-106
func ISDecode(lSpec, rSpec []float64, cfg *ISDecodeConfig) {
	// Stub - to be implemented
}
