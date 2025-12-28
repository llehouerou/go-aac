// internal/syntax/asc.go
package syntax

import (
	"errors"
)

// ASC parsing errors.
var (
	// ErrASCNil is returned when nil config is passed.
	ErrASCNil = errors.New("nil AudioSpecificConfig")

	// ErrASCUnsupportedObjectType is returned for unsupported object types.
	ErrASCUnsupportedObjectType = errors.New("unsupported audio object type")

	// ErrASCInvalidSampleRate is returned for invalid sample rate index.
	ErrASCInvalidSampleRate = errors.New("invalid sample rate")

	// ErrASCInvalidChannelConfig is returned for invalid channel configuration.
	ErrASCInvalidChannelConfig = errors.New("invalid channel configuration")

	// ErrASCGAConfigFailed is returned when GASpecificConfig parsing fails.
	ErrASCGAConfigFailed = errors.New("GASpecificConfig parsing failed")

	// ErrASCEPConfigNotSupported is returned for unsupported epConfig values.
	ErrASCEPConfigNotSupported = errors.New("epConfig != 0 not supported")

	// ErrASCBitstreamError is returned for bitstream initialization errors.
	ErrASCBitstreamError = errors.New("bitstream initialization error")
)

// objectTypesTable defines which audio object types can be decoded.
// Ported from: ~/dev/faad2/libfaad/mp4.c:40-117 (ObjectTypesTable)
// This table assumes all optional features are enabled:
// MAIN_DEC, LTP_DEC, SBR_DEC, ERROR_RESILIENCE, LD_DEC, PS_DEC
//
// Note: This table is specifically for ASC (AudioSpecificConfig) parsing.
// A separate CanDecodeOT() function exists in tables/sample_rates.go (from common.c)
// for runtime object type validation. The apparent discrepancy at index 27 is
// intentional: MPEG-4 defines index 27 as "ER Parametric" (not supported here),
// but DRM mode uses index 27 for "DRM ER LC" (supported via CanDecodeOT).
var objectTypesTable = [32]bool{
	false, // 0: NULL
	true,  // 1: AAC Main
	true,  // 2: AAC LC
	false, // 3: AAC SSR (not supported)
	true,  // 4: AAC LTP
	true,  // 5: SBR (HE-AAC)
	false, // 6: AAC Scalable
	false, // 7: TwinVQ
	false, // 8: CELP
	false, // 9: HVXC
	false, // 10: Reserved
	false, // 11: Reserved
	false, // 12: TTSI
	false, // 13: Main synthetic
	false, // 14: Wavetable synthesis
	false, // 15: General MIDI
	false, // 16: Algorithmic Synthesis and Audio FX
	true,  // 17: ER AAC LC
	false, // 18: Reserved
	true,  // 19: ER AAC LTP
	false, // 20: ER AAC scalable
	false, // 21: ER TwinVQ
	false, // 22: ER BSAC
	true,  // 23: ER AAC LD
	false, // 24: ER CELP
	false, // 25: ER HVXC
	false, // 26: ER HILN
	false, // 27: ER Parametric
	false, // 28: Reserved
	true,  // 29: AAC LC + SBR + PS (HE-AACv2)
	false, // 30: Reserved
	false, // 31: Reserved
}

// isObjectTypeSupported returns true if the audio object type can be decoded.
// Ported from: ~/dev/faad2/libfaad/mp4.c:40-117
func isObjectTypeSupported(objType uint8) bool {
	if objType >= 32 {
		return false
	}
	return objectTypesTable[objType]
}
