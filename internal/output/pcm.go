// Package output provides PCM output conversion.
// Ported from: ~/dev/faad2/libfaad/output.c
package output

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
