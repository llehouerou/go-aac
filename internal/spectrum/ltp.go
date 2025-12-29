// internal/spectrum/ltp.go
package spectrum

import (
	"math"

	"github.com/llehouerou/go-aac"
)

// ltpCodebook contains the 8 LTP coefficient values.
// The transmitted coef index (0-7) indexes into this table.
//
// Ported from: codebook[] in ~/dev/faad2/libfaad/lt_predict.c:68-78
var ltpCodebook = [8]float64{
	0.570829,
	0.696616,
	0.813004,
	0.911304,
	0.984900,
	1.067894,
	1.194601,
	1.369533,
}

// IsLTPObjectType returns true if the given object type supports LTP.
//
// Ported from: is_ltp_ot() in ~/dev/faad2/libfaad/lt_predict.c:49-66
func IsLTPObjectType(objectType aac.ObjectType) bool {
	switch objectType {
	case aac.ObjectTypeLTP, aac.ObjectTypeERLTP, aac.ObjectTypeLD:
		return true
	default:
		return false
	}
}

// realToInt16 converts a floating-point sample to int16 with rounding and clamping.
// Uses round-half-away-from-zero for .5 values.
//
// Ported from: real_to_int16() in ~/dev/faad2/libfaad/lt_predict.c:152-170
func realToInt16(sigIn float64) int16 {
	// Round to nearest integer (away from zero for .5)
	var rounded float64
	if sigIn >= 0 {
		rounded = math.Floor(sigIn + 0.5)
		if rounded >= 32768.0 {
			return 32767
		}
	} else {
		rounded = math.Ceil(sigIn - 0.5)
		if rounded <= -32768.0 {
			return -32768
		}
	}
	return int16(rounded)
}
