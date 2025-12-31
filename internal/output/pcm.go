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

// getSample retrieves a sample, optionally applying 5.1 to stereo downmix.
//
// When downMatrix is true, channels 0-4 are: C, L, R, Ls, Rs
// Output channel 0 = L + C*RSQRT2 + Ls*RSQRT2, scaled by DM_MUL
// Output channel 1 = R + C*RSQRT2 + Rs*RSQRT2, scaled by DM_MUL
//
// Ported from: get_sample in ~/dev/faad2/libfaad/output.c:45-61
func getSample(input [][]float32, channel uint8, sample uint16,
	downMatrix bool, channelMap []uint8) float32 {

	if !downMatrix {
		return input[channelMap[channel]][sample]
	}

	// 5.1 to stereo downmix
	// channelMap[0] = Center, [1] = Left, [2] = Right, [3] = Ls, [4] = Rs
	if channel == 0 {
		// Left output
		return DMMul * (input[channelMap[1]][sample] +
			input[channelMap[0]][sample]*RSQRT2 +
			input[channelMap[3]][sample]*RSQRT2)
	}
	// Right output
	return DMMul * (input[channelMap[2]][sample] +
		input[channelMap[0]][sample]*RSQRT2 +
		input[channelMap[4]][sample]*RSQRT2)
}

// ToPCM16Bit converts float32 samples to 16-bit PCM.
//
// Parameters:
//   - input: Per-channel float32 samples (input[channel][sample])
//   - channelMap: Maps output channels to input channels
//   - channels: Number of output channels
//   - frameLen: Number of samples per channel
//   - downMatrix: Enable 5.1 to stereo downmixing
//   - upMatrix: Enable mono to stereo upmixing
//   - output: Destination slice for interleaved int16 samples
//
// Ported from: to_PCM_16bit in ~/dev/faad2/libfaad/output.c:89-152
func ToPCM16Bit(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool, output []int16) {

	switch {
	case channels == 1 && !downMatrix:
		// Mono: direct copy with clipping
		ch := channelMap[0]
		for i := uint16(0); i < frameLen; i++ {
			output[i] = clip16(input[ch][i])
		}

	case channels == 2 && !downMatrix:
		if upMatrix {
			// Mono to stereo upmix: duplicate to both channels
			ch := channelMap[0]
			for i := uint16(0); i < frameLen; i++ {
				sample := clip16(input[ch][i])
				output[i*2+0] = sample
				output[i*2+1] = sample
			}
		} else {
			// True stereo
			chL := channelMap[0]
			chR := channelMap[1]
			for i := uint16(0); i < frameLen; i++ {
				output[i*2+0] = clip16(input[chL][i])
				output[i*2+1] = clip16(input[chR][i])
			}
		}

	default:
		// Generic multichannel with optional downmix
		for ch := uint8(0); ch < channels; ch++ {
			for i := uint16(0); i < frameLen; i++ {
				inp := getSample(input, ch, i, downMatrix, channelMap)
				output[int(i)*int(channels)+int(ch)] = clip16(inp)
			}
		}
	}
}
