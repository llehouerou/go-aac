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

// ICS info errors.
var (
	// ErrICSReservedBit indicates ics_reserved_bit is not 0.
	ErrICSReservedBit = errors.New("syntax: ics_reserved_bit must be 0")
)

// LTP errors.
var (
	// ErrLTPLagTooLarge indicates LTP lag exceeds 2 * frame_length.
	ErrLTPLagTooLarge = errors.New("syntax: LTP lag exceeds maximum")
)

// Section data errors.
var (
	// ErrBitstreamRead indicates a bitstream read error occurred.
	ErrBitstreamRead = errors.New("syntax: bitstream read error")

	// ErrSectionLimit indicates the section limit was exceeded.
	ErrSectionLimit = errors.New("syntax: section limit exceeded")

	// ErrReservedCodebook indicates reserved codebook 12 was used.
	ErrReservedCodebook = errors.New("syntax: reserved codebook 12 used")

	// ErrSectionLength indicates the section length exceeds the limit.
	ErrSectionLength = errors.New("syntax: section length exceeds limit")

	// ErrSectionCoverage indicates sections do not cover all SFBs.
	ErrSectionCoverage = errors.New("syntax: sections do not cover all SFBs")
)

// Scale factor errors.
var (
	// ErrScaleFactorRange indicates a scale factor is out of the valid range [0, 255].
	ErrScaleFactorRange = errors.New("syntax: scale factor out of range [0, 255]")
)

// Pulse data errors.
var (
	// ErrPulseStartSFB indicates pulse_start_sfb exceeds num_swb.
	ErrPulseStartSFB = errors.New("syntax: pulse_start_sfb exceeds num_swb")

	// ErrPulseInShortBlock indicates pulse coding is not allowed in short blocks.
	ErrPulseInShortBlock = errors.New("syntax: pulse coding not allowed in short blocks")
)

// Gain control errors.
var (
	// ErrGainControlNotSupported indicates gain control (SSR profile) is not supported.
	ErrGainControlNotSupported = errors.New("syntax: gain control (SSR) not supported")
)

// SCE/LFE errors.
var (
	// ErrIntensityStereoInSCE indicates intensity stereo was used in a single channel element.
	// Intensity stereo is only valid in Channel Pair Elements (CPE).
	// FAAD2 error code: 32
	ErrIntensityStereoInSCE = errors.New("syntax: intensity stereo not allowed in single channel element")
)

// CPE errors.
var (
	// ErrMSMaskReserved indicates ms_mask_present has reserved value 3.
	// FAAD2 error code: 32
	ErrMSMaskReserved = errors.New("syntax: ms_mask_present value 3 is reserved")
)

// CCE errors.
var (
	// ErrIntensityStereoInCCE indicates intensity stereo was used in a coupling channel element.
	// Intensity stereo is not valid in CCE.
	// FAAD2 error code: 32
	ErrIntensityStereoInCCE = errors.New("syntax: intensity stereo not allowed in coupling channel element")
)

// Raw data block errors.
var (
	// ErrPCENotFirst indicates PCE appeared after other elements in the frame.
	// Per ISO/IEC 14496-4:5.6.4.1.2.1.3, PCE in raw_data_block should be ignored
	// but FAAD2 returns error 31 when PCE is not the first element.
	ErrPCENotFirst = errors.New("syntax: PCE must be first element in frame")

	// ErrCCENotSupported indicates CCE is present but coupling decoding is disabled.
	// FAAD2 returns error 6 when COUPLING_DEC is not defined.
	ErrCCENotSupported = errors.New("syntax: coupling channel element not supported")

	// ErrUnknownElement indicates an unknown or invalid element ID was encountered.
	// FAAD2 error code: 32
	ErrUnknownElement = errors.New("syntax: unknown element type")

	// ErrBitstreamError indicates a bitstream read error occurred.
	// FAAD2 error code: 32
	ErrBitstreamError = errors.New("syntax: bitstream error")
)
