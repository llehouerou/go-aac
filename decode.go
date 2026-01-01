// decode.go
package aac

import (
	"github.com/llehouerou/go-aac/internal/bits"
)

// Decode decodes one AAC frame and returns PCM samples.
//
// Parameters:
//   - buffer: Input AAC frame data
//
// Returns:
//   - samples: Interleaved PCM samples (int16 for 16-bit format)
//   - info: Frame information (channels, sample rate, bytes consumed, etc.)
//   - err: Error if decoding fails
//
// The decoder must be initialized with Init() or Init2() before calling Decode().
// Each call to Decode() processes exactly one frame. For ADTS streams, the ADTS
// header is parsed automatically. For raw AAC, the caller must provide frame
// boundaries.
//
// Note: The first frame returns zero samples due to the overlap-add delay.
// This matches FAAD2 behavior (decoder.c:1204-1206).
//
// Ported from: aac_frame_decode() in ~/dev/faad2/libfaad/decoder.c:848-1255
func (d *Decoder) Decode(buffer []byte) (interface{}, *FrameInfo, error) {
	// Safety checks
	// Ported from: decoder.c:872-876
	if d == nil {
		return nil, nil, ErrNilDecoder
	}
	if buffer == nil {
		return nil, nil, ErrNilBuffer
	}
	if len(buffer) == 0 {
		return nil, nil, ErrBufferTooSmall
	}

	// Initialize FrameInfo
	info := &FrameInfo{}

	// Check for ID3v1 tag (128 bytes starting with "TAG")
	// Ported from: decoder.c:901-910
	if len(buffer) >= 128 && buffer[0] == 'T' && buffer[1] == 'A' && buffer[2] == 'G' {
		info.BytesConsumed = 128
		// No error, but no output either
		return nil, info, nil
	}

	// Initialize bitstream reader
	// Ported from: decoder.c:914-917
	r := bits.NewReader(buffer)

	// Parse ADTS header if present
	// Ported from: decoder.c:965-977
	// Note: We use parseADTSFrameHeader (local version) to avoid import cycle with syntax package.
	if d.adtsHeaderPresent {
		_, err := parseADTSFrameHeader(r, d.config.UseOldADTSFormat)
		if err != nil {
			return nil, nil, err
		}
		info.HeaderType = HeaderTypeADTS
	} else if d.adifHeaderPresent {
		info.HeaderType = HeaderTypeADIF
	} else {
		info.HeaderType = HeaderTypeRAW
	}

	// TODO: Continue with raw_data_block parsing
	return nil, info, nil
}

// adtsFrameHeader contains the full ADTS frame header for Decode().
// This extends adtsHeader (used by Init) with variable header fields.
//
// Ported from: adts_header in ~/dev/faad2/libfaad/structs.h:146-168
type adtsFrameHeader struct {
	// Fixed header
	Profile              uint8 // 2 bits: object type - 1
	SFIndex              uint8 // 4 bits: sample frequency index
	ChannelConfiguration uint8 // 3 bits: channel config
	// Variable header
	FrameLength    uint16 // 13 bits: total frame bytes including header
	BufferFullness uint16 // 11 bits: buffer fullness
	NumBlocks      uint8  // 2 bits: number of raw_data_block - 1
	CRCPresent     bool   // true if CRC is present
}

// parseADTSFrameHeader parses a complete ADTS frame header.
// This parses both fixed and variable headers for use in Decode().
// Local version to avoid import cycles with the syntax package.
//
// Ported from: adts_frame() in ~/dev/faad2/libfaad/syntax.c:2449-2538
func parseADTSFrameHeader(r *bits.Reader, oldFormat bool) (*adtsFrameHeader, error) {
	// Search for syncword (0xFFF)
	const maxSyncSearch = 768
	for i := 0; i < maxSyncSearch; i++ {
		syncword := r.ShowBits(12)
		if syncword == 0x0FFF {
			r.FlushBits(12)

			// Parse fixed header (16 bits after syncword)
			// Ported from: adts_fixed_header() in ~/dev/faad2/libfaad/syntax.c:2484-2511
			id := r.Get1Bit()
			r.FlushBits(2) // layer (always 0)
			protectionAbsent := r.Get1Bit() == 1
			profile := uint8(r.GetBits(2))
			sfIndex := uint8(r.GetBits(4))
			r.FlushBits(1) // private_bit
			chanConfig := uint8(r.GetBits(3))
			r.FlushBits(1) // original
			r.FlushBits(1) // home

			// Old ADTS format (removed in corrigendum 14496-3:2002)
			if oldFormat && id == 0 {
				r.FlushBits(2) // emphasis
			}

			// Parse variable header (28 bits)
			// Ported from: adts_variable_header() in ~/dev/faad2/libfaad/syntax.c:2517-2528
			r.FlushBits(1) // copyright_id_bit
			r.FlushBits(1) // copyright_id_start
			frameLength := uint16(r.GetBits(13))
			bufferFullness := uint16(r.GetBits(11))
			numBlocks := uint8(r.GetBits(2))

			// Parse error check (CRC if present)
			// Ported from: adts_error_check() in ~/dev/faad2/libfaad/syntax.c:2532-2538
			if !protectionAbsent {
				r.FlushBits(16) // crc_check
			}

			return &adtsFrameHeader{
				Profile:              profile,
				SFIndex:              sfIndex,
				ChannelConfiguration: chanConfig,
				FrameLength:          frameLength,
				BufferFullness:       bufferFullness,
				NumBlocks:            numBlocks,
				CRCPresent:           !protectionAbsent,
			}, nil
		}
		r.FlushBits(8)
	}
	return nil, ErrADTSSyncwordNotFound
}
