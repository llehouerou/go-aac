// internal/output/drc.go
package output

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
