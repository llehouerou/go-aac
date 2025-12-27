// Package tables contains lookup tables for AAC decoding.
// This file provides lookup functions for Scalefactor Band (SFB) tables.
// Ported from: ~/dev/faad2/libfaad/specrec.c:66-285
package tables

import "errors"

// ErrInvalidSRIndex indicates an invalid sample rate index.
var ErrInvalidSRIndex = errors.New("tables: invalid sample rate index")

// GetSWBOffset returns the SFB offset table for the given parameters.
// For long windows (isShort=false), returns SWBOffset1024Window[srIndex].
// For short windows (isShort=true), returns SWBOffset128Window[srIndex].
// Returns error if srIndex >= 12.
// Source: ~/dev/faad2/libfaad/specrec.c:221-285
func GetSWBOffset(srIndex uint8, frameLength uint16, isShort bool) ([]uint16, error) {
	if srIndex >= 12 {
		return nil, ErrInvalidSRIndex
	}

	if isShort {
		return SWBOffset128Window[srIndex], nil
	}

	// Long window - only 1024 supported for now (960 for AAC LD would need additional tables)
	return SWBOffset1024Window[srIndex], nil
}

// GetNumSWB returns the number of scale factor window bands.
// For long windows: returns NumSWB1024Window or NumSWB960Window based on frameLength.
// For short windows: returns NumSWB128Window.
// Returns error if srIndex >= 12.
// Source: ~/dev/faad2/libfaad/specrec.c:66-89
func GetNumSWB(srIndex uint8, frameLength uint16, isShort bool) (uint8, error) {
	if srIndex >= 12 {
		return 0, ErrInvalidSRIndex
	}

	if isShort {
		return NumSWB128Window[srIndex], nil
	}

	if frameLength == 960 {
		return NumSWB960Window[srIndex], nil
	}

	return NumSWB1024Window[srIndex], nil
}
