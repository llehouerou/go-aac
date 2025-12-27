// internal/syntax/ics.go
package syntax

// ICStream represents an Individual Channel Stream.
// This is the core data structure for a single audio channel,
// containing window info, section data, scale factors, and tool flags.
//
// Ported from: ic_stream in ~/dev/faad2/libfaad/structs.h:240-301
type ICStream struct {
	// Window configuration
	MaxSFB              uint8                  // Maximum scale factor band used
	GlobalGain          uint8                  // Global gain value (0-255)
	NumSWB              uint8                  // Number of scale factor bands
	NumWindowGroups     uint8                  // Number of window groups (1 for long, 1-8 for short)
	NumWindows          uint8                  // Number of windows (1 for long, 8 for short)
	WindowSequence      WindowSequence         // Window sequence type
	WindowGroupLength   [MaxWindowGroups]uint8 // Number of windows per group
	WindowShape         uint8                  // Window shape (0=sine, 1=KBD)
	ScaleFactorGrouping uint8                  // Scale factor band grouping pattern

	// Scale factor band offsets (calculated from tables)
	SectSFBOffset [MaxWindowGroups][15 * 8]uint16 // Section SFB offsets per group
	SWBOffset     [52]uint16                      // SFB offsets for this frame
	SWBOffsetMax  uint16                          // Maximum SFB offset

	// Section data (Huffman codebook assignment)
	SectCB    [MaxWindowGroups][15 * 8]uint8  // Codebook index per section
	SectStart [MaxWindowGroups][15 * 8]uint16 // Section start SFB
	SectEnd   [MaxWindowGroups][15 * 8]uint16 // Section end SFB
	SFBCB     [MaxWindowGroups][8 * 15]uint8  // Codebook per SFB (derived)
	NumSec    [MaxWindowGroups]uint8          // Number of sections per group

	// Scale factors
	ScaleFactors [MaxWindowGroups][MaxSFB]int16 // Scale factors (0-255, except for noise and intensity)

	// M/S stereo info (only used for CPE)
	MSMaskPresent uint8                          // 0=none, 1=per-band, 2=all
	MSUsed        [MaxWindowGroups][MaxSFB]uint8 // M/S mask per SFB

	// Tool usage flags
	NoiseUsed              bool // True if noise (PNS) bands present
	IsUsed                 bool // True if intensity stereo bands present
	PulseDataPresent       bool // True if pulse data follows
	TNSDataPresent         bool // True if TNS data follows
	GainControlDataPresent bool // True if gain control (SSR) data follows
	PredictorDataPresent   bool // True if predictor (MAIN/LTP) data follows

	// Embedded tool data
	Pul PulseInfo // Pulse data (if present)
	TNS TNSInfo   // TNS data (if present)
}
