package tables

import "github.com/llehouerou/go-aac"

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

// predSFBMax contains max prediction SFB per sample rate index.
// Source: ~/dev/faad2/libfaad/common.c:75-78
var predSFBMax = [12]uint8{
	33, 33, 38, 40, 40, 40, 41, 41, 37, 37, 37, 34,
}

// MaxPredSFB returns the maximum prediction scalefactor band for a sample rate index.
// Returns 0 for invalid indices.
// Source: ~/dev/faad2/libfaad/common.c:73-85
func MaxPredSFB(srIndex uint8) uint8 {
	if srIndex >= 12 {
		return 0
	}
	return predSFBMax[srIndex]
}

// tnsSFBMax contains max TNS SFB values.
// Columns: [Main/LC long, Main/LC short, SSR long, SSR short]
// Source: ~/dev/faad2/libfaad/common.c:96-114
var tnsSFBMax = [16][4]uint8{
	{31, 9, 28, 7},  // 96000
	{31, 9, 28, 7},  // 88200
	{34, 10, 27, 7}, // 64000
	{40, 14, 26, 6}, // 48000
	{42, 14, 26, 6}, // 44100
	{51, 14, 26, 6}, // 32000
	{46, 14, 29, 7}, // 24000
	{46, 14, 29, 7}, // 22050
	{42, 14, 23, 8}, // 16000
	{42, 14, 23, 8}, // 12000
	{42, 14, 23, 8}, // 11025
	{39, 14, 19, 7}, // 8000
	{39, 14, 19, 7}, // 7350
	{0, 0, 0, 0},
	{0, 0, 0, 0},
	{0, 0, 0, 0},
}

// MaxTNSSFB returns the maximum TNS scalefactor band.
// Source: ~/dev/faad2/libfaad/common.c:87-121
func MaxTNSSFB(srIndex uint8, objectType aac.ObjectType, isShort bool) uint8 {
	if srIndex >= 16 {
		return 0
	}

	i := 0
	if isShort {
		i = 1
	}
	if objectType == aac.ObjectTypeSSR {
		i += 2
	}

	return tnsSFBMax[srIndex][i]
}

// CanDecodeOT returns true if the object type can be decoded.
// Source: ~/dev/faad2/libfaad/common.c:124-172
func CanDecodeOT(objectType aac.ObjectType) bool {
	switch objectType {
	case aac.ObjectTypeLC:
		return true
	case aac.ObjectTypeMain:
		return true
	case aac.ObjectTypeLTP:
		return true
	case aac.ObjectTypeSSR:
		return false // SSR not supported
	case aac.ObjectTypeERLC:
		return true
	case aac.ObjectTypeERLTP:
		return true
	case aac.ObjectTypeLD:
		return true
	case aac.ObjectTypeDRMERLC:
		return true
	default:
		return false
	}
}
