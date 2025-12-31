// Package output provides PCM output conversion.
// This file contains downmix functionality for multichannel to stereo conversion.
//
// Ported from: ~/dev/faad2/libfaad/output.c (get_sample function)
package output

// Channel position constants for AAC 5.1 layout.
// These match FAAD2's internal_channel ordering for downmix.
//
// Ported from: implicit ordering in ~/dev/faad2/libfaad/output.c:45-61
const (
	ChannelCenter     uint8 = 0 // Center channel (C)
	ChannelFrontLeft  uint8 = 1 // Front left channel (L)
	ChannelFrontRight uint8 = 2 // Front right channel (R)
	ChannelRearLeft   uint8 = 3 // Rear/surround left (Ls)
	ChannelRearRight  uint8 = 4 // Rear/surround right (Rs)
	ChannelLFE        uint8 = 5 // Low Frequency Effects (subwoofer)
)

// Downmix matrix coefficients for 5.1 to stereo conversion.
// Based on ITU-R BS.775-1 recommendation.
//
// Ported from: ~/dev/faad2/libfaad/output.c:41-42
const (
	// DownmixMul is the overall normalization factor.
	// DownmixMul = 1/(1 + sqrt(2) + 1/sqrt(2)) ≈ 0.3204
	// This prevents clipping when all channels are at full scale.
	DownmixMul = float32(0.3203772410170407)

	// InvSqrt2 is 1/sqrt(2) ≈ 0.7071, used for center and surround mixing.
	InvSqrt2 = float32(0.7071067811865475244)
)
