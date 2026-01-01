// decoder.go
package aac

import (
	"github.com/llehouerou/go-aac/internal/bits"
)

// FilterBankFactory is a function that creates a filter bank for the given frame length.
// This is used to break the import cycle between aac and filterbank packages.
// The factory is registered by the filterbank package during import.
type FilterBankFactory func(frameLength uint16) any

// filterBankFactory is the registered factory function for creating filter banks.
// It's set by RegisterFilterBankFactory, typically called from filterbank package init.
var filterBankFactory FilterBankFactory

// RegisterFilterBankFactory registers the factory function for creating filter banks.
// This is called by the filterbank package during its initialization to break
// the import cycle between aac and filterbank.
//
// Usage: The filterbank package should call this in its init() function:
//
//	func init() {
//	    aac.RegisterFilterBankFactory(func(frameLength uint16) any {
//	        return NewFilterBank(frameLength)
//	    })
//	}
func RegisterFilterBankFactory(factory FilterBankFactory) {
	filterBankFactory = factory
}

// Maximum limits for decoder state arrays.
// These match the constants in internal/syntax/limits.go but are duplicated
// here to avoid import cycles (syntax imports aac for AudioSpecificConfig).
//
//nolint:unused // Used in Decoder struct field sizing
const (
	maxChannels       = 64 // Maximum number of channels
	maxSyntaxElements = 48 // Maximum number of syntax elements
)

// Decoder is the main AAC decoder.
// It maintains all state needed for decoding an AAC stream.
//
// Ported from: NeAACDecStruct in ~/dev/faad2/libfaad/structs.h:332-439
//
//nolint:unused // Fields are used incrementally as decoder features are implemented
type Decoder struct {
	// Configuration
	config Config

	// Stream format detection
	adtsHeaderPresent bool
	adifHeaderPresent bool
	latmHeaderPresent bool

	// Stream parameters
	sfIndex              uint8  // Sample frequency index
	objectType           uint8  // Audio object type
	channelConfiguration uint8  // Channel configuration
	frameLength          uint16 // Frame length (typically 1024)

	// Frame state
	frame             uint32 // Current frame number
	postSeekResetFlag bool   // Reset state after seek

	// Output configuration
	sampleBufferSize uint32 // Output buffer size
	downMatrix       bool   // Enable 5.1 to stereo downmix
	upMatrix         bool   // Enable mono to stereo upmix
	firstSynEle      bool   // First syntax element of frame
	hasLFE           bool   // Stream has LFE channel

	// Per-frame element info
	frChannels uint8 // Channels in current frame
	frChEle    uint8 // Elements in current frame

	// Element tracking
	elementOutputChannels [maxSyntaxElements]uint8 // Output channels per element
	elementAlloced        [maxSyntaxElements]bool  // Element buffers allocated

	// Processing components
	// Note: FilterBank and DRC are typed as 'any' to avoid import cycles.
	// The filterbank and output packages import syntax, which imports aac.
	// These are initialized lazily during first decode.
	fb  any // Filter bank for IMDCT (*filterbank.FilterBank)
	drc any // Dynamic range control (*output.DRC)

	// Per-channel state
	windowShapePrev [maxChannels]uint8     // Previous window shape
	ltpLag          [maxChannels]uint16    // LTP lag values
	timeOut         [maxChannels][]float32 // Time-domain output buffers
	fbIntermed      [maxChannels][]float32 // Filter bank intermediate buffers

	// LTP prediction state (for LTP profile)
	ltPredStat [maxChannels][]int16

	// RNG state (for PNS)
	rngState1 uint32
	rngState2 uint32

	// Program config
	pceSet          bool               // PCE has been parsed
	pce             any                // Program config element (*syntax.ProgramConfig)
	elementID       [maxChannels]uint8 // Element ID per channel
	internalChannel [maxChannels]uint8 // Internal channel mapping
}

// NewDecoder creates a new AAC decoder with default settings.
//
// Ported from: NeAACDecOpen() in ~/dev/faad2/libfaad/decoder.c:123-182
func NewDecoder() *Decoder {
	d := &Decoder{
		config: Config{
			DefObjectType: ObjectTypeMain, // FAAD2 default is MAIN (decoder.c:135)
			DefSampleRate: 44100,
			OutputFormat:  OutputFormat16Bit,
		},
		frameLength: 1024,
		// RNG seeds from FAAD2 decoder.c:151-153
		// "Same as (1, 1) after 1024 iterations; otherwise first values does not look random at all"
		rngState1: 0x2bb431ea,
		rngState2: 0x206155b7,
	}

	// TODO(Step 7.2): Initialize FilterBank and DRC here once import cycle is resolved.
	// Currently deferred to Init() method due to: aac -> filterbank -> syntax -> aac cycle.
	// The syntax/asc.go file imports the root aac package for AudioSpecificConfig.

	return d
}

