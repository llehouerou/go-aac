// internal/syntax/cpe.go
package syntax

import "github.com/llehouerou/go-aac/internal/bits"

// CPEConfig holds configuration for Channel Pair Element parsing.
// Ported from: channel_pair_element() parameters in ~/dev/faad2/libfaad/syntax.c:698
type CPEConfig struct {
	SFIndex     uint8  // Sample rate index (0-11)
	FrameLength uint16 // Frame length (960 or 1024)
	ObjectType  uint8  // Audio object type
}

// CPEResult holds the result of parsing a Channel Pair Element.
// Ported from: channel_pair_element() return values in ~/dev/faad2/libfaad/syntax.c:698-826
type CPEResult struct {
	Element   Element // Parsed element data (contains ICS1 and ICS2)
	SpecData1 []int16 // Spectral coefficients for channel 1 (1024 or 960 values)
	SpecData2 []int16 // Spectral coefficients for channel 2 (1024 or 960 values)
	Tag       uint8   // Element instance tag (for channel mapping)
}

// ParseChannelPairElement parses a Channel Pair Element (CPE).
// CPE contains two audio channels that may share window configuration
// and use M/S (Mid/Side) stereo coding.
//
// This function:
// 1. Reads the element_instance_tag (4 bits)
// 2. Reads the common_window flag (1 bit)
// 3. If common_window, parses shared ics_info and M/S mask
// 4. Parses individual_channel_stream for both channels
//
// The spectral reconstruction (M/S decoding, inverse quantization, filter bank)
// is handled separately in Phase 4.
//
// Ported from: channel_pair_element() in ~/dev/faad2/libfaad/syntax.c:698-826
func ParseChannelPairElement(r *bits.Reader, channels uint8, cfg *CPEConfig) (*CPEResult, error) {
	result := &CPEResult{
		SpecData1: make([]int16, cfg.FrameLength),
		SpecData2: make([]int16, cfg.FrameLength),
	}

	// Initialize element
	// Ported from: syntax.c:709-710
	result.Element.Channel = channels
	result.Element.PairedChannel = int16(channels + 1)

	// Read element_instance_tag (4 bits)
	// Ported from: syntax.c:712-714
	result.Element.ElementInstanceTag = uint8(r.GetBits(LenTag))
	result.Tag = result.Element.ElementInstanceTag

	// Read common_window flag (1 bit)
	// Ported from: syntax.c:716-717
	result.Element.CommonWindow = r.Get1Bit() != 0

	if result.Element.CommonWindow {
		// Parse shared ics_info
		// Ported from: syntax.c:719-721
		icsCfg := &ICSInfoConfig{
			SFIndex:      cfg.SFIndex,
			FrameLength:  cfg.FrameLength,
			ObjectType:   cfg.ObjectType,
			CommonWindow: true,
		}
		if err := ParseICSInfo(r, &result.Element.ICS1, icsCfg); err != nil {
			return nil, err
		}

		// Parse M/S mask
		// Ported from: syntax.c:723-741
		if err := parseMSMask(r, &result.Element.ICS1); err != nil {
			return nil, err
		}

		// Copy ICS1 to ICS2 (they share window configuration)
		// Ported from: syntax.c:764
		result.Element.ICS2 = result.Element.ICS1
	} else {
		// No common window - M/S stereo not used
		// Ported from: syntax.c:765-767
		result.Element.ICS1.MSMaskPresent = 0
	}

	// Parse individual channel stream for channel 1
	// Ported from: syntax.c:769-773
	ics1Cfg := &ICSConfig{
		SFIndex:      cfg.SFIndex,
		FrameLength:  cfg.FrameLength,
		ObjectType:   cfg.ObjectType,
		CommonWindow: result.Element.CommonWindow,
		ScalFlag:     false,
	}
	if err := ParseIndividualChannelStream(r, &result.Element, &result.Element.ICS1, result.SpecData1, ics1Cfg); err != nil {
		return nil, err
	}

	// Parse individual channel stream for channel 2
	// Ported from: syntax.c:797-801
	ics2Cfg := &ICSConfig{
		SFIndex:      cfg.SFIndex,
		FrameLength:  cfg.FrameLength,
		ObjectType:   cfg.ObjectType,
		CommonWindow: result.Element.CommonWindow,
		ScalFlag:     false,
	}
	if err := ParseIndividualChannelStream(r, &result.Element, &result.Element.ICS2, result.SpecData2, ics2Cfg); err != nil {
		return nil, err
	}

	// Note: SBR fill element handling is done in Phase 8
	// Note: reconstruct_channel_pair is done in Phase 4

	return result, nil
}

// parseMSMask parses the M/S stereo mask from the bitstream.
// Ported from: channel_pair_element() ms_mask section in syntax.c:723-741
func parseMSMask(r *bits.Reader, ics *ICStream) error {
	// Read ms_mask_present (2 bits)
	ics.MSMaskPresent = uint8(r.GetBits(2))

	// Value 3 is reserved
	if ics.MSMaskPresent == 3 {
		return ErrMSMaskReserved
	}

	// If ms_mask_present == 1, read per-band mask
	if ics.MSMaskPresent == 1 {
		for g := uint8(0); g < ics.NumWindowGroups; g++ {
			for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
				ics.MSUsed[g][sfb] = r.Get1Bit()
			}
		}
	}
	// If ms_mask_present == 2, all bands use M/S (handled in spectrum reconstruction)
	// If ms_mask_present == 0, no M/S stereo

	return nil
}
