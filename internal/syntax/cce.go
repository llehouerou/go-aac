// internal/syntax/cce.go
package syntax

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
