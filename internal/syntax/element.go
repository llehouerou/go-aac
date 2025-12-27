// internal/syntax/element.go
package syntax

// Element represents a syntax element (SCE, CPE, or LFE).
// SCE (Single Channel Element) uses only ICS1.
// CPE (Channel Pair Element) uses both ICS1 and ICS2.
// LFE (Low Frequency Effects) uses only ICS1.
//
// Ported from: element in ~/dev/faad2/libfaad/structs.h:303-313
type Element struct {
	Channel            uint8 // Output channel index
	PairedChannel      int16 // Paired channel for CPE (-1 if none)
	ElementInstanceTag uint8 // Element instance tag (0-15)
	CommonWindow       bool  // True if CPE shares window info

	ICS1 ICStream // First (or only) channel stream
	ICS2 ICStream // Second channel stream (CPE only)
}
