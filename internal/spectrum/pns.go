// internal/spectrum/pns.go
package spectrum

import (
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
