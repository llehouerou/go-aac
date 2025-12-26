// Package aac provides a pure Go AAC decoder.
// Ported from FAAD2: ~/dev/faad2/
package aac

// ObjectType represents an AAC audio object type.
// Source: ~/dev/faad2/include/neaacdec.h:74-83
type ObjectType uint8

// AAC Object Types.
const (
	ObjectTypeMain    ObjectType = 1
	ObjectTypeLC      ObjectType = 2  // Most common - Low Complexity
	ObjectTypeSSR     ObjectType = 3  // Scalable Sample Rate
	ObjectTypeLTP     ObjectType = 4  // Long Term Prediction
	ObjectTypeHEAAC   ObjectType = 5  // High Efficiency AAC (with SBR)
	ObjectTypeERLC    ObjectType = 17 // Error Resilient LC
	ObjectTypeERLTP   ObjectType = 19 // Error Resilient LTP
	ObjectTypeLD      ObjectType = 23 // Low Delay
	ObjectTypeDRMERLC ObjectType = 27 // DRM specific
)

// HeaderType represents an AAC stream header type.
// Source: ~/dev/faad2/include/neaacdec.h:85-89
type HeaderType uint8

// Header Types.
const (
	HeaderTypeRAW  HeaderType = 0 // Raw AAC data, no header
	HeaderTypeADIF HeaderType = 1 // Audio Data Interchange Format
	HeaderTypeADTS HeaderType = 2 // Audio Data Transport Stream
	HeaderTypeLATM HeaderType = 3 // Low-latency Audio Transport Multiplex
)

// OutputFormat represents the PCM output sample format.
// Source: ~/dev/faad2/include/neaacdec.h:97-103
type OutputFormat uint8

// Output Formats.
const (
	OutputFormat16Bit  OutputFormat = 1 // 16-bit signed integer
	OutputFormat24Bit  OutputFormat = 2 // 24-bit signed integer
	OutputFormat32Bit  OutputFormat = 3 // 32-bit signed integer
	OutputFormatFloat  OutputFormat = 4 // 32-bit float
	OutputFormatDouble OutputFormat = 5 // 64-bit float
)

// ChannelPosition represents the spatial position of an audio channel.
// Source: ~/dev/faad2/include/neaacdec.h:113-123
type ChannelPosition uint8

// Channel Positions.
const (
	ChannelUnknown     ChannelPosition = 0
	ChannelFrontCenter ChannelPosition = 1
	ChannelFrontLeft   ChannelPosition = 2
	ChannelFrontRight  ChannelPosition = 3
	ChannelSideLeft    ChannelPosition = 4
	ChannelSideRight   ChannelPosition = 5
	ChannelBackLeft    ChannelPosition = 6
	ChannelBackRight   ChannelPosition = 7
	ChannelBackCenter  ChannelPosition = 8
	ChannelLFE         ChannelPosition = 9 // Low Frequency Effects
)

// SBRSignalling represents the SBR (Spectral Band Replication) status.
// Source: ~/dev/faad2/include/neaacdec.h:91-95
type SBRSignalling uint8

// SBR Signalling values.
const (
	SBRNone          SBRSignalling = 0 // No SBR
	SBRUpsampled     SBRSignalling = 1 // SBR with upsampling
	SBRDownsampled   SBRSignalling = 2 // SBR with downsampling
	SBRNoneUpsampled SBRSignalling = 3 // No SBR but upsampled
)

// MinStreamSize is the minimum bytes per channel that should be available.
// Source: ~/dev/faad2/include/neaacdec.h:135
const MinStreamSize = 768 // 6144 bits/channel

// Config contains decoder configuration options.
// Source: ~/dev/faad2/include/neaacdec.h:163-171
type Config struct {
	DefObjectType           ObjectType   // Default object type
	DefSampleRate           uint32       // Default sample rate
	OutputFormat            OutputFormat // Output sample format
	DownMatrix              bool         // Downmix multichannel to stereo
	UseOldADTSFormat        bool         // Use old ADTS format
	DontUpSampleImplicitSBR bool         // Don't upsample implicit SBR
}

// FrameInfo contains information about a decoded frame.
// Source: ~/dev/faad2/include/neaacdec.h:173-199
type FrameInfo struct {
	BytesConsumed uint32 // Bytes consumed from input
	Samples       uint32 // Total PCM samples output
	Channels      uint8  // Number of output channels
	Error         Error  // Error code (0 = no error)
	SampleRate    uint32 // Output sample rate

	// SBR status: 0=off, 1=upsampled, 2=downsampled, 3=off but upsampled
	SBR SBRSignalling

	ObjectType ObjectType // MPEG-4 ObjectType
	HeaderType HeaderType // AAC header type (RAW, ADIF, ADTS, LATM)

	// Multichannel configuration
	NumFrontChannels uint8
	NumSideChannels  uint8
	NumBackChannels  uint8
	NumLFEChannels   uint8
	ChannelPosition  [64]ChannelPosition

	// Parametric Stereo: 0=off, 1=on
	PS uint8
}

// AudioSpecificConfig contains the MP4 AudioSpecificConfig data.
// Source: ~/dev/faad2/include/neaacdec.h:140-161
type AudioSpecificConfig struct {
	// Audio Specific Info
	ObjectTypeIndex        uint8
	SamplingFrequencyIndex uint8
	SamplingFrequency      uint32
	ChannelsConfiguration  uint8

	// GA Specific Info
	FrameLengthFlag                  bool
	DependsOnCoreCoder               bool
	CoreCoderDelay                   uint16
	ExtensionFlag                    bool
	AACSectionDataResilienceFlag     bool
	AACScalefactorDataResilienceFlag bool
	AACSpectralDataResilienceFlag    bool
	EPConfig                         uint8

	// SBR extension
	SBRPresentFlag  int8
	ForceUpSampling bool
	DownSampledSBR  bool
}
