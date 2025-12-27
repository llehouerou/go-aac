package tables

// SampleRates maps sample rate index to actual sample rate in Hz.
// Index 0-11 are valid; indices >= 12 are invalid.
//
// Source: ~/dev/faad2/libfaad/common.c:61-65
var SampleRates = [12]uint32{
	96000, 88200, 64000, 48000, 44100, 32000,
	24000, 22050, 16000, 12000, 11025, 8000,
}

// GetSampleRate returns the sample rate for a given index.
// Returns 0 for invalid indices (>= 12).
//
// Source: ~/dev/faad2/libfaad/common.c:59-71 (get_sample_rate function)
func GetSampleRate(srIndex uint8) uint32 {
	if srIndex >= 12 {
		return 0
	}
	return SampleRates[srIndex]
}

// GetSRIndex returns the sample rate index for a given sample rate.
// Uses threshold-based matching as defined in the MPEG-4 AAC standard.
// The thresholds are calculated as geometric means between adjacent rates.
//
// Source: ~/dev/faad2/libfaad/common.c:41-56 (get_sr_index function)
func GetSRIndex(sampleRate uint32) uint8 {
	if sampleRate >= 92017 {
		return 0
	}
	if sampleRate >= 75132 {
		return 1
	}
	if sampleRate >= 55426 {
		return 2
	}
	if sampleRate >= 46009 {
		return 3
	}
	if sampleRate >= 37566 {
		return 4
	}
	if sampleRate >= 27713 {
		return 5
	}
	if sampleRate >= 23004 {
		return 6
	}
	if sampleRate >= 18783 {
		return 7
	}
	if sampleRate >= 13856 {
		return 8
	}
	if sampleRate >= 11502 {
		return 9
	}
	if sampleRate >= 9391 {
		return 10
	}
	return 11
}