// Config returns the current decoder configuration.
func (d *Decoder) Config() Config {
	return d.config
}

// SetConfiguration sets the decoder configuration.
// Should be called before Init() for the settings to take effect.
//
// Ported from: NeAACDecSetConfiguration() in ~/dev/faad2/libfaad/decoder.c:264-299
func (d *Decoder) SetConfiguration(cfg Config) {
	d.config = cfg
	d.downMatrix = cfg.DownMatrix
}

// allocateChannelBuffers allocates per-channel buffers for the specified number of channels.
// Buffers are only allocated once; subsequent calls with the same or fewer channels are no-ops.
//
// Ported from: allocate_single_channel() and allocate_channel_pair() in ~/dev/faad2/libfaad/specrec.c:700-850
func (d *Decoder) allocateChannelBuffers(numChannels uint8) error {
	if numChannels > maxChannels {
		return ErrInvalidNumChannels
	}

	frameLen := int(d.frameLength)

	for ch := uint8(0); ch < numChannels; ch++ {
		// Allocate timeOut buffer if not already allocated
		if d.timeOut[ch] == nil {
			d.timeOut[ch] = make([]float32, frameLen)
		}

		// Allocate fbIntermed buffer if not already allocated
		if d.fbIntermed[ch] == nil {
			d.fbIntermed[ch] = make([]float32, frameLen)
		}
	}

	return nil
}

// allocateLTPBuffers allocates Long Term Prediction buffers for the specified channels.
// LTP requires storing 2*frameLength samples for prediction.
//
// Ported from: decoder.c handling of LTP allocation
func (d *Decoder) allocateLTPBuffers(numChannels uint8) {
	bufLen := 2 * int(d.frameLength)

	for ch := uint8(0); ch < numChannels; ch++ {
		if d.ltPredStat[ch] == nil {
			d.ltPredStat[ch] = make([]int16, bufLen)
		}
	}
}

// Sample rate lookup table.
// Indices 0-11 ported from: sample_rates[] in ~/dev/faad2/libfaad/common.c:61-65
// Index 12 (7350 Hz) is defined in ISO/IEC 14496-3.
// Indices 13-15 are reserved (return 0).
var sampleRates = [16]uint32{
	96000, 88200, 64000, 48000, 44100, 32000,
	24000, 22050, 16000, 12000, 11025, 8000,
	7350, 0, 0, 0,
}

// SampleRate returns the current sample rate in Hz.
func (d *Decoder) SampleRate() uint32 {
	if d.sfIndex < 16 {
		return sampleRates[d.sfIndex]
	}
	return 0
}

// Channels returns the channel configuration value.
// This corresponds to the channel_configuration field in the audio specific config.
// Note: This returns the raw configuration (0-7), not the actual channel count.
// Configuration 0 means channels are defined elsewhere; 1=mono, 2=stereo, etc.
func (d *Decoder) Channels() uint8 {
	return d.channelConfiguration
}

// FrameLength returns the number of samples per frame per channel.
func (d *Decoder) FrameLength() uint16 {
	return d.frameLength
}

// ObjectType returns the AAC object type.
func (d *Decoder) ObjectType() ObjectType {
	return ObjectType(d.objectType)
}

// PostSeekReset resets decoder state after a seek operation.
// If frame >= 0, sets the frame counter to that value.
// If frame == -1, the frame counter is left unchanged.
//
// Ported from: NeAACDecPostSeekReset() in ~/dev/faad2/libfaad/decoder.c:586-596
func (d *Decoder) PostSeekReset(frame int64) {
	d.postSeekResetFlag = true

	if frame != -1 {
		d.frame = uint32(frame)
	}
}

// InitResult contains the result of decoder initialization.
// Returned by Init() and Init2() methods.
//
// Ported from: return values of NeAACDecInit/NeAACDecInit2 in decoder.c
type InitResult struct {
	SampleRate uint32 // Output sample rate in Hz
	Channels   uint8  // Number of output channels
	BytesRead  uint32 // Bytes consumed during init (ADIF only, 0 for ADTS/raw)
}

