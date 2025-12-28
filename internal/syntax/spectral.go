// internal/syntax/spectral.go
package syntax

import (
	"github.com/llehouerou/go-aac/internal/bits"
	"github.com/llehouerou/go-aac/internal/huffman"
)

// ParseSpectralData decodes spectral coefficients from the bitstream.
// The spectral data is Huffman-decoded using the codebook assigned by section data.
//
// Codebooks 1-4 decode 4 values at a time (quad), codebooks 5-11 decode 2 values (pair).
// Zero codebook (0) means silence - no spectral data in bitstream.
// Noise codebook (13) - spectral data is synthesized later (PNS).
// Intensity codebooks (14, 15) - stereo parameters, not spectral data.
//
// Ported from: spectral_data() in ~/dev/faad2/libfaad/syntax.c:2156-2236
func ParseSpectralData(r *bits.Reader, ics *ICStream, specData []int16, frameLength uint16) error {
	nshort := frameLength / 8
	groups := uint8(0)

	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		p := uint16(groups) * nshort

		for i := uint8(0); i < ics.NumSec[g]; i++ {
			sectCB := ics.SectCB[g][i]

			// Determine increment (quad vs pair)
			var inc uint16
			if sectCB >= uint8(huffman.FirstPairHCB) {
				inc = 2
			} else {
				inc = 4
			}

			switch huffman.Codebook(sectCB) {
			case huffman.ZeroHCB, huffman.NoiseHCB, huffman.IntensityHCB, huffman.IntensityHCB2:
				// No spectral data - just advance position
				p += ics.SectSFBOffset[g][ics.SectEnd[g][i]] - ics.SectSFBOffset[g][ics.SectStart[g][i]]

			default:
				// Decode spectral data using Huffman
				start := ics.SectSFBOffset[g][ics.SectStart[g][i]]
				end := ics.SectSFBOffset[g][ics.SectEnd[g][i]]

				for k := start; k < end; k += inc {
					if err := huffman.SpectralData(sectCB, r, specData[p:]); err != nil {
						return err
					}
					p += inc
				}
			}
		}

		groups += ics.WindowGroupLength[g]
	}

	return nil
}
