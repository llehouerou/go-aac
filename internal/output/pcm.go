// Package output provides PCM output conversion.
// Ported from: ~/dev/faad2/libfaad/output.c
package output

import "math"

// PCM conversion constants.
// Ported from: ~/dev/faad2/libfaad/output.c:39-42

// FloatScale normalizes 16-bit range to [-1.0, 1.0].
// FLOAT_SCALE = 1.0 / (1 << 15)
const FloatScale = float32(1.0 / 32768.0)

// DMMul is the downmix multiplier: 1/(1+sqrt(2)+1/sqrt(2)).
// Used for 5.1 to stereo downmixing per ITU-R BS.775-1.
const DMMul = float32(0.3203772410170407)

// RSQRT2 is 1/sqrt(2), used for downmix calculations.
const RSQRT2 = float32(0.7071067811865475244)

// clip16 clips and rounds a float32 to int16 range.
// Matches FAAD2's CLIP macro + lrintf behavior.
//
// Ported from: ~/dev/faad2/libfaad/output.c:64-85
func clip16(sample float32) int16 {
	// Clipping
	if sample >= 32767.0 {
		return 32767
	}
	if sample <= -32768.0 {
		return -32768
	}
	// Round to nearest (lrintf behavior)
	return int16(math.RoundToEven(float64(sample)))
}

// clip24 clips and rounds a float32 to 24-bit signed integer range.
// Input should already be scaled by 256.
//
// Ported from: ~/dev/faad2/libfaad/output.c:154-172 (24-bit section)
func clip24(sample float32) int32 {
	// Clipping to 24-bit signed range
	if sample >= 8388607.0 {
		return 8388607
	}
	if sample <= -8388608.0 {
		return -8388608
	}
	return int32(math.RoundToEven(float64(sample)))
}

// clip32 clips and rounds a float32 to int32 range.
// Input should already be scaled by 65536.
//
// Ported from: ~/dev/faad2/libfaad/output.c:224-243 (32-bit section)
func clip32(sample float32) int32 {
	// Clipping to 32-bit signed range
	if sample >= 2147483647.0 {
		return 2147483647
	}
	if sample <= -2147483648.0 {
		return -2147483648
	}
	return int32(math.RoundToEven(float64(sample)))
}
