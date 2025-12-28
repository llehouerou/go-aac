// Package syntax implements AAC bitstream syntax parsing.
// This file contains scale factor decoding.
package syntax

import (
	"github.com/llehouerou/go-aac/internal/bits"
	"github.com/llehouerou/go-aac/internal/huffman"
)

// DecodeScaleFactors decodes scale factors from the bitstream.
// Scale factors are differentially coded relative to the global gain.
//
// The algorithm maintains three separate running totals:
//   - scaleFactor: for spectral codebooks (1-11, 16-31)
//   - isPosition: for intensity stereo codebooks (14, 15)
//   - noiseEnergy: for noise (PNS) codebook (13)
//
// Zero codebook (0) results in scale factor 0.
//
// Ported from: decode_scale_factors() in ~/dev/faad2/libfaad/syntax.c:1894-1985
func DecodeScaleFactors(r *bits.Reader, ics *ICStream) error {
	// Initialize running totals
	scaleFactor := int16(ics.GlobalGain)
	isPosition := int16(0)
	noisePCMFlag := true
	noiseEnergy := int16(ics.GlobalGain) - 90

	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
			cb := ics.SFBCB[g][sfb]

			switch huffman.Codebook(cb) {
			case huffman.ZeroHCB:
				// Zero codebook: no spectral data, scale factor is 0
				ics.ScaleFactors[g][sfb] = 0

			case huffman.IntensityHCB, huffman.IntensityHCB2:
				// Intensity stereo: decode position delta
				// Note: ScaleFactor() returns delta directly (already - 60)
				delta := huffman.ScaleFactor(r)
				isPosition += int16(delta)
				ics.ScaleFactors[g][sfb] = isPosition

			case huffman.NoiseHCB:
				// PNS: decode noise energy
				// First noise value uses 9-bit PCM, rest use Huffman delta
				if noisePCMFlag {
					noisePCMFlag = false
					t := int16(r.GetBits(9)) - 256
					noiseEnergy += t
				} else {
					delta := huffman.ScaleFactor(r)
					noiseEnergy += int16(delta)
				}
				ics.ScaleFactors[g][sfb] = noiseEnergy

			default:
				// Spectral codebook: decode scale factor delta
				// Note: ScaleFactor() returns delta directly (already - 60)
				delta := huffman.ScaleFactor(r)
				scaleFactor += int16(delta)
				if scaleFactor < 0 || scaleFactor > 255 {
					return ErrScaleFactorRange
				}
				ics.ScaleFactors[g][sfb] = scaleFactor
			}
		}
	}

	return nil
}

// ParseScaleFactorData is the wrapper that matches FAAD2's scale_factor_data().
// It's a simple wrapper around DecodeScaleFactors for now; error-resilient
// mode (RVLC) is not supported.
//
// Ported from: scale_factor_data() in ~/dev/faad2/libfaad/syntax.c:1988-2016
func ParseScaleFactorData(r *bits.Reader, ics *ICStream) error {
	return DecodeScaleFactors(r, ics)
}
