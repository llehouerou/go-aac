// internal/output/drc.go
package output

// DRCRefLevel is the reference level for DRC calculations.
// Represents -20 dB (20 * 4 = 80 in quarter-dB units).
//
// Ported from: DRC_REF_LEVEL in ~/dev/faad2/libfaad/drc.h:38
const DRCRefLevel = 80
