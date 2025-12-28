// Package syntax implements AAC bitstream syntax parsing.
// This file contains window grouping information calculation.
package syntax

import (
	"github.com/llehouerou/go-aac/internal/tables"
)

// WindowGroupingInfo calculates window grouping information for an ICS.
// It sets up the number of windows, window groups, and SFB offsets
// based on the window sequence and sample rate.
//
// Ported from: window_grouping_info() in ~/dev/faad2/libfaad/specrec.c:302-428
func WindowGroupingInfo(ics *ICStream, sfIndex uint8, frameLength uint16) error {
	if sfIndex >= 12 {
		return ErrInvalidSRIndex
	}

	switch ics.WindowSequence {
	case OnlyLongSequence, LongStartSequence, LongStopSequence:
		return windowGroupingLong(ics, sfIndex, frameLength)
	case EightShortSequence:
		return windowGroupingShort(ics, sfIndex, frameLength)
	default:
		return ErrInvalidWindowSequence
	}
}

// windowGroupingLong handles long window sequences.
// Ported from: window_grouping_info() cases ONLY_LONG_SEQUENCE, LONG_START_SEQUENCE, LONG_STOP_SEQUENCE
// in ~/dev/faad2/libfaad/specrec.c:312-375
func windowGroupingLong(ics *ICStream, sfIndex uint8, frameLength uint16) error {
	ics.NumWindows = 1
	ics.NumWindowGroups = 1
	ics.WindowGroupLength[0] = 1

	// Get number of SFBs for this sample rate and frame length
	numSWB, err := tables.GetNumSWB(sfIndex, frameLength, false)
	if err != nil {
		return err
	}
	ics.NumSWB = numSWB

	// Validate max_sfb
	if ics.MaxSFB > ics.NumSWB {
		return ErrMaxSFBTooLarge
	}

	// Get SFB offsets
	offsets, err := tables.GetSWBOffset(sfIndex, frameLength, false)
	if err != nil {
		return err
	}

	// Copy to sect_sfb_offset[0] and swb_offset
	for i := uint8(0); i < ics.NumSWB; i++ {
		ics.SectSFBOffset[0][i] = offsets[i]
		ics.SWBOffset[i] = offsets[i]
	}
	ics.SectSFBOffset[0][ics.NumSWB] = frameLength
	ics.SWBOffset[ics.NumSWB] = frameLength
	ics.SWBOffsetMax = frameLength

	return nil
}

// windowGroupingShort handles eight short window sequences.
// Ported from: window_grouping_info() case EIGHT_SHORT_SEQUENCE
// in ~/dev/faad2/libfaad/specrec.c:376-424
func windowGroupingShort(ics *ICStream, sfIndex uint8, frameLength uint16) error {
	ics.NumWindows = 8
	ics.NumWindowGroups = 1
	ics.WindowGroupLength[0] = 1

	// Get number of SFBs for short windows
	numSWB, err := tables.GetNumSWB(sfIndex, frameLength, true)
	if err != nil {
		return err
	}
	ics.NumSWB = numSWB

	// Validate max_sfb
	if ics.MaxSFB > ics.NumSWB {
		return ErrMaxSFBTooLarge
	}

	// Get SFB offsets for short windows
	offsets, err := tables.GetSWBOffset(sfIndex, frameLength, true)
	if err != nil {
		return err
	}

	// Copy to swb_offset
	for i := uint8(0); i < ics.NumSWB; i++ {
		ics.SWBOffset[i] = offsets[i]
	}
	shortLen := frameLength / 8
	ics.SWBOffset[ics.NumSWB] = shortLen
	ics.SWBOffsetMax = shortLen

	// Calculate window groups from scale_factor_grouping
	// Bits 6-0 indicate grouping: bit N=0 means new group starts at window N+1
	// Ported from: specrec.c:392-400
	for i := uint8(0); i < 7; i++ {
		if !bitSet(ics.ScaleFactorGrouping, 6-i) {
			// New group
			ics.NumWindowGroups++
			ics.WindowGroupLength[ics.NumWindowGroups-1] = 1
		} else {
			// Same group
			ics.WindowGroupLength[ics.NumWindowGroups-1]++
		}
	}

	// Calculate sect_sfb_offset for each group
	// Ported from: specrec.c:403-423
	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		sectSFB := uint8(0)
		offset := uint16(0)

		for i := uint8(0); i < ics.NumSWB; i++ {
			var width uint16
			if i+1 == ics.NumSWB {
				width = shortLen - offsets[i]
			} else {
				width = offsets[i+1] - offsets[i]
			}
			width *= uint16(ics.WindowGroupLength[g])
			ics.SectSFBOffset[g][sectSFB] = offset
			sectSFB++
			offset += width
		}
		ics.SectSFBOffset[g][sectSFB] = offset
	}

	return nil
}

// bitSet returns true if bit B is set in A.
// Ported from: #define bit_set(A, B) ((A) & (1<<(B))) in specrec.c:287
func bitSet(a uint8, b uint8) bool {
	return (a & (1 << b)) != 0
}
