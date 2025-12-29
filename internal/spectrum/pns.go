// internal/spectrum/pns.go
package spectrum

import (
	"math"

	"github.com/llehouerou/go-aac/internal/syntax"
)

// NoiseOffset is the offset applied to PNS scale factors.
// Ported from: NOISE_OFFSET in ~/dev/faad2/libfaad/pns.h:40
const NoiseOffset = 90

// PNSState holds the random number generator state for PNS decoding.
// The state must be preserved across frames for proper decoder behavior.
//
// Ported from: __r1, __r2 in ~/dev/faad2/libfaad/structs.h:406-407
type PNSState struct {
	R1 uint32
	R2 uint32
}

// NewPNSState creates a new PNS state with default initial values.
func NewPNSState() *PNSState {
	// Initial values pre-computed as equivalent to (1, 1) after 1024 iterations.
	// Copied from: ~/dev/faad2/libfaad/decoder.c:152-153
	return &PNSState{
		R1: 0x2bb431ea,
		R2: 0x206155b7,
	}
}

// PNSDecodeConfig holds configuration for PNS decoding.
type PNSDecodeConfig struct {
	// ICSL is the left channel's individual channel stream
	ICSL *syntax.ICStream

	// ICSR is the right channel's individual channel stream (nil for mono)
	ICSR *syntax.ICStream

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16

	// ChannelPair is true if this is a CPE (channel pair element)
	ChannelPair bool

	// ObjectType is the AAC object type (for IMDCT scaling in fixed-point, unused in float)
	ObjectType uint8
}

// genRandVector generates a random noise vector with energy scaled by scale_factor.
// The formula is: spec[i] = random * scale, where scale = 2^(0.25 * scale_factor)
// and the random values are normalized to unit energy.
//
// Ported from: gen_rand_vector() in ~/dev/faad2/libfaad/pns.c:80-107 (floating-point path)
func genRandVector(spec []float64, scaleFactor int16, r1, r2 *uint32) {
	size := len(spec)
	if size == 0 {
		return
	}

	// Clamp scale factor to prevent overflow
	sf := scaleFactor
	if sf < -120 {
		sf = -120
	} else if sf > 120 {
		sf = 120
	}

	// Generate random values and accumulate energy
	energy := 0.0
	for i := 0; i < size; i++ {
		// Convert RNG output to signed float
		tmp := float64(int32(RNG(r1, r2)))
		spec[i] = tmp
		energy += tmp * tmp
	}

	// Normalize and scale
	if energy > 0 {
		// Normalize to unit energy
		scale := 1.0 / math.Sqrt(energy)
		// Apply scale factor: 2^(0.25 * sf)
		scale *= math.Pow(2.0, 0.25*float64(sf))

		for i := 0; i < size; i++ {
			spec[i] *= scale
		}
	}
}

// PNSDecode applies Perceptual Noise Substitution decoding.
// For bands coded with NOISE_HCB, generates pseudo-random noise
// scaled by the band's scale factor.
//
// For stereo (when specR != nil), handles noise correlation:
//   - If both channels have PNS on the same band AND ms_used is set,
//     the same noise is used for both channels (correlated).
//   - Otherwise, independent noise is generated for each channel.
//
// Ported from: pns_decode() in ~/dev/faad2/libfaad/pns.c:150-270
func PNSDecode(specL, specR []float64, state *PNSState, cfg *PNSDecodeConfig) {
	icsL := cfg.ICSL
	icsR := cfg.ICSR

	nshort := cfg.FrameLength / 8
	group := uint16(0)

	for g := uint8(0); g < icsL.NumWindowGroups; g++ {
		for b := uint8(0); b < icsL.WindowGroupLength[g]; b++ {
			base := group * nshort

			for sfb := uint8(0); sfb < icsL.MaxSFB; sfb++ {
				// RNG state for potential right channel correlation
				// Captured inside left channel block, per FAAD2 pns.c:209-210
				var r1Dep, r2Dep uint32

				// Process left channel PNS
				if IsNoiseICS(icsL, g, sfb) {
					// Clamp the final index (base + offset), not the offset alone
					// Per FAAD2 pns.c:206-207
					beginIdx := base + icsL.SWBOffset[sfb]
					endIdx := base + icsL.SWBOffset[sfb+1]
					if beginIdx > icsL.SWBOffsetMax {
						beginIdx = icsL.SWBOffsetMax
					}
					if endIdx > icsL.SWBOffsetMax {
						endIdx = icsL.SWBOffsetMax
					}

					if beginIdx < endIdx && int(endIdx) <= len(specL) {
						// Capture RNG state for potential right channel correlation
						// This must happen inside the left noise block, per FAAD2 pns.c:209-210
						r1Dep = state.R1
						r2Dep = state.R2
						genRandVector(specL[beginIdx:endIdx], icsL.ScaleFactors[g][sfb], &state.R1, &state.R2)
					}
				}

				// Process right channel PNS (if present)
				if icsR != nil && specR != nil && IsNoiseICS(icsR, g, sfb) {
					// Clamp the final index (base + offset), not the offset alone
					// Per FAAD2 pns.c:250-251, 258-259
					beginIdx := base + icsR.SWBOffset[sfb]
					endIdx := base + icsR.SWBOffset[sfb+1]
					if beginIdx > icsR.SWBOffsetMax {
						beginIdx = icsR.SWBOffsetMax
					}
					if endIdx > icsR.SWBOffsetMax {
						endIdx = icsR.SWBOffsetMax
					}

					// Determine if noise should be correlated
					// Correlated if: channel pair, both have PNS, and ms_used is set
					useCorrelated := cfg.ChannelPair &&
						IsNoiseICS(icsL, g, sfb) &&
						((icsL.MSMaskPresent == 1 && icsL.MSUsed[g][sfb] != 0) ||
							icsL.MSMaskPresent == 2)

					if beginIdx < endIdx && int(endIdx) <= len(specR) {
						if useCorrelated {
							// Use the same RNG state as left channel (dependent)
							genRandVector(specR[beginIdx:endIdx], icsR.ScaleFactors[g][sfb], &r1Dep, &r2Dep)
						} else {
							// Use independent RNG state
							genRandVector(specR[beginIdx:endIdx], icsR.ScaleFactors[g][sfb], &state.R1, &state.R2)
						}
					}
				}
			}
			group++
		}
	}
}
