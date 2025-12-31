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

// Downmixer handles multichannel to stereo downmixing.
//
// By default, it performs 5.1 to stereo downmix using ITU-R BS.775-1 coefficients.
// The LFE channel is excluded by default (as in FAAD2).
//
// Ported from: get_sample in ~/dev/faad2/libfaad/output.c:45-61
type Downmixer struct {
	// Enabled controls whether downmixing is active.
	// When false, channels are passed through unchanged.
	Enabled bool

	// IncludeLFE controls whether the LFE channel contributes to the mix.
	// When true, LFE is added to both L and R at LFEGain level.
	// Default is false (matching FAAD2 behavior).
	IncludeLFE bool

	// LFEGain is the mixing level for LFE when IncludeLFE is true.
	// Typical values: 0.5 to 0.7 (LFE is usually attenuated in downmix).
	// Only used when IncludeLFE is true.
	LFEGain float32
}

// NewDownmixer creates a new Downmixer with default settings.
// Downmixing is enabled, LFE is excluded (matching FAAD2 defaults).
func NewDownmixer() *Downmixer {
	return &Downmixer{
		Enabled:    true,
		IncludeLFE: false,
		LFEGain:    0.0,
	}
}

// Downmix5_1ToStereo converts a 5.1 channel sample to stereo.
//
// The channel map specifies which input channels correspond to which positions:
// channelMap[0]=Center, [1]=FrontLeft, [2]=FrontRight, [3]=RearLeft, [4]=RearRight, [5]=LFE
//
// Formula (ITU-R BS.775-1):
//
//	L = DM_MUL * (L + C*InvSqrt2 + Ls*InvSqrt2)
//	R = DM_MUL * (R + C*InvSqrt2 + Rs*InvSqrt2)
//
// Ported from: get_sample in ~/dev/faad2/libfaad/output.c:45-61
func (d *Downmixer) Downmix5_1ToStereo(input [][]float32, channelMap []uint8, sampleIdx uint16) (left, right float32) {
	if !d.Enabled {
		// Pass through front L/R when disabled
		return input[channelMap[ChannelFrontLeft]][sampleIdx],
			input[channelMap[ChannelFrontRight]][sampleIdx]
	}

	// Get channel samples using the channel map
	center := input[channelMap[ChannelCenter]][sampleIdx]
	frontL := input[channelMap[ChannelFrontLeft]][sampleIdx]
	frontR := input[channelMap[ChannelFrontRight]][sampleIdx]
	rearL := input[channelMap[ChannelRearLeft]][sampleIdx]
	rearR := input[channelMap[ChannelRearRight]][sampleIdx]

	// Apply ITU-R BS.775-1 downmix matrix
	left = DownmixMul * (frontL + center*InvSqrt2 + rearL*InvSqrt2)
	right = DownmixMul * (frontR + center*InvSqrt2 + rearR*InvSqrt2)

	// Optionally mix in LFE
	if d.IncludeLFE && len(channelMap) > int(ChannelLFE) {
		lfe := input[channelMap[ChannelLFE]][sampleIdx]
		lfeContrib := lfe * d.LFEGain * DownmixMul
		left += lfeContrib
		right += lfeContrib
	}

	return left, right
}

// DownmixFrame converts a full frame of 5.1 audio to stereo.
//
// Returns two slices: left and right channel output samples.
// The output length matches frameLen.
//
// This is more efficient than calling Downmix5_1ToStereo for each sample
// when processing complete frames.
func (d *Downmixer) DownmixFrame(input [][]float32, channelMap []uint8, frameLen uint16) (left, right []float32) {
	left = make([]float32, frameLen)
	right = make([]float32, frameLen)

	for i := uint16(0); i < frameLen; i++ {
		left[i], right[i] = d.Downmix5_1ToStereo(input, channelMap, i)
	}

	return left, right
}

// GetDownmixedSample returns a sample for the specified output channel,
// applying 5.1 to stereo downmix if enabled.
//
// This provides compatibility with the get_sample pattern used in FAAD2.
// For stereo output from 5.1 input:
// - channel 0 returns the downmixed left sample
// - channel 1 returns the downmixed right sample
//
// When downmix is disabled, returns the raw sample from channelMap[channel].
//
// Ported from: get_sample in ~/dev/faad2/libfaad/output.c:45-61
func (d *Downmixer) GetDownmixedSample(input [][]float32, channel uint8, sampleIdx uint16, channelMap []uint8) float32 {
	if !d.Enabled {
		return input[channelMap[channel]][sampleIdx]
	}

	// For 5.1 to stereo, we only output channels 0 and 1
	left, right := d.Downmix5_1ToStereo(input, channelMap, sampleIdx)
	if channel == 0 {
		return left
	}
	return right
}
