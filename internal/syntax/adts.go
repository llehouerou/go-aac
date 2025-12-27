// internal/syntax/adts.go
package syntax

import (
	"errors"

	"github.com/llehouerou/go-aac/internal/bits"
)

// ADTSSyncword is the 12-bit sync pattern for ADTS frames.
const ADTSSyncword = 0x0FFF

// ErrADTSSyncwordNotFound is returned when no ADTS syncword is found.
var ErrADTSSyncwordNotFound = errors.New("unable to find ADTS syncword")

// MaxSyncSearchBytes is the maximum bytes to search for ADTS syncword.
// Matches FAAD2's limit of 768 bytes.
const MaxSyncSearchBytes = 768

// FindSyncword searches for the ADTS syncword (0xFFF) in the bitstream.
// It will skip up to MaxSyncSearchBytes looking for the sync pattern.
// After finding the syncword, the 12 syncword bits are consumed.
// Returns ErrADTSSyncwordNotFound if no syncword is found.
//
// Ported from: adts_fixed_header() sync recovery loop in
// ~/dev/faad2/libfaad/syntax.c:2466-2482
func FindSyncword(r *bits.Reader) error {
	for i := 0; i < MaxSyncSearchBytes; i++ {
		syncword := r.ShowBits(12)
		if syncword == ADTSSyncword {
			// Found it - consume the syncword
			r.FlushBits(12)
			return nil
		}
		// Skip 8 bits and try again
		r.FlushBits(8)
	}
	return ErrADTSSyncwordNotFound
}

// ADTSHeader contains Audio Data Transport Stream header data.
// ADTS is the most common AAC transport format (used in .aac files).
//
// Header structure (56 bits fixed + 16 bits CRC if present):
//   - syncword: 12 bits (0xFFF)
//   - id: 1 bit (0=MPEG-4, 1=MPEG-2)
//   - layer: 2 bits (always 0)
//   - protection_absent: 1 bit (1=no CRC)
//   - profile: 2 bits (0=Main, 1=LC, 2=SSR, 3=LTP)
//   - sf_index: 4 bits (sample rate index)
//   - private_bit: 1 bit
//   - channel_configuration: 3 bits
//   - original: 1 bit
//   - home: 1 bit
//   - copyright_id_bit: 1 bit
//   - copyright_id_start: 1 bit
//   - frame_length: 13 bits (includes header)
//   - buffer_fullness: 11 bits
//   - no_raw_data_blocks: 2 bits
//   - crc_check: 16 bits (if protection_absent=0)
//
// Ported from: adts_header in ~/dev/faad2/libfaad/structs.h:146-168
type ADTSHeader struct {
	Syncword             uint16 // 12 bits, must be 0xFFF
	ID                   uint8  // 1 bit: 0=MPEG-4, 1=MPEG-2
	Layer                uint8  // 2 bits: always 0
	ProtectionAbsent     bool   // 1 bit: true=no CRC
	Profile              uint8  // 2 bits: object type - 1
	SFIndex              uint8  // 4 bits: sample frequency index
	PrivateBit           bool   // 1 bit
	ChannelConfiguration uint8  // 3 bits: channel config
	Original             bool   // 1 bit
	Home                 bool   // 1 bit
	Emphasis             uint8  // 2 bits (MPEG-2 only)

	// Variable header
	CopyrightIDBit         bool   // 1 bit
	CopyrightIDStart       bool   // 1 bit
	AACFrameLength         uint16 // 13 bits: total frame bytes
	ADTSBufferFullness     uint16 // 11 bits: buffer fullness
	CRCCheck               uint16 // 16 bits (if protection_absent=0)
	NoRawDataBlocksInFrame uint8  // 2 bits: num blocks - 1

	// Control parameter
	OldFormat bool // Use old ADTS format parsing
}

// HeaderSize returns the ADTS header size in bytes.
// Returns 7 if CRC is absent, 9 if CRC is present.
func (h *ADTSHeader) HeaderSize() int {
	if h.ProtectionAbsent {
		return 7
	}
	return 9
}

// DataSize returns the raw audio data size (frame length minus header).
func (h *ADTSHeader) DataSize() int {
	return int(h.AACFrameLength) - h.HeaderSize()
}
