// decode.go
package aac

import (
	"fmt"

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

	// Determine output channels (downmix if configured)
	// Ported from: decoder.c:1056-1061
	outputChannels := rdbResult.numChannels
	if (outputChannels == 5 || outputChannels == 6) && d.config.DownMatrix {
		d.downMatrix = true
		outputChannels = 2
	}

	// Create channel configuration
	d.createChannelConfig(info)

	// Populate FrameInfo
	// Ported from: decoder.c:1075-1083
	info.Samples = uint32(d.frameLength) * uint32(outputChannels)
	info.Channels = outputChannels
	info.SampleRate = getSampleRate(d.sfIndex)
	info.ObjectType = ObjectType(d.objectType)
	info.SBR = SBRNone

	// TODO: Process each element (SCE, CPE, LFE) when parsing is implemented
	// For each SCE: d.reconstructSCE() -> d.applyFilterBank()
	// For each CPE: d.reconstructCPE() -> d.applyFilterBank() (x2)
	// For each LFE: d.reconstructSCE() -> d.applyFilterBank()

	// Generate PCM output
	samples := d.generatePCMOutput(outputChannels)

	// Post-decode processing
	d.postSeekResetFlag = false
	d.frame++

	// Mute first frame (overlap-add delay)
	// Ported from: decoder.c:1204-1206
	if d.frame <= 1 {
		info.Samples = 0
	}

	return samples, info, nil
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
			// Channel Pair Element (stereo)
			// Ported from: channel_pair_element() in ~/dev/faad2/libfaad/syntax.c:698-796
			//
			// CPE contains:
			// - element_instance_tag (4 bits)
			// - common_window flag (1 bit)
			// - if common_window: ics_info() and ms_mask
			// - individual_channel_stream() for each channel
			// - spectral_data() for each channel
			//
			// After parsing, reconstruct_channel_pair() is called.

			// Increment channel count (CPE = 2 channels)
			result.numChannels += 2

			// Skip element_instance_tag for now (4 bits)
			// TODO: Parse full CPE with individual_channel_stream for both channels
			_ = r.GetBits(4) // element_instance_tag

			// For now, return error - full ICS parsing not yet implemented
			return nil, fmt.Errorf("CPE parsing not yet implemented")

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

// cpeParseResult holds parsed data from a Channel Pair Element.
// Ported from: channel_pair_element() in ~/dev/faad2/libfaad/syntax.c:698-796
//
//nolint:unused // Infrastructure for future CPE decoding
type cpeParseResult struct {
	ElementInstanceTag uint8   // element_instance_tag (4 bits)
	CommonWindow       bool    // common_window flag
	Channel1           uint8   // first channel index
	Channel2           uint8   // second channel index
	WindowSequence1    uint8   // window sequence for channel 1
	WindowShape1       uint8   // window shape for channel 1
	WindowSequence2    uint8   // window sequence for channel 2
	WindowShape2       uint8   // window shape for channel 2
	MSMaskPresent      uint8   // ms_mask_present (0=off, 1=some, 2=all)
	SpecData1          []int16 // quantized spectral coefficients channel 1
	SpecData2          []int16 // quantized spectral coefficients channel 2
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

// reconstructCPE performs spectral reconstruction for a channel pair element.
// This method will be called after CPE parsing is implemented.
//
// The reconstruction pipeline is:
// 1. Dequantize spectral coefficients (apply_scalefactors + quant_to_spec)
// 2. Apply M/S stereo decoding if enabled
// 3. Apply Intensity Stereo if enabled
// 4. Apply PNS (Perceptual Noise Substitution) if enabled
// 5. Apply TNS (Temporal Noise Shaping) if enabled
// 6. Apply filterbank (IMDCT + windowing + overlap-add)
//
// Ported from: reconstruct_channel_pair() in ~/dev/faad2/libfaad/specrec.c:1131-1323
//
//nolint:unused // Infrastructure for future CPE decoding
func (d *Decoder) reconstructCPE(cpe *cpeParseResult, channelBase uint8) error {
	// Verify both channels are valid
	if channelBase+1 >= maxChannels {
		return ErrInvalidNumChannels
	}

	// Verify buffers are allocated for both channels
	// Security: Matches FAAD2 checks for CVE-2018-20199, CVE-2018-20360
	for ch := uint8(0); ch < 2; ch++ {
		idx := channelBase + ch
		if d.timeOut[idx] == nil {
			return ErrArrayIndexOutOfRange
		}
		if d.fbIntermed[idx] == nil {
			return ErrArrayIndexOutOfRange
		}
	}

	// TODO: When spectrum package is connected (via factory pattern like filterbank):
	// 1. spectrum.ReconstructChannelPair(specData1, specData2, ...)
	// 2. filterbank.IFilterBank() for each channel
	// 3. Update LTP state if Main profile

	// Update window shapes for next frame
	// Ported from: specrec.c:1312-1313
	d.windowShapePrev[channelBase] = cpe.WindowShape1
	d.windowShapePrev[channelBase+1] = cpe.WindowShape2

	return nil
}

// generatePCMOutput converts time-domain samples to PCM format.
//
// Parameters:
//   - outputChannels: Number of channels to output
//
// Returns the PCM samples in the format specified by d.config.OutputFormat.
// The returned type depends on the format:
//   - OutputFormat16Bit: []int16
//   - OutputFormat24Bit: []int32 (packed 24-bit in 32-bit container)
//   - OutputFormat32Bit: []int32
//   - OutputFormatFloat: []float32
//   - OutputFormatDouble: []float64
//
// Ported from: output_to_PCM() call in ~/dev/faad2/libfaad/decoder.c:1188-1189
//
//nolint:unused // Infrastructure for future decoding
func (d *Decoder) generatePCMOutput(outputChannels uint8) interface{} {
	// For now, just convert float32 to int16 directly
	// This will be replaced with proper output package integration

	samples := make([]int16, int(d.frameLength)*int(outputChannels))

	for ch := uint8(0); ch < outputChannels; ch++ {
		if d.timeOut[ch] == nil {
			continue
		}
		for i := 0; i < int(d.frameLength); i++ {
			sample := d.timeOut[ch][i]
			// Clip and convert to int16
			if sample > 32767.0 {
				sample = 32767.0
			} else if sample < -32768.0 {
				sample = -32768.0
			}
			// Interleave: sample[i*numCh + ch]
			samples[i*int(outputChannels)+int(ch)] = int16(sample)
		}
	}

	return samples
}

// createChannelConfig creates the channel position mapping.
//
// Standard AAC channel configurations:
//
//	1: C (mono)
//	2: L, R (stereo)
//	3: C, L, R
//	4: C, L, R, Cs (rear center)
//	5: C, L, R, Ls, Rs
//	6: C, L, R, Ls, Rs, LFE (5.1)
//	7: C, L, R, Ls, Rs, Lrs, Rrs, LFE (7.1)
//
// Ported from: create_channel_config() in ~/dev/faad2/libfaad/decoder.c:598-819
//
//nolint:unused
func (d *Decoder) createChannelConfig(info *FrameInfo) {
	info.NumFrontChannels = 0
	info.NumSideChannels = 0
	info.NumBackChannels = 0
	info.NumLFEChannels = 0

	// Clear channel positions
	for i := range info.ChannelPosition {
		info.ChannelPosition[i] = ChannelUnknown
	}

	// Handle downmix to stereo
	if d.downMatrix {
		info.NumFrontChannels = 2
		info.ChannelPosition[0] = ChannelFrontLeft
		info.ChannelPosition[1] = ChannelFrontRight
		return
	}

	// TODO: Handle PCE-based channel config when pceSet is true
	// For now, only standard channel configurations are supported

	// Standard channel configurations
	switch d.channelConfiguration {
	case 1: // mono
		info.NumFrontChannels = 1
		info.ChannelPosition[0] = ChannelFrontCenter
	case 2: // stereo
		info.NumFrontChannels = 2
		info.ChannelPosition[0] = ChannelFrontLeft
		info.ChannelPosition[1] = ChannelFrontRight
	case 3: // 3.0
		info.NumFrontChannels = 3
		info.ChannelPosition[0] = ChannelFrontCenter
		info.ChannelPosition[1] = ChannelFrontLeft
		info.ChannelPosition[2] = ChannelFrontRight
	case 4: // 4.0
		info.NumFrontChannels = 3
		info.NumBackChannels = 1
		info.ChannelPosition[0] = ChannelFrontCenter
		info.ChannelPosition[1] = ChannelFrontLeft
		info.ChannelPosition[2] = ChannelFrontRight
		info.ChannelPosition[3] = ChannelBackCenter
	case 5: // 5.0
		info.NumFrontChannels = 3
		info.NumBackChannels = 2
		info.ChannelPosition[0] = ChannelFrontCenter
		info.ChannelPosition[1] = ChannelFrontLeft
		info.ChannelPosition[2] = ChannelFrontRight
		info.ChannelPosition[3] = ChannelBackLeft
		info.ChannelPosition[4] = ChannelBackRight
	case 6: // 5.1
		info.NumFrontChannels = 3
		info.NumBackChannels = 2
		info.NumLFEChannels = 1
		info.ChannelPosition[0] = ChannelFrontCenter
		info.ChannelPosition[1] = ChannelFrontLeft
		info.ChannelPosition[2] = ChannelFrontRight
		info.ChannelPosition[3] = ChannelBackLeft
		info.ChannelPosition[4] = ChannelBackRight
		info.ChannelPosition[5] = ChannelLFE
	case 7: // 7.1
		info.NumFrontChannels = 3
		info.NumSideChannels = 2
		info.NumBackChannels = 2
		info.NumLFEChannels = 1
		info.ChannelPosition[0] = ChannelFrontCenter
		info.ChannelPosition[1] = ChannelFrontLeft
		info.ChannelPosition[2] = ChannelFrontRight
		info.ChannelPosition[3] = ChannelSideLeft
		info.ChannelPosition[4] = ChannelSideRight
		info.ChannelPosition[5] = ChannelBackLeft
		info.ChannelPosition[6] = ChannelBackRight
		info.ChannelPosition[7] = ChannelLFE
	default:
		// Configuration 0 or >7: channels defined by elements in bitstream
		// TODO: Implement fallback channel config based on fr_channels and has_lfe
		// For now, leave channel positions as ChannelUnknown
	}
}

// DecodeInt16 decodes one AAC frame and returns int16 PCM samples.
// This is a convenience wrapper that returns only samples and error,
// matching the simplified API from MIGRATION_STEPS.md Step 7.4.
//
// For detailed frame information (channels, sample rate, bytes consumed),
// use Decode() which returns *FrameInfo.
//
// The first frame returns nil samples due to the overlap-add delay.
// This matches FAAD2 behavior.
func (d *Decoder) DecodeInt16(frame []byte) ([]int16, error) {
	samples, info, err := d.Decode(frame)
	if err != nil {
		return nil, err
	}

	// No samples (first frame or empty)
	if info == nil || info.Samples == 0 {
		return nil, nil
	}

	// Type assert to []int16
	int16Samples, ok := samples.([]int16)
	if !ok {
		// For non-16bit output formats, return nil
		// Users should use Decode() directly for other formats
		return nil, nil
	}

	return int16Samples, nil
}

// DecodeFloat decodes one AAC frame and returns float32 PCM samples.
// This is a convenience wrapper around Decode() with float output format.
//
// Ported from: NeAACDecDecode() with FAAD_FMT_FLOAT
func (d *Decoder) DecodeFloat(buffer []byte) ([]float32, *FrameInfo, error) {
	// Temporarily set output format to float
	originalFormat := d.config.OutputFormat
	d.config.OutputFormat = OutputFormatFloat

	samples, info, err := d.Decode(buffer)

	// Restore original format
	d.config.OutputFormat = originalFormat

	if err != nil || samples == nil {
		return nil, info, err
	}

	floatSamples, ok := samples.([]float32)
	if !ok {
		// If Decode returned different type, return nil
		return nil, info, nil
	}

	return floatSamples, info, nil
}

// applyFilterBank applies the inverse filter bank (IMDCT + windowing + overlap-add).
//
// Parameters:
//   - specData: Spectral coefficients (float32)
//   - channel: Channel index for accessing overlap buffers
//   - windowSequence: Window type (0=long, 1=long_start, 2=8-short, 3=long_stop)
//   - windowShape: Window shape for current frame
//
// The filter bank performs:
// 1. IMDCT (Modified Discrete Cosine Transform inverse)
// 2. Windowing with overlap-add
// 3. Writes output to d.timeOut[channel]
//
// Ported from: ifilter_bank() in ~/dev/faad2/libfaad/filtbank.c
//
//nolint:unused // Infrastructure for future decoding
func (d *Decoder) applyFilterBank(
	specData []float32,
	channel uint8,
	windowSequence uint8,
	windowShape uint8,
) error {
	// Ensure filter bank is initialized
	if d.fb == nil {
		return ErrNilDecoder // Filter bank not initialized
	}

	// Type-assert to access IFilterBank method
	// The actual type is *filterbank.FilterBank registered via factory
	// WindowSequence is defined as uint8 in internal/syntax/constants.go
	type filterBankInterface interface {
		IFilterBank(
			windowSequence uint8,
			windowShape uint8,
			windowShapePrev uint8,
			freqIn []float32,
			timeOut []float32,
			overlap []float32,
		)
	}

	fb, ok := d.fb.(filterBankInterface)
	if !ok {
		return ErrNilDecoder // Filter bank not properly initialized
	}

	// Apply inverse filter bank
	fb.IFilterBank(
		windowSequence,
		windowShape,
		d.windowShapePrev[channel],
		specData,
		d.timeOut[channel],
		d.fbIntermed[channel],
	)

	return nil
}
