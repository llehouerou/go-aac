// internal/syntax/section.go
package syntax

import (
	"github.com/llehouerou/go-aac/internal/bits"
	"github.com/llehouerou/go-aac/internal/huffman"
)

// ParseSectionData parses section data (Table 4.4.25).
// Section data assigns Huffman codebooks to ranges of scale factor bands.
//
// Ported from: section_data() in ~/dev/faad2/libfaad/syntax.c:1731-1881
func ParseSectionData(r *bits.Reader, ics *ICStream) error {
	var sectBits, sectLim uint8

	if ics.WindowSequence == EightShortSequence {
		sectBits = 3
		sectLim = 8 * 15 // 120
	} else {
		sectBits = 5
		sectLim = MaxSFB // 51
	}
	sectEscVal := uint8((1 << sectBits) - 1)

	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		k := uint8(0)
		i := uint8(0)

		for k < ics.MaxSFB {
			if r.Error() {
				return ErrBitstreamRead
			}
			if i >= sectLim {
				return ErrSectionLimit
			}

			// Read codebook (4 bits)
			sectCB := uint8(r.GetBits(4))
			ics.SectCB[g][i] = sectCB

			// Codebook 12 is reserved
			if sectCB == 12 {
				return ErrReservedCodebook
			}

			// Track special codebook usage
			if sectCB == uint8(huffman.NoiseHCB) {
				ics.NoiseUsed = true
			}
			if sectCB == uint8(huffman.IntensityHCB) || sectCB == uint8(huffman.IntensityHCB2) {
				ics.IsUsed = true
			}

			// Read section length
			sectLen := uint8(0)
			for {
				sectLenIncr := uint8(r.GetBits(uint(sectBits)))
				if sectLen > sectLim {
					return ErrSectionLength
				}
				sectLen += sectLenIncr
				if sectLenIncr != sectEscVal {
					break
				}
			}

			ics.SectStart[g][i] = uint16(k)
			ics.SectEnd[g][i] = uint16(k + sectLen)

			if sectLen > sectLim || k+sectLen > sectLim {
				return ErrSectionLength
			}

			// Assign codebook to each SFB in this section
			for sfb := k; sfb < k+sectLen; sfb++ {
				ics.SFBCB[g][sfb] = sectCB
			}

			k += sectLen
			i++
		}

		ics.NumSec[g] = i

		// Verify all SFBs covered
		if k != ics.MaxSFB {
			return ErrSectionCoverage
		}
	}

	return nil
}
