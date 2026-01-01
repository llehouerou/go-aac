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

	// Lazy-initialize filter bank if not already done
	// This replaces the boolean marker set by initFilterBank() with the actual filter bank
	d.ensureFilterBank()

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

	// Parse raw_data_block
	// Ported from: decoder.c:990
	rdbResult, err := d.parseRawDataBlock(r)
	if err != nil {
		return nil, nil, err
	}

	// Update frame state
	d.frChannels = rdbResult.numChannels
	d.frChEle = rdbResult.numElements

	// Calculate bytes consumed
	// Ported from: decoder.c:1022-1023
	bitsConsumed := r.GetProcessedBits()
	info.BytesConsumed = (bitsConsumed + 7) / 8

	// Validate channel count
	// Ported from: decoder.c:1014-1019
	if rdbResult.numChannels == 0 || rdbResult.numChannels > 64 {
		if rdbResult.numChannels > 64 {
			return nil, nil, ErrInvalidNumChannels
		}
		// Zero channels means empty frame (only ID_END)
		d.frame++
		info.Channels = 0
		return nil, info, nil
	}

	// Allocate channel buffers if needed
	// Ported from: allocate_single_channel() and allocate_channel_pair() in decoder.c
	if err := d.allocateChannelBuffers(rdbResult.numChannels); err != nil {
		return nil, nil, err
	}

	// TODO: Continue with spectral reconstruction
	info.Channels = rdbResult.numChannels
	d.frame++
	return nil, info, nil
}

