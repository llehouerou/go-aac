package spectrum

import (
	"github.com/llehouerou/go-aac/internal/huffman"
	"github.com/llehouerou/go-aac/internal/syntax"
)

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

// IsIntensityICS returns the intensity stereo direction for a scalefactor band.
// Returns 1 for in-phase (INTENSITY_HCB), -1 for out-of-phase (INTENSITY_HCB2), 0 otherwise.
//
// Ported from: is_intensity() in ~/dev/faad2/libfaad/is.h:43-54
func IsIntensityICS(ics *syntax.ICStream, group, sfb uint8) int8 {
	return IsIntensity(huffman.Codebook(ics.SFBCB[group][sfb]))
}

// IsNoiseICS returns true if the scalefactor band uses noise (PNS) coding.
//
// Ported from: is_noise() in ~/dev/faad2/libfaad/pns.h:47-52
func IsNoiseICS(ics *syntax.ICStream, group, sfb uint8) bool {
	return IsNoise(huffman.Codebook(ics.SFBCB[group][sfb]))
}

// InvertIntensity returns the intensity stereo sign inversion factor.
// Returns -1 if the M/S mask indicates inversion, 1 otherwise.
//
// Ported from: invert_intensity() in ~/dev/faad2/libfaad/is.h:56-61
func InvertIntensity(ics *syntax.ICStream, group, sfb uint8) int8 {
	if ics.MSMaskPresent == 1 {
		return 1 - 2*int8(ics.MSUsed[group][sfb])
	}
	return 1
}
