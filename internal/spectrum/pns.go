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
