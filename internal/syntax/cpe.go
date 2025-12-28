// internal/syntax/cpe.go
package syntax

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
