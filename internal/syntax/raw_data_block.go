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

import "github.com/llehouerou/go-aac/internal/bits"

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

// ParseRawDataBlock parses a raw_data_block() from the bitstream.
// This is the main entry point for parsing AAC frame data.
//
// The function reads syntax elements in a loop until ID_END (0x7) is
// encountered. Each element is parsed by its respective parser and
// the results are collected in RawDataBlockResult.
//
// Ported from: raw_data_block() in ~/dev/faad2/libfaad/syntax.c:449-648
func ParseRawDataBlock(r *bits.Reader, cfg *RawDataBlockConfig, drc *DRCInfo) (*RawDataBlockResult, error) {
	result := &RawDataBlockResult{
		FirstElement: InvalidElementID,
	}

	// Main parsing loop
	// Ported from: syntax.c:465-544
	for {
		// Read element ID (3 bits)
		idSynEle := ElementID(r.GetBits(LenSEID))

		if idSynEle == IDEND {
			break
		}

		// Track elements
		result.NumElements++
		if result.FirstElement == InvalidElementID {
			result.FirstElement = idSynEle
		}

		switch idSynEle {
		case IDSCE:
			// Parse Single Channel Element
			// Ported from: decode_sce_lfe() call in syntax.c:472
			sceCfg := &SCEConfig{
				SFIndex:     cfg.SFIndex,
				FrameLength: cfg.FrameLength,
				ObjectType:  cfg.ObjectType,
			}
			sceResult, err := ParseSingleChannelElement(r, result.NumChannels, sceCfg)
			if err != nil {
				return nil, err
			}
			result.SCEResults[result.SCECount] = sceResult
			result.SCECount++
			result.NumChannels++

		case IDCPE:
			// TODO: Parse CPE (Task 6)

		case IDLFE:
			// TODO: Parse LFE (Task 7)

		case IDCCE:
			// TODO: Parse CCE (Task 8)

		case IDDSE:
			// Parse DSE (data is discarded)
			_ = ParseDataStreamElement(r)

		case IDPCE:
			// PCE must be first element
			if result.NumElements != 1 {
				return nil, ErrPCENotFirst
			}
			pce, err := ParsePCE(r)
			if err != nil {
				return nil, err
			}
			result.PCE = pce

		case IDFIL:
			// Parse fill element
			if err := ParseFillElement(r, drc); err != nil {
				return nil, err
			}

		default:
			return nil, ErrUnknownElement
		}

		// Check for bitstream errors
		if r.Error() {
			return nil, ErrBitstreamError
		}
	}

	// Byte align after parsing
	// Ported from: syntax.c:644
	r.ByteAlign()

	return result, nil
}