// Close releases decoder resources.
// The decoder should not be used after calling Close.
//
// Ported from: NeAACDecClose() in ~/dev/faad2/libfaad/decoder.c:532-582
func (d *Decoder) Close() {
	// Clear per-channel buffers to help GC
	for ch := 0; ch < maxChannels; ch++ {
		d.timeOut[ch] = nil
		d.fbIntermed[ch] = nil
		d.ltPredStat[ch] = nil
	}

	// Clear component references
	d.fb = nil
	d.drc = nil
	d.pce = nil
}

// Init initializes the decoder with the given AAC bitstream data.
// It detects the stream format (ADTS, ADIF, or raw) and extracts stream parameters.
//
// For ADTS streams, the header is detected but not consumed (BytesRead=0).
// For ADIF streams, the header is consumed and BytesRead reflects bytes read.
// For raw AAC, default parameters from Config are used.
//
// Returns stream parameters in InitResult, or an error if initialization fails.
//
// Ported from: NeAACDecInit() in ~/dev/faad2/libfaad/decoder.c:303-426
func (d *Decoder) Init(data []byte) (InitResult, error) {
	if d == nil {
		return InitResult{}, ErrNilDecoder
	}
	if data == nil {
		return InitResult{}, ErrNilBuffer
	}
	if len(data) < 2 {
		return InitResult{}, ErrBufferTooSmall
	}

	// Set defaults from config
	d.sfIndex = getSRIndex(d.config.DefSampleRate)
	d.objectType = uint8(d.config.DefObjectType)

	result := InitResult{
		SampleRate: getSampleRate(d.sfIndex),
		Channels:   1,
	}

	// Check for ADIF header ("ADIF" magic) first
	// ADIF is rare but must be detected before trying ADTS syncword search
	// Ported from: NeAACDecInit() ADIF check in ~/dev/faad2/libfaad/decoder.c:307-338
	if len(data) >= 4 && data[0] == 'A' && data[1] == 'D' && data[2] == 'I' && data[3] == 'F' {
		return d.initFromADIF(data)
	}

	r := bits.NewReader(data)

	// Try ADTS parsing (most common format)
	adts, err := parseADTSHeader(r, d.config.UseOldADTSFormat)
	if err == nil {
		return d.initFromADTS(adts, &result)
	}

	// Fallback to defaults (raw AAC or unrecognized format)
	if err := d.initFilterBank(); err != nil {
		return InitResult{}, err
	}
	return result, nil
}

// adtsHeader contains the minimal ADTS header fields needed for Init().
// This is a local type to avoid importing the syntax package.
//
// Ported from: adts_header in ~/dev/faad2/libfaad/structs.h:146-168
type adtsHeader struct {
	Profile              uint8 // 2 bits: object type - 1
	SFIndex              uint8 // 4 bits: sample frequency index
	ChannelConfiguration uint8 // 3 bits: channel config
}

// parseADTSHeader parses an ADTS header from the bitstream.
// This is a local version to avoid import cycles with the syntax package.
//
// Ported from: adts_frame() in ~/dev/faad2/libfaad/syntax.c:2449-2458
func parseADTSHeader(r *bits.Reader, oldFormat bool) (*adtsHeader, error) {
	// Search for syncword (0xFFF)
	const maxSyncSearch = 768
	for i := 0; i < maxSyncSearch; i++ {
		syncword := r.ShowBits(12)
		if syncword == 0x0FFF {
			r.FlushBits(12)
			// Parse fixed header
			id := r.Get1Bit()
			r.FlushBits(2) // layer (always 0)
			r.FlushBits(1) // protection_absent
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

			return &adtsHeader{
				Profile:              profile,
				SFIndex:              sfIndex,
				ChannelConfiguration: chanConfig,
			}, nil
		}
		r.FlushBits(8)
	}
	return nil, ErrADTSSyncwordNotFound
}

// initFromADIF initializes the decoder from an ADIF header.
// ADIF (Audio Data Interchange Format) is a container format that stores
// a single audio program with a header at the beginning of the file.
// It's much rarer than ADTS but must be detected and handled.
//
// For now, we detect ADIF and return ErrADIFNotSupported.
// Full ADIF support requires parsing the Program Config Element (PCE).
//
// Ported from: NeAACDecInit() ADIF handling in ~/dev/faad2/libfaad/decoder.c:307-338
func (d *Decoder) initFromADIF(_ []byte) (InitResult, error) {
	d.adifHeaderPresent = true
	// ADIF format is rare; return unsupported for now
	// Full implementation would parse PCE from the ADIF header
	return InitResult{}, ErrADIFNotSupported
}

