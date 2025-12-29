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
	icsR := cfg.ICSR

	// M/S not present
	if icsL.MSMaskPresent < 1 {
		return
	}

	nshort := cfg.FrameLength / 8
	group := uint16(0)

	for g := uint8(0); g < icsL.NumWindowGroups; g++ {
		for b := uint8(0); b < icsL.WindowGroupLength[g]; b++ {
			for sfb := uint8(0); sfb < icsL.MaxSFB; sfb++ {
				// Apply M/S if:
				// - ms_used[g][sfb] is set OR ms_mask_present == 2 (all bands)
				// - AND NOT intensity stereo in right channel
				// - AND NOT noise in left channel
				msEnabled := icsL.MSUsed[g][sfb] != 0 || icsL.MSMaskPresent == 2
				if msEnabled && IsIntensityICS(icsR, g, sfb) == 0 && !IsNoiseICS(icsL, g, sfb) {
					// Calculate SFB bounds, clamped to swb_offset_max
					start := icsL.SWBOffset[sfb]
					end := icsL.SWBOffset[sfb+1]
					if end > icsL.SWBOffsetMax {
						end = icsL.SWBOffsetMax
					}

					for i := start; i < end; i++ {
						k := group*nshort + i
						tmp := lSpec[k] - rSpec[k]
						lSpec[k] = lSpec[k] + rSpec[k]
						rSpec[k] = tmp
					}
				}
			}
			group++
		}
	}
}
