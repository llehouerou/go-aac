package spectrum

import "github.com/llehouerou/go-aac/internal/huffman"

// IsIntensity returns the intensity stereo direction for a codebook.
// Returns 1 for in-phase (INTENSITY_HCB), -1 for out-of-phase (INTENSITY_HCB2), 0 otherwise.
//
// Ported from: is_intensity() in ~/dev/faad2/libfaad/is.h:43-54
func IsIntensity(cb huffman.Codebook) int8 {
	switch cb {
	case huffman.IntensityHCB:
		return 1
	case huffman.IntensityHCB2:
		return -1
	default:
		return 0
	}
}

// IsNoise returns true if the codebook indicates a PNS (noise) band.
//
// Ported from: is_noise() in ~/dev/faad2/libfaad/pns.h:47-52
func IsNoise(cb huffman.Codebook) bool {
	return cb == huffman.NoiseHCB
}
