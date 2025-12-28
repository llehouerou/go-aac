// internal/syntax/cce.go
package syntax

import (
	"github.com/llehouerou/go-aac/internal/bits"
	"github.com/llehouerou/go-aac/internal/huffman"
)

// CCEConfig holds configuration for Coupling Channel Element parsing.
// Ported from: coupling_channel_element() parameters in ~/dev/faad2/libfaad/syntax.c:987
type CCEConfig struct {
	SFIndex     uint8  // Sample rate index (0-11)
	FrameLength uint16 // Frame length (960 or 1024)
	ObjectType  uint8  // Audio object type
}

// CCECoupledElement holds information about a coupled element target.
// Ported from: coupling_channel_element() loop in ~/dev/faad2/libfaad/syntax.c:1006-1027
type CCECoupledElement struct {
	TargetIsCPE bool  // True if target is a CPE (vs SCE)
	TargetTag   uint8 // Target element instance tag (0-15)
	CCL         bool  // Apply coupling to left channel (only if TargetIsCPE)
	CCR         bool  // Apply coupling to right channel (only if TargetIsCPE)
}

// CCEResult holds the result of parsing a Coupling Channel Element.
// Note: CCE data is parsed but not used for decoding (rarely used in practice).
// Ported from: coupling_channel_element() in ~/dev/faad2/libfaad/syntax.c:987-1076
type CCEResult struct {
	Tag                 uint8                // Element instance tag (0-15)
	IndSwCCEFlag        bool                 // Independently switched CCE
	NumCoupledElements  uint8                // Number of coupled elements (0-7)
	CoupledElements     [8]CCECoupledElement // Coupled element targets
	NumGainElementLists uint8                // Number of gain element lists
	CCDomain            bool                 // Coupling domain (0=before TNS, 1=after TNS)
	GainElementSign     bool                 // Sign of gain elements
	GainElementScale    uint8                // Scale of gain elements (0-3)
	Element             Element              // Parsed ICS element
	SpecData            []int16              // Spectral data (parsed but not used)
}

// parseCCEHeader parses the CCE header fields.
// Ported from: coupling_channel_element() in ~/dev/faad2/libfaad/syntax.c:998-1034
func parseCCEHeader(r *bits.Reader, result *CCEResult) error {
	// Read element_instance_tag (4 bits)
	// Ported from: syntax.c:998-999
	result.Tag = uint8(r.GetBits(LenTag))

	// Read ind_sw_cce_flag (1 bit)
	// Ported from: syntax.c:1001-1002
	result.IndSwCCEFlag = r.Get1Bit() != 0

	// Read num_coupled_elements (3 bits)
	// Ported from: syntax.c:1003-1004
	result.NumCoupledElements = uint8(r.GetBits(3))

	// Parse coupled element targets
	// Ported from: syntax.c:1006-1027
	result.NumGainElementLists = 0

	for c := uint8(0); c <= result.NumCoupledElements; c++ {
		result.NumGainElementLists++

		// Read cc_target_is_cpe (1 bit)
		result.CoupledElements[c].TargetIsCPE = r.Get1Bit() != 0

		// Read cc_target_tag_select (4 bits)
		result.CoupledElements[c].TargetTag = uint8(r.GetBits(4))

		if result.CoupledElements[c].TargetIsCPE {
			// Read cc_l and cc_r (1 bit each)
			result.CoupledElements[c].CCL = r.Get1Bit() != 0
			result.CoupledElements[c].CCR = r.Get1Bit() != 0

			// If both channels are coupled, we need an extra gain element list
			if result.CoupledElements[c].CCL && result.CoupledElements[c].CCR {
				result.NumGainElementLists++
			}
		}
	}

	// Read cc_domain (1 bit)
	// Ported from: syntax.c:1029-1030
	result.CCDomain = r.Get1Bit() != 0

	// Read gain_element_sign (1 bit)
	// Ported from: syntax.c:1031-1032
	result.GainElementSign = r.Get1Bit() != 0

	// Read gain_element_scale (2 bits)
	// Ported from: syntax.c:1033-1034
	result.GainElementScale = uint8(r.GetBits(2))

	return nil
}

// ParseCouplingChannelElement parses a Coupling Channel Element (CCE).
// CCE allows coupling channels to share gain information with target channels.
//
// This function:
// 1. Parses the CCE header (tag, coupled elements, domain flags)
// 2. Parses the individual channel stream
// 3. Validates that intensity stereo is not used (illegal in CCE)
// 4. Parses gain element lists
//
// Note: CCE data is parsed but discarded (rarely used in practice).
// Ported from: coupling_channel_element() in ~/dev/faad2/libfaad/syntax.c:987-1076
func ParseCouplingChannelElement(r *bits.Reader, cfg *CCEConfig) (*CCEResult, error) {
	result := &CCEResult{
		SpecData: make([]int16, cfg.FrameLength),
	}

	// Initialize element
	result.Element.PairedChannel = -1 // No paired channel for CCE
	result.Element.CommonWindow = false

	// Parse CCE header
	// Ported from: syntax.c:998-1034
	if err := parseCCEHeader(r, result); err != nil {
		return nil, err
	}

	// Parse individual channel stream
	// Ported from: syntax.c:1036-1040
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

	// Intensity stereo is not allowed in coupling channel elements
	// Ported from: syntax.c:1042-1044
	if result.Element.ICS1.IsUsed {
		return nil, ErrIntensityStereoInCCE
	}

	// Parse gain element lists
	// Ported from: syntax.c:1046-1073
	if err := parseCCEGainElements(r, result); err != nil {
		return nil, err
	}

	return result, nil
}

// parseCCEGainElements parses the gain element lists for CCE.
// Ported from: coupling_channel_element() in ~/dev/faad2/libfaad/syntax.c:1046-1073
func parseCCEGainElements(r *bits.Reader, result *CCEResult) error {
	ics := &result.Element.ICS1

	// For each gain element list (starting from c=1)
	// Ported from: syntax.c:1046
	for c := uint8(1); c < result.NumGainElementLists; c++ {
		var cge bool

		if result.IndSwCCEFlag {
			// For independently switched CCE, always use common gain
			// Ported from: syntax.c:1050-1052
			cge = true
		} else {
			// Read common_gain_element_present (1 bit)
			// Ported from: syntax.c:1054-1055
			cge = r.Get1Bit() != 0
		}

		if cge {
			// Common gain element: decode single huffman scale factor
			// Ported from: syntax.c:1058-1060
			_ = huffman.ScaleFactor(r)
		} else {
			// Per-SFB gain elements: decode scale factor for each non-zero SFB
			// Ported from: syntax.c:1062-1071
			for g := uint8(0); g < ics.NumWindowGroups; g++ {
				for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
					if ics.SFBCB[g][sfb] != uint8(huffman.ZeroHCB) {
						_ = huffman.ScaleFactor(r)
					}
				}
			}
		}
	}

	return nil
}
