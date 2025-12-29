package spectrum

import "github.com/llehouerou/go-aac/internal/syntax"

// MSDecodeConfig holds configuration for M/S stereo decoding.
type MSDecodeConfig struct {
	// ICSL is the left channel's individual channel stream (contains ms_mask_present, ms_used)
	ICSL *syntax.ICStream

	// ICSR is the right channel's individual channel stream
	ICSR *syntax.ICStream

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16
}

// MSDecode applies Mid/Side stereo decoding to spectral coefficients in-place.
// Converts M/S encoded bands back to L/R: L = M + S, R = M - S
//
// M/S decoding is skipped for:
// - Bands where ms_mask_present = 0
// - Intensity stereo bands (handled by is_decode)
// - Noise bands (handled by pns_decode)
//
// Ported from: ms_decode() in ~/dev/faad2/libfaad/ms.c:39-77
func MSDecode(lSpec, rSpec []float64, cfg *MSDecodeConfig) {
	icsL := cfg.ICSL

	// M/S not present
	if icsL.MSMaskPresent < 1 {
		return
	}
}
