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
