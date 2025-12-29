package spectrum

import (
	"math"

	"github.com/llehouerou/go-aac/internal/syntax"
)

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
	icsL := cfg.ICSL
	icsR := cfg.ICSR

	nshort := cfg.FrameLength / 8
	group := uint16(0)

	for g := uint8(0); g < icsR.NumWindowGroups; g++ {
		for b := uint8(0); b < icsR.WindowGroupLength[g]; b++ {
			for sfb := uint8(0); sfb < icsR.MaxSFB; sfb++ {
				isDir := IsIntensityICS(icsR, g, sfb)
				if isDir != 0 {
					// Get scale factor and clamp to valid range
					scaleFactor := icsR.ScaleFactors[g][sfb]
					if scaleFactor < -120 {
						scaleFactor = -120
					} else if scaleFactor > 120 {
						scaleFactor = 120
					}

					// Calculate scale: 0.5^(scaleFactor/4)
					scale := math.Pow(0.5, 0.25*float64(scaleFactor))

					// Determine sign inversion
					invertSign := isDir != InvertIntensity(icsL, g, sfb)

					// Calculate SFB bounds, clamped to swb_offset_max
					start := icsR.SWBOffset[sfb]
					end := icsR.SWBOffset[sfb+1]
					if end > icsL.SWBOffsetMax {
						end = icsL.SWBOffsetMax
					}

					// Copy scaled left to right
					for i := start; i < end; i++ {
						k := group*nshort + i
						rSpec[k] = lSpec[k] * scale
						if invertSign {
							rSpec[k] = -rSpec[k]
						}
					}
				}
			}
			group++
		}
	}
}
