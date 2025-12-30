// internal/syntax/ics.go
package syntax

import "github.com/llehouerou/go-aac/internal/bits"

// SideInfoConfig holds configuration for side info parsing.
type SideInfoConfig struct {
	SFIndex      uint8
	FrameLength  uint16
	ObjectType   uint8
	CommonWindow bool
	ScalFlag     bool // True for scalable AAC
}

// ICSConfig holds configuration for ICS parsing.
type ICSConfig struct {
	SFIndex      uint8
	FrameLength  uint16
	ObjectType   uint8
	CommonWindow bool
	ScalFlag     bool
}

// ParseSideInfo parses side information for an ICS.
// Ported from: side_info() in ~/dev/faad2/libfaad/syntax.c:1578-1668
func ParseSideInfo(r *bits.Reader, ele *Element, ics *ICStream, cfg *SideInfoConfig) error {
	// Read global gain (8 bits)
	ics.GlobalGain = uint8(r.GetBits(8))

	// Parse ics_info if not common_window and not scalable
	if !ele.CommonWindow && !cfg.ScalFlag {
		icsCfg := &ICSInfoConfig{
			SFIndex:      cfg.SFIndex,
			FrameLength:  cfg.FrameLength,
			ObjectType:   cfg.ObjectType,
			CommonWindow: ele.CommonWindow,
		}
		if err := ParseICSInfo(r, ics, icsCfg); err != nil {
			return err
		}
	}

	// Parse section data
	if err := ParseSectionData(r, ics); err != nil {
		return err
	}

	// Parse scale factor data
	if err := ParseScaleFactorData(r, ics); err != nil {
		return err
	}

	// Only parse tool data if not scalable
	if !cfg.ScalFlag {
		// Pulse data
		ics.PulseDataPresent = r.Get1Bit() != 0
		if ics.PulseDataPresent {
			if err := ParsePulseData(r, ics, &ics.Pul); err != nil {
				return err
			}
		}

		// TNS data
		ics.TNSDataPresent = r.Get1Bit() != 0
		if ics.TNSDataPresent {
			// Only parse TNS for non-ER object types
			if cfg.ObjectType < ERObjectStart {
				ParseTNSData(r, ics, &ics.TNS)
			}
		}

		// Gain control data (SSR profile only)
		ics.GainControlDataPresent = r.Get1Bit() != 0
		if ics.GainControlDataPresent {
			return ErrGainControlNotSupported
		}
	}

	return nil
}

// ParseIndividualChannelStream parses a complete individual channel stream.
// This is the main entry point for decoding one channel's data.
//
// Ported from: individual_channel_stream() in ~/dev/faad2/libfaad/syntax.c:1671-1728
func ParseIndividualChannelStream(r *bits.Reader, ele *Element, ics *ICStream, specData []int16, cfg *ICSConfig) error {
	// Parse side info (global gain, section, scale factors, tools)
	sideCfg := &SideInfoConfig{
		SFIndex:      cfg.SFIndex,
		FrameLength:  cfg.FrameLength,
		ObjectType:   cfg.ObjectType,
		CommonWindow: cfg.CommonWindow,
		ScalFlag:     cfg.ScalFlag,
	}
	if err := ParseSideInfo(r, ele, ics, sideCfg); err != nil {
		return err
	}

	// For ER object types, TNS data is parsed here
	if cfg.ObjectType >= ERObjectStart && ics.TNSDataPresent {
		ParseTNSData(r, ics, &ics.TNS)
	}

	// Parse spectral data
	if err := ParseSpectralData(r, ics, specData, cfg.FrameLength); err != nil {
		return err
	}

	// Validate pulse not used with short blocks
	// Note: In FAAD2, pulse_decode is called here for long blocks,
	// but we defer that to the spectrum reconstruction phase.
	if ics.PulseDataPresent && ics.WindowSequence == EightShortSequence {
		return ErrPulseInShortBlock
	}

	return nil
}

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

	// Optional profile data
	LTP  LTPInfo  // LTP data (LTP profile, first predictor)
	LTP2 LTPInfo  // LTP data (LTP profile, second predictor for CPE)
	Pred PredInfo // MAIN profile prediction data
}