// ensureFilterBank initializes the filter bank if not already done.
// Uses lazy initialization to avoid import cycles. The filter bank factory
// must be registered by the filterbank package during its init().
//
// This method checks if fb is the boolean marker (true) set by initFilterBank()
// and replaces it with an actual filter bank instance created by the factory.
func (d *Decoder) ensureFilterBank() {
	// Check if already initialized (not the boolean marker and not nil)
	if _, isMarker := d.fb.(bool); !isMarker && d.fb != nil {
		return
	}

	// Use the registered factory to create the filter bank
	if filterBankFactory != nil {
		d.fb = filterBankFactory(d.frameLength)
	}
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

// elementID represents a syntax element identifier.
// Local version to avoid import cycles.
// Source: ~/dev/faad2/libfaad/syntax.h:85-94
type elementID uint8

// Syntax Element IDs.
const (
	idSCE            elementID = 0x0 // Single Channel Element
	idCPE            elementID = 0x1 // Channel Pair Element
	idCCE            elementID = 0x2 // Coupling Channel Element
	idLFE            elementID = 0x3 // LFE Channel Element
	idDSE            elementID = 0x4 // Data Stream Element
	idPCE            elementID = 0x5 // Program Config Element
	idFIL            elementID = 0x6 // Fill Element
	idEND            elementID = 0x7 // Terminating Element
	lenSEID          uint      = 3   // Syntax element identifier length in bits
	invalidElementID elementID = 255
)

// rawDataBlockResult holds the result of parsing a raw data block.
// Local version to avoid import cycles.
// Ported from: raw_data_block() local variables in ~/dev/faad2/libfaad/syntax.c:452-458
type rawDataBlockResult struct {
	numChannels  uint8     // Total channels in this frame (fr_channels)
	numElements  uint8     // Number of elements parsed (fr_ch_ele)
	firstElement elementID // First syntax element type (first_syn_ele)
	hasLFE       bool      // True if LFE element present (has_lfe)
}

// parseRawDataBlock parses a raw_data_block() from the bitstream.
// This is the main entry point for parsing AAC frame data.
// Local version to avoid import cycles with the syntax package.
//
// The function reads syntax elements in a loop until ID_END (0x7) is
// encountered. Currently, only ID_END is handled; other element types
// will be added as the decoder implementation progresses.
//
// Ported from: raw_data_block() in ~/dev/faad2/libfaad/syntax.c:449-648
func (d *Decoder) parseRawDataBlock(r *bits.Reader) (*rawDataBlockResult, error) {
	result := &rawDataBlockResult{
		firstElement: invalidElementID,
	}

	// Main parsing loop
	// Ported from: syntax.c:465-544
	for {
		// Read element ID (3 bits)
		idSynEle := elementID(r.GetBits(lenSEID))

		if idSynEle == idEND {
			break
		}

		// Track elements
		result.numElements++
		if result.firstElement == invalidElementID {
			result.firstElement = idSynEle
		}

		switch idSynEle {
		case idSCE:
			// Single Channel Element
			// Ported from: single_lfe_channel_element() in ~/dev/faad2/libfaad/syntax.c:652-696
			//
			// SCE contains:
			// - element_instance_tag (4 bits)
			// - individual_channel_stream() which includes:
			//   - ics_info()
			//   - section_data()
			//   - scale_factor_data()
			//   - pulse_data() (optional)
			//   - tns_data() (optional)
			//   - gain_control_data() (optional, SSR only)
			//   - spectral_data()
			//
			// After parsing, reconstruct_single_channel() is called.

			// Increment channel count (SCE = 1 channel)
			result.numChannels++

			// Skip element_instance_tag for now (4 bits)
			// TODO: Parse full SCE with individual_channel_stream
			_ = r.GetBits(4) // element_instance_tag

			// For now, return error - full ICS parsing not yet implemented
			// This requires huffman decoding and spectral data parsing
			return nil, ErrMaxBitstreamElements

		case idCPE:
			// TODO: Parse Channel Pair Element
			// For now, return error - not yet implemented
			return nil, ErrMaxBitstreamElements

		case idLFE:
			// TODO: Parse LFE Channel Element
			result.hasLFE = true
			return nil, ErrMaxBitstreamElements

		case idCCE:
			// TODO: Parse Coupling Channel Element
			return nil, ErrChannelCouplingNotImpl

		case idDSE:
			// TODO: Parse Data Stream Element
			return nil, ErrMaxBitstreamElements

		case idPCE:
			// PCE must be first element
			if result.numElements != 1 {
				return nil, ErrPCENotFirst
			}
			// TODO: Parse Program Config Element
			return nil, ErrProgramConfigElement

		case idFIL:
			// TODO: Parse Fill Element
			return nil, ErrMaxBitstreamElements

		default:
			return nil, ErrMaxBitstreamElements
		}
	}

	// Byte align after parsing
	// Ported from: syntax.c:644
	r.ByteAlign()

	return result, nil
}

// sceParseResult holds the parsed data from a Single Channel Element.
// This structure will be populated when full SCE parsing is implemented.
//
// Ported from: single_lfe_channel_element() local variables in ~/dev/faad2/libfaad/syntax.c:652-666
//
//nolint:unused // Infrastructure for future SCE decoding
type sceParseResult struct {
	// ElementInstanceTag is the element instance tag (4 bits)
	ElementInstanceTag uint8

	// Channel is the channel index for this element
	Channel uint8

	// WindowSequence is the window sequence type
	WindowSequence uint8

	// WindowShape is the window shape for this frame
	WindowShape uint8

	// SpecData holds the quantized spectral coefficients (1024 values)
	SpecData []int16
}

// reconstructSCE performs spectral reconstruction for a single channel element.
// This method will be called after SCE parsing is implemented.
//
// The reconstruction pipeline includes:
// 1. Inverse quantization (|x|^(4/3))
// 2. Scale factor application
// 3. PNS decode (noise substitution)
// 4. IC Prediction (MAIN profile)
// 5. LTP prediction (LTP profile)
// 6. TNS decode (temporal noise shaping)
// 7. Filter bank (IMDCT)
//
// Parameters:
//   - sce: Parsed SCE data including spectral coefficients
//   - channel: Channel index for output buffer
//
// Ported from: reconstruct_single_channel() in ~/dev/faad2/libfaad/specrec.c:905-1129
//
//nolint:unused // Infrastructure for future SCE decoding
func (d *Decoder) reconstructSCE(sce *sceParseResult, channel uint8) error {
	// Verify channel is valid
	// Ported from: specrec.c:960-962
	if channel >= maxChannels {
		return ErrInvalidNumChannels
	}

	// Verify buffer is allocated
	// Ported from: specrec.c:961-966 (sanity check for CVE-2018-20199, CVE-2018-20360)
	if d.timeOut[channel] == nil {
		return ErrArrayIndexOutOfRange
	}
	if d.fbIntermed[channel] == nil {
		return ErrArrayIndexOutOfRange
	}

	// TODO: Call spectrum.ReconstructSingleChannel when syntax parsing is complete.
	// The import cycle between aac and spectrum packages needs to be resolved first.
	// For now, just update window shape state for the frame.
	//
	// The full pipeline will be:
	// 1. spectrum.ReconstructSingleChannel(quantData, specData, cfg)
	// 2. filterbank.IFilterBank(..., specData, timeOut, fbIntermed, ...)
	// 3. LTP state update (if LTP profile)

	// Save window shape for next frame
	// Ported from: specrec.c:1055
	d.windowShapePrev[channel] = sce.WindowShape

	return nil
}