// initFromADTS initializes the decoder from a parsed ADTS header.
//
// Ported from: NeAACDecInit() ADTS handling in ~/dev/faad2/libfaad/decoder.c:340-380
func (d *Decoder) initFromADTS(adts *adtsHeader, result *InitResult) (InitResult, error) {
	d.adtsHeaderPresent = true
	d.sfIndex = adts.SFIndex
	d.objectType = adts.Profile + 1 // ADTS profile is object_type - 1
	d.channelConfiguration = adts.ChannelConfiguration

	result.SampleRate = getSampleRate(d.sfIndex)
	if adts.ChannelConfiguration > 6 {
		// Channel configs > 6 are complex; default to stereo
		result.Channels = 2
	} else {
		result.Channels = adts.ChannelConfiguration
	}

	if result.SampleRate == 0 {
		return InitResult{}, ErrInvalidSampleRate
	}
	if !canDecodeOT(ObjectType(d.objectType)) {
		return InitResult{}, ErrUnsupportedObjectType
	}

	// Update channel configuration in decoder state
	d.channelConfiguration = result.Channels

	if err := d.initFilterBank(); err != nil {
		return InitResult{}, err
	}
	return *result, nil
}

// initFilterBank initializes the filter bank for the current frame length.
// The filter bank is stored as 'any' to avoid import cycles.
// If the factory is registered, it creates the filter bank immediately.
// Otherwise, it sets a marker for lazy initialization during decode.
func (d *Decoder) initFilterBank() error {
	// If factory is registered, use it to create the filter bank immediately
	if filterBankFactory != nil {
		d.fb = filterBankFactory(d.frameLength)
		return nil
	}
	// Otherwise, set a marker value to indicate initialization was requested.
	// The filter bank will be created lazily during first decode.
	d.fb = true // Marker: filter bank init requested
	return nil
}

// getSampleRate returns the sample rate for a given index.
// Returns 0 for invalid indices (>= 16).
// Local version to avoid import cycle with tables package.
//
// Source: ~/dev/faad2/libfaad/common.c:59-71 (get_sample_rate function)
func getSampleRate(srIndex uint8) uint32 {
	if srIndex >= 16 {
		return 0
	}
	return sampleRates[srIndex]
}

// getSRIndex returns the sample rate index for a given sample rate.
// Uses threshold-based matching as defined in the MPEG-4 AAC standard.
// Local version to avoid import cycle with tables package.
//
// Source: ~/dev/faad2/libfaad/common.c:41-56 (get_sr_index function)
func getSRIndex(sampleRate uint32) uint8 {
	if sampleRate >= 92017 {
		return 0
	}
	if sampleRate >= 75132 {
		return 1
	}
	if sampleRate >= 55426 {
		return 2
	}
	if sampleRate >= 46009 {
		return 3
	}
	if sampleRate >= 37566 {
		return 4
	}
	if sampleRate >= 27713 {
		return 5
	}
	if sampleRate >= 23004 {
		return 6
	}
	if sampleRate >= 18783 {
		return 7
	}
	if sampleRate >= 13856 {
		return 8
	}
	if sampleRate >= 11502 {
		return 9
	}
	if sampleRate >= 9391 {
		return 10
	}
	return 11
}

// canDecodeOT returns true if the object type can be decoded.
// Local version to avoid import cycle with tables package.
//
// Source: ~/dev/faad2/libfaad/common.c:124-172
func canDecodeOT(objectType ObjectType) bool {
	switch objectType {
	case ObjectTypeLC:
		return true
	case ObjectTypeMain:
		return true
	case ObjectTypeLTP:
		return true
	case ObjectTypeSSR:
		return false // SSR not supported
	case ObjectTypeERLC:
		return true
	case ObjectTypeERLTP:
		return true
	case ObjectTypeLD:
		return true
	case ObjectTypeDRMERLC:
		return true
	default:
		return false
	}
}

