// Package syntax implements AAC bitstream syntax parsing.
// This file contains error definitions for the syntax package.
package syntax

import "errors"

// Window grouping errors.
var (
	// ErrInvalidSRIndex indicates an invalid sample rate index (must be 0-11).
	ErrInvalidSRIndex = errors.New("syntax: invalid sample rate index")

	// ErrInvalidWindowSequence indicates an invalid window sequence type.
	ErrInvalidWindowSequence = errors.New("syntax: invalid window sequence")

	// ErrMaxSFBTooLarge indicates max_sfb exceeds the number of SFBs for this sample rate.
	ErrMaxSFBTooLarge = errors.New("syntax: max_sfb exceeds num_swb")
)
