// internal/syntax/ltp.go
package syntax

// LTPInfo contains Long Term Prediction data.
// LTP uses previously decoded samples to improve coding efficiency.
// This is used only for the LTP audio object type.
//
// Ported from: ltp_info in ~/dev/faad2/libfaad/structs.h:186-197
type LTPInfo struct {
	LastBand    uint8  // Last band using LTP
	DataPresent bool   // LTP data present
	Lag         uint16 // LTP lag in samples (0-2047)
	LagUpdate   bool   // Lag value updated
	Coef        uint8  // LTP coefficient index (0-7)

	// Per-SFB usage for long windows
	LongUsed [MaxSFB]bool

	// Per-window info for short windows
	ShortUsed       [8]bool  // LTP used per short window
	ShortLagPresent [8]bool  // Short lag present per window
	ShortLag        [8]uint8 // Short window lag values
}