// Init2 initializes the decoder from an MP4 AudioSpecificConfig.
// This is used when decoding AAC from MP4/M4A containers.
//
// The ASC contains the object type, sample rate, and channel configuration
// in a compact format. This method parses it and configures the decoder.
//
// Ported from: NeAACDecInit2() in ~/dev/faad2/libfaad/decoder.c:395-486
func (d *Decoder) Init2(asc []byte) (InitResult, error) {
	if d == nil {
		return InitResult{}, ErrNilDecoder
	}
	if asc == nil {
		return InitResult{}, ErrNilBuffer
	}
	if len(asc) < 2 {
		return InitResult{}, ErrBufferTooSmall
	}

	// Clear header present flags (not ADTS or ADIF)
	d.adtsHeaderPresent = false
	d.adifHeaderPresent = false

	// Parse the AudioSpecificConfig
	r := bits.NewReader(asc)

	mp4ASC, err := parseAudioSpecificConfig(r, uint32(len(asc)))
	if err != nil {
		return InitResult{}, err
	}

	// Validate object type
	if !canDecodeOT(ObjectType(mp4ASC.objectType)) {
		return InitResult{}, ErrUnsupportedObjectType
	}

	// Validate sample rate
	if mp4ASC.sampleRate == 0 {
		return InitResult{}, ErrInvalidSampleRate
	}

	// Copy to decoder state
	d.sfIndex = mp4ASC.sfIndex
	d.objectType = mp4ASC.objectType
	d.channelConfiguration = mp4ASC.channelConfig

	// Build result
	result := InitResult{
		SampleRate: mp4ASC.sampleRate,
		Channels:   mp4ASC.channelConfig,
		BytesRead:  0, // ASC is typically copied, not consumed
	}

	// Initialize filter bank
	if err := d.initFilterBank(); err != nil {
		return InitResult{}, err
	}

	return result, nil
}

// SimpleInit initializes the decoder and returns sample rate and channels directly.
// This is a convenience wrapper around Init() that matches the simplified API
// specified in MIGRATION_STEPS.md Step 7.4.
//
// For more detailed initialization info (e.g., bytes consumed for ADIF),
// use Init() which returns an InitResult struct.
func (d *Decoder) SimpleInit(data []byte) (sampleRate uint32, channels uint8, err error) {
	result, err := d.Init(data)
	if err != nil {
		return 0, 0, err
	}
	return result.SampleRate, result.Channels, nil
}

// SimpleInit2 initializes the decoder from an AudioSpecificConfig.
// This is a convenience wrapper around Init2() that matches the simplified API.
//
// For more detailed initialization info, use Init2() which returns an InitResult.
func (d *Decoder) SimpleInit2(asc []byte) (sampleRate uint32, channels uint8, err error) {
	result, err := d.Init2(asc)
	if err != nil {
		return 0, 0, err
	}
	return result.SampleRate, result.Channels, nil
}

// mp4AudioSpecificConfig holds parsed ASC data.
// Local type to avoid import cycles.
//
// Ported from: mp4AudioSpecificConfig in ~/dev/faad2/libfaad/mp4.h:36-76
type mp4AudioSpecificConfig struct {
	objectType    uint8  // Audio object type (1=Main, 2=LC, etc.)
	sfIndex       uint8  // Sample frequency index
	sampleRate    uint32 // Actual sample rate in Hz
	channelConfig uint8  // Channel configuration
}

// parseAudioSpecificConfig parses an MP4 AudioSpecificConfig.
// This is a simplified local version to avoid import cycles.
//
// Ported from: AudioSpecificConfigFromBitfile() in ~/dev/faad2/libfaad/mp4.c:127-297
func parseAudioSpecificConfig(r *bits.Reader, _ uint32) (*mp4AudioSpecificConfig, error) {
	asc := &mp4AudioSpecificConfig{}

	// 5 bits: audioObjectType
	asc.objectType = uint8(r.GetBits(5))

	// 4 bits: samplingFrequencyIndex
	asc.sfIndex = uint8(r.GetBits(4))

	// If sfIndex == 0x0F, read 24-bit explicit sample rate
	if asc.sfIndex == 0x0F {
		asc.sampleRate = r.GetBits(24)
	} else {
		asc.sampleRate = getSampleRate(asc.sfIndex)
	}

	// 4 bits: channelConfiguration
	asc.channelConfig = uint8(r.GetBits(4))

	// Note: We skip GASpecificConfig parsing for basic initialization.
	// Full parsing would include frameLengthFlag, dependsOnCoreCoder, extensionFlag.
	// For now, we use default frameLength (1024) set in NewDecoder().

	return asc, nil
}
