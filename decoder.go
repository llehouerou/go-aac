// decoder.go
package aac

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
