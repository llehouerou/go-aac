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
