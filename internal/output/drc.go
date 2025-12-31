// internal/output/drc.go
package output

import (
	"math"

	"github.com/llehouerou/go-aac/internal/syntax"
)

// DRCRefLevel is the reference level for DRC calculations.
// Represents -20 dB (20 * 4 = 80 in quarter-dB units).
//
// Ported from: DRC_REF_LEVEL in ~/dev/faad2/libfaad/drc.h:38
const DRCRefLevel = 80

// DRC holds the Dynamic Range Control state.
//
// Cut and Boost are application-configurable parameters (0.0 to 1.0):
// - Cut: Controls compression (reduces dynamic range)
// - Boost: Controls expansion (increases quiet passages)
//
// Ported from: drc_info in ~/dev/faad2/libfaad/structs.h:85-101
type DRC struct {
	Cut   float32 // Compression control (ctrl1 in FAAD2)
	Boost float32 // Boost control (ctrl2 in FAAD2)
}

// NewDRC creates a new DRC processor with the specified cut and boost factors.
//
// Parameters:
// - cut: Compression factor (0.0 = no compression, 1.0 = full compression)
// - boost: Boost factor (0.0 = no boost, 1.0 = full boost)
//
// Ported from: drc_init() in ~/dev/faad2/libfaad/drc.c:38-52
func NewDRC(cut, boost float32) *DRC {
	return &DRC{
		Cut:   cut,
		Boost: boost,
	}
}

// Decode applies Dynamic Range Control to spectral coefficients.
//
// The DRC info is parsed from fill elements in the bitstream.
// This function modifies spec in-place.
//
// Ported from: drc_decode() in ~/dev/faad2/libfaad/drc.c:112-172
func (d *DRC) Decode(info *syntax.DRCInfo, spec []float32) {
	if info == nil || info.NumBands == 0 {
		return
	}

	bottom := uint16(0)
	numBands := info.NumBands

	// Clamp numBands to array size
	if numBands > 17 {
		numBands = 17
	}

	// Default band_top for single band
	if numBands == 1 {
		info.BandTop[0] = 1024/4 - 1
	}

	for bd := uint8(0); bd < numBands; bd++ {
		top := uint16(4 * (uint16(info.BandTop[bd]) + 1))

		// Clamp top to spec length
		if int(top) > len(spec) {
			top = uint16(len(spec))
		}

		// Decode DRC gain factor
		var exp float32
		if info.DynRngSgn[bd] == 1 {
			// Compress
			exp = ((-d.Cut * float32(info.DynRngCtl[bd])) -
				float32(DRCRefLevel-int(info.ProgRefLevel))) / 24.0
		} else {
			// Boost
			exp = ((d.Boost * float32(info.DynRngCtl[bd])) -
				float32(DRCRefLevel-int(info.ProgRefLevel))) / 24.0
		}

		factor := float32(math.Pow(2.0, float64(exp)))

		// Apply gain factor
		for i := bottom; i < top; i++ {
			spec[i] *= factor
		}

		bottom = top
	}
}
