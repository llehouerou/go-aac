// internal/syntax/sce.go
package syntax

import "github.com/llehouerou/go-aac/internal/bits"

// Ensure bits is imported (will be used by ParseSingleChannelElement).
var _ = bits.Reader{}

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
