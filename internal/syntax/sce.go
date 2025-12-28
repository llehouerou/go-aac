// internal/syntax/sce.go
package syntax

import "github.com/llehouerou/go-aac/internal/bits"

// SCEConfig holds configuration for Single Channel Element parsing.
// Ported from: single_lfe_channel_element() parameters in ~/dev/faad2/libfaad/syntax.c:1060
type SCEConfig struct {
	SFIndex     uint8  // Sample rate index (0-11)
	FrameLength uint16 // Frame length (960 or 1024)
	ObjectType  uint8  // Audio object type
}

// SCEResult holds the result of parsing a Single Channel Element.
// Ported from: single_lfe_channel_element() return values in ~/dev/faad2/libfaad/syntax.c:1060-1095
type SCEResult struct {
	Element  Element // Parsed element data
	SpecData []int16 // Spectral coefficients (1024 or 960 values)
	Tag      uint8   // Element instance tag (for channel mapping)
}

// ParseSingleChannelElement parses a Single Channel Element (SCE) or LFE element.
// SCE and LFE share the same syntax, differing only in their semantic use
// (SCE for mono audio, LFE for subwoofer channel).
//
// This function:
// 1. Reads the element_instance_tag (4 bits)
// 2. Parses the individual_channel_stream
// 3. Validates that intensity stereo is not used (illegal in SCE/LFE)
//
// The spectral reconstruction (inverse quantization, filter bank) is handled
// separately in Phase 4.
//
// Ported from: single_lfe_channel_element() in ~/dev/faad2/libfaad/syntax.c:652-696
func ParseSingleChannelElement(r *bits.Reader, channel uint8, cfg *SCEConfig) (*SCEResult, error) {
	result := &SCEResult{
		SpecData: make([]int16, cfg.FrameLength),
	}

	// Initialize element
	result.Element.Channel = channel
	result.Element.PairedChannel = -1 // No paired channel for SCE
	result.Element.CommonWindow = false

	// Read element_instance_tag (4 bits)
	// Ported from: syntax.c:660
	result.Element.ElementInstanceTag = uint8(r.GetBits(LenTag))
	result.Tag = result.Element.ElementInstanceTag

	// Parse the individual channel stream
	// Ported from: syntax.c:667
	icsCfg := &ICSConfig{
		SFIndex:      cfg.SFIndex,
		FrameLength:  cfg.FrameLength,
		ObjectType:   cfg.ObjectType,
		CommonWindow: false,
		ScalFlag:     false,
	}

	if err := ParseIndividualChannelStream(r, &result.Element, &result.Element.ICS1, result.SpecData, icsCfg); err != nil {
		return nil, err
	}

	// Intensity stereo is not allowed in single channel elements
	// Ported from: syntax.c:671-673
	if result.Element.ICS1.IsUsed {
		return nil, ErrIntensityStereoInSCE
	}

	// Note: SBR fill element handling is done in Phase 8
	// Note: reconstruct_single_channel is done in Phase 4

	return result, nil
}
