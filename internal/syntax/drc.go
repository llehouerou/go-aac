// internal/syntax/drc.go
package syntax

// DRCInfo contains Dynamic Range Control information.
// DRC allows controlling the dynamic range of the output signal
// for different playback environments.
//
// Ported from: drc_info in ~/dev/faad2/libfaad/structs.h:85-101
type DRCInfo struct {
	Present             bool  // DRC data present in stream
	NumBands            uint8 // Number of DRC bands
	PCEInstanceTag      uint8 // Associated PCE instance tag
	ExcludedChnsPresent bool  // Excluded channels present

	BandTop      [17]uint8 // Top of each DRC band (SFB)
	ProgRefLevel uint8     // Program reference level

	DynRngSgn [17]uint8 // Dynamic range sign
	DynRngCtl [17]uint8 // Dynamic range control

	ExcludeMask            [MaxChannels]uint8 // Channel exclude mask
	AdditionalExcludedChns [MaxChannels]uint8 // Additional excluded channels
}
