package spectrum

import (
	"github.com/llehouerou/go-aac/internal/huffman"
	"github.com/llehouerou/go-aac/internal/syntax"
	"github.com/llehouerou/go-aac/internal/tables"
)

// ApplyScaleFactorsConfig holds configuration for scale factor application.
type ApplyScaleFactorsConfig struct {
	// ICS contains window and scale factor information
	ICS *syntax.ICStream

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16
}

// ApplyScaleFactors applies scale factors to spectral coefficients in-place.
// For each scalefactor band: spec[i] *= 2^((sf - 100) / 4)
//
// Intensity stereo and noise (PNS) bands are zeroed, as they are filled
// by dedicated tools (is_decode, pns_decode) later in the pipeline.
//
// Ported from: quant_to_spec() scale factor part in ~/dev/faad2/libfaad/specrec.c:549-693
func ApplyScaleFactors(specData []float64, cfg *ApplyScaleFactorsConfig) {
	ics := cfg.ICS

	// Process each window group
	gindex := uint16(0)
	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		// win_inc is the offset between windows within a group
		winInc := ics.SWBOffset[ics.NumSWB]

		// Process each scalefactor band
		j := uint16(0)
		for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
			cb := huffman.Codebook(ics.SFBCB[g][sfb])
			sf := ics.ScaleFactors[g][sfb]

			width := ics.SWBOffset[sfb+1] - ics.SWBOffset[sfb]

			// Intensity stereo and noise bands: zero the coefficients
			// They will be filled later by dedicated tools
			if IsIntensity(cb) != 0 || IsNoise(cb) {
				for win := uint8(0); win < ics.WindowGroupLength[g]; win++ {
					wa := gindex + uint16(win)*winInc + j
					for bin := uint16(0); bin < width; bin++ {
						specData[wa+bin] = 0.0
					}
				}
			} else {
				// Normal spectral band: apply scale factor
				// Formula: spec[i] *= 2^((sf - 100) / 4)
				sfAdjusted := int(sf) - tables.ScaleFactorOffset
				exp := sfAdjusted >> 2 // Integer part of exponent
				frac := sfAdjusted & 3 // Fractional part (0-3)

				// Compute scale: pow2sf_tab[exp+25] * pow2_table[frac]
				// Pow2SFTable index 25 = 1.0 (2^0)
				expIdx := exp + 25
				if expIdx < 0 {
					expIdx = 0
				} else if expIdx >= len(tables.Pow2SFTable) {
					expIdx = len(tables.Pow2SFTable) - 1
				}

				scf := tables.Pow2SFTable[expIdx] * tables.Pow2FracTable[frac]

				// Apply to all windows in this group
				for win := uint8(0); win < ics.WindowGroupLength[g]; win++ {
					wa := gindex + uint16(win)*winInc + j
					for bin := uint16(0); bin < width; bin++ {
						specData[wa+bin] *= scf
					}
				}
			}

			j += width
		}

		// Advance gindex by the total span of this group
		gindex += uint16(ics.WindowGroupLength[g]) * winInc
	}
}
