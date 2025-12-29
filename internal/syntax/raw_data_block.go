// internal/syntax/raw_data_block.go
//
// # Raw Data Block Parsing
//
// This file implements:
// - ParseRawDataBlock: Main entry point for parsing AAC frames
//
// The raw_data_block() is the core parsing loop that reads and dispatches
// all syntax elements (SCE, CPE, LFE, CCE, DSE, PCE, FIL) in an AAC frame.
//
// Ported from: ~/dev/faad2/libfaad/syntax.c:449-648
package syntax

// RawDataBlockConfig holds configuration for raw data block parsing.
// Ported from: raw_data_block() parameters in ~/dev/faad2/libfaad/syntax.c:449-450
type RawDataBlockConfig struct {
	SFIndex              uint8  // Sample rate index (0-11)
	FrameLength          uint16 // Frame length (960 or 1024)
	ObjectType           uint8  // Audio object type
	ChannelConfiguration uint8  // Channel configuration (0-7)
}

// RawDataBlockResult holds the result of parsing a raw data block.
// Ported from: raw_data_block() local variables in ~/dev/faad2/libfaad/syntax.c:452-458
type RawDataBlockResult struct {
	// Frame statistics (from hDecoder state)
	NumChannels  uint8     // Total channels in this frame (fr_channels)
	NumElements  uint8     // Number of elements parsed (fr_ch_ele)
	FirstElement ElementID // First syntax element type (first_syn_ele)
	HasLFE       bool      // True if LFE element present (has_lfe)

	// Parsed elements - fixed arrays to avoid allocations
	// Up to MaxSyntaxElements of each type can be present
	SCEResults [MaxSyntaxElements]*SCEResult // Single Channel Elements
	CPEResults [MaxSyntaxElements]*CPEResult // Channel Pair Elements
	CCEResults [MaxSyntaxElements]*CCEResult // Coupling Channel Elements
	SCECount   uint8                         // Number of SCE elements
	CPECount   uint8                         // Number of CPE elements
	LFECount   uint8                         // Number of LFE elements
	CCECount   uint8                         // Number of CCE elements

	// DRC info is updated in place (passed by reference)
	// PCE is returned separately if present
	PCE *ProgramConfig
}
