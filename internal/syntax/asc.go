// internal/syntax/asc.go
package syntax

import (
	"errors"

	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/bits"
	"github.com/llehouerou/go-aac/internal/tables"
)

// ASC parsing errors.
var (
	// ErrASCNil is returned when nil config is passed.
	ErrASCNil = errors.New("nil AudioSpecificConfig")

	// ErrASCUnsupportedObjectType is returned for unsupported object types.
	ErrASCUnsupportedObjectType = errors.New("unsupported audio object type")

	// ErrASCInvalidSampleRate is returned for invalid sample rate index.
	ErrASCInvalidSampleRate = errors.New("invalid sample rate")

	// ErrASCInvalidChannelConfig is returned for invalid channel configuration.
	ErrASCInvalidChannelConfig = errors.New("invalid channel configuration")

	// ErrASCGAConfigFailed is returned when GASpecificConfig parsing fails.
	ErrASCGAConfigFailed = errors.New("GASpecificConfig parsing failed")

	// ErrASCEPConfigNotSupported is returned for unsupported epConfig values.
	ErrASCEPConfigNotSupported = errors.New("epConfig != 0 not supported")

	// ErrASCBitstreamError is returned for bitstream initialization errors.
	ErrASCBitstreamError = errors.New("bitstream initialization error")
)

// SRIndexExplicit indicates an explicit 24-bit sample rate follows.
const SRIndexExplicit = 0x0f

// objectTypesTable defines which audio object types can be decoded.
// Ported from: ~/dev/faad2/libfaad/mp4.c:40-117 (ObjectTypesTable)
// This table assumes all optional features are enabled:
// MAIN_DEC, LTP_DEC, SBR_DEC, ERROR_RESILIENCE, LD_DEC, PS_DEC
//
// Note: This table is specifically for ASC (AudioSpecificConfig) parsing.
// A separate CanDecodeOT() function exists in tables/sample_rates.go (from common.c)
// for runtime object type validation. The apparent discrepancy at index 27 is
// intentional: MPEG-4 defines index 27 as "ER Parametric" (not supported here),
// but DRM mode uses index 27 for "DRM ER LC" (supported via CanDecodeOT).
var objectTypesTable = [32]bool{
	false, // 0: NULL
	true,  // 1: AAC Main
	true,  // 2: AAC LC
	false, // 3: AAC SSR (not supported)
	true,  // 4: AAC LTP
	true,  // 5: SBR (HE-AAC)
	false, // 6: AAC Scalable
	false, // 7: TwinVQ
	false, // 8: CELP
	false, // 9: HVXC
	false, // 10: Reserved
	false, // 11: Reserved
	false, // 12: TTSI
	false, // 13: Main synthetic
	false, // 14: Wavetable synthesis
	false, // 15: General MIDI
	false, // 16: Algorithmic Synthesis and Audio FX
	true,  // 17: ER AAC LC
	false, // 18: Reserved
	true,  // 19: ER AAC LTP
	false, // 20: ER AAC scalable
	false, // 21: ER TwinVQ
	false, // 22: ER BSAC
	true,  // 23: ER AAC LD
	false, // 24: ER CELP
	false, // 25: ER HVXC
	false, // 26: ER HILN
	false, // 27: ER Parametric
	false, // 28: Reserved
	true,  // 29: AAC LC + SBR + PS (HE-AACv2)
	false, // 30: Reserved
	false, // 31: Reserved
}

// isObjectTypeSupported returns true if the audio object type can be decoded.
// Ported from: ~/dev/faad2/libfaad/mp4.c:40-117
func isObjectTypeSupported(objType uint8) bool {
	if objType >= 32 {
		return false
	}
	return objectTypesTable[objType]
}

// parseGASpecificConfig parses the General Audio Specific Config.
// Returns the parsed PCE if channelsConfiguration is 0, otherwise nil.
//
// Ported from: ~/dev/faad2/libfaad/syntax.c:109-165
func parseGASpecificConfig(r *bits.Reader, asc *aac.AudioSpecificConfig) (*ProgramConfig, error) {
	// 1 bit: frameLengthFlag (0 = 1024, 1 = 960)
	// Note: FAAD2 conditionally rejects frameLengthFlag=1 unless ALLOW_SMALL_FRAMELENGTH
	// is defined. This implementation allows both frame lengths for flexibility.
	asc.FrameLengthFlag = r.Get1Bit() == 1

	// 1 bit: dependsOnCoreCoder
	asc.DependsOnCoreCoder = r.Get1Bit() == 1
	if asc.DependsOnCoreCoder {
		// 14 bits: coreCoderDelay
		asc.CoreCoderDelay = uint16(r.GetBits(14))
	}

	// 1 bit: extensionFlag
	asc.ExtensionFlag = r.Get1Bit() == 1

	// If channelsConfiguration == 0, parse PCE
	var pce *ProgramConfig
	if asc.ChannelsConfiguration == 0 {
		var err error
		pce, err = ParsePCE(r)
		if err != nil {
			return nil, err
		}
	}

	// Handle extensionFlag for ER object types
	if asc.ExtensionFlag {
		if asc.ObjectTypeIndex >= ERObjectStart {
			// 1 bit each: resilience flags
			asc.AACSectionDataResilienceFlag = r.Get1Bit() == 1
			asc.AACScalefactorDataResilienceFlag = r.Get1Bit() == 1
			asc.AACSpectralDataResilienceFlag = r.Get1Bit() == 1
		}
		// 1 bit: extensionFlag3 (reserved, skip)
		r.FlushBits(1)
	}

	return pce, nil
}

// ParseASC parses an AudioSpecificConfig from raw bytes.
// Returns the parsed config, optional PCE, and any error.
//
// Ported from: ~/dev/faad2/libfaad/mp4.c:299-313 (AudioSpecificConfig2)
func ParseASC(data []byte) (*aac.AudioSpecificConfig, *ProgramConfig, error) {
	r := bits.NewReader(data)
	if r.Error() {
		return nil, nil, ErrASCBitstreamError
	}
	return ParseASCFromBitstream(r, uint32(len(data)), false)
}

// ParseASCShortForm parses an AudioSpecificConfig without SBR extension detection.
// Use this when you know there's no SBR extension data in the config.
//
// Ported from: ~/dev/faad2/libfaad/mp4.c short_form parameter
func ParseASCShortForm(data []byte) (*aac.AudioSpecificConfig, *ProgramConfig, error) {
	r := bits.NewReader(data)
	if r.Error() {
		return nil, nil, ErrASCBitstreamError
	}
	return ParseASCFromBitstream(r, uint32(len(data)), true)
}

// ParseASCFromBitstream parses an AudioSpecificConfig from a bitstream.
// bufferSize is the total size available.
// shortForm disables SBR extension parsing when true.
//
// Ported from: ~/dev/faad2/libfaad/mp4.c:127-297 (AudioSpecificConfigFromBitfile)
func ParseASCFromBitstream(r *bits.Reader, bufferSize uint32, shortForm bool) (*aac.AudioSpecificConfig, *ProgramConfig, error) {
	asc := &aac.AudioSpecificConfig{}
	startPos := r.GetProcessedBits()

	// 5 bits: objectTypeIndex
	asc.ObjectTypeIndex = uint8(r.GetBits(5))

	// 4 bits: samplingFrequencyIndex
	asc.SamplingFrequencyIndex = uint8(r.GetBits(4))
	if asc.SamplingFrequencyIndex == SRIndexExplicit {
		// 24 bits: explicit sampling frequency
		// Note: FAAD2 (mp4.c:147-148) reads this value but discards it, then calls
		// get_sample_rate(0x0f) which returns 0, failing with "invalid sample rate".
		// This implementation correctly stores the explicit sample rate.
		asc.SamplingFrequency = r.GetBits(24)
	} else {
		asc.SamplingFrequency = tables.GetSampleRate(asc.SamplingFrequencyIndex)
	}

	// 4 bits: channelsConfiguration
	asc.ChannelsConfiguration = uint8(r.GetBits(4))

	// Validate object type
	if !isObjectTypeSupported(asc.ObjectTypeIndex) {
		return nil, nil, ErrASCUnsupportedObjectType
	}

	// Validate sample rate
	if asc.SamplingFrequency == 0 {
		return nil, nil, ErrASCInvalidSampleRate
	}

	// Validate channel config
	if asc.ChannelsConfiguration > 7 {
		return nil, nil, ErrASCInvalidChannelConfig
	}

	// Upmatrix mono to stereo for implicit PS signaling
	if asc.ChannelsConfiguration == 1 {
		asc.ChannelsConfiguration = 2
	}

	// Initialize SBR present flag to unknown (-1)
	asc.SBRPresentFlag = -1

	// Handle explicit SBR signaling (object types 5 and 29)
	if asc.ObjectTypeIndex == 5 || asc.ObjectTypeIndex == 29 {
		asc.SBRPresentFlag = 1

		// 4 bits: extensionSamplingFrequencyIndex
		extSRIndex := uint8(r.GetBits(4))

		// Check for downsampled SBR
		if extSRIndex == asc.SamplingFrequencyIndex {
			asc.DownSampledSBR = true
		}
		asc.SamplingFrequencyIndex = extSRIndex

		if asc.SamplingFrequencyIndex == SRIndexExplicit {
			asc.SamplingFrequency = r.GetBits(24)
		} else {
			asc.SamplingFrequency = tables.GetSampleRate(asc.SamplingFrequencyIndex)
		}

		// 5 bits: new objectTypeIndex (the core codec type)
		asc.ObjectTypeIndex = uint8(r.GetBits(5))
	}

	// Parse GASpecificConfig for appropriate object types
	var pce *ProgramConfig
	var err error
	switch asc.ObjectTypeIndex {
	case 1, 2, 3, 4, 6, 7: // Main, LC, SSR, LTP, Scalable, TwinVQ
		pce, err = parseGASpecificConfig(r, asc)
		if err != nil {
			return nil, nil, ErrASCGAConfigFailed
		}
	default:
		if asc.ObjectTypeIndex >= ERObjectStart {
			// ER object types
			pce, err = parseGASpecificConfig(r, asc)
			if err != nil {
				return nil, nil, ErrASCGAConfigFailed
			}
			// 2 bits: epConfig
			asc.EPConfig = uint8(r.GetBits(2))
			if asc.EPConfig != 0 {
				return nil, nil, ErrASCEPConfigNotSupported
			}
		} else {
			return nil, nil, ErrASCUnsupportedObjectType
		}
	}

	// Handle implicit SBR signaling (backward compatible extension)
	if !shortForm {
		// Calculate remaining bits in ASC buffer for SBR extension detection.
		// Note: FAAD2 uses buffer_size*8 + processed - start, which appears incorrect.
		// This implementation uses the mathematically correct formula: total - consumed.
		bitsToDecode := int32(bufferSize*8) - int32(r.GetProcessedBits()-startPos)
		if asc.ObjectTypeIndex != 5 && asc.ObjectTypeIndex != 29 && bitsToDecode >= 16 {
			// Look for syncExtensionType
			syncExtType := r.GetBits(11)
			if syncExtType == 0x2b7 {
				// 5 bits: extensionAudioObjectType
				extOTi := uint8(r.GetBits(5))
				if extOTi == 5 {
					// 1 bit: sbrPresentFlag
					asc.SBRPresentFlag = int8(r.Get1Bit())
					if asc.SBRPresentFlag == 1 {
						asc.ObjectTypeIndex = extOTi

						// 4 bits: extensionSamplingFrequencyIndex
						extSRIndex := uint8(r.GetBits(4))

						// Check for downsampled SBR
						if extSRIndex == asc.SamplingFrequencyIndex {
							asc.DownSampledSBR = true
						}
						asc.SamplingFrequencyIndex = extSRIndex

						if asc.SamplingFrequencyIndex == SRIndexExplicit {
							asc.SamplingFrequency = r.GetBits(24)
						} else {
							asc.SamplingFrequency = tables.GetSampleRate(asc.SamplingFrequencyIndex)
						}
					}
				}
			}
		}
	}

	// Handle implicit SBR based on sample rate
	if asc.SBRPresentFlag == -1 {
		if asc.SamplingFrequency <= 24000 {
			asc.SamplingFrequency *= 2
			asc.ForceUpSampling = true
		} else {
			asc.DownSampledSBR = true
		}
	}

	return asc, pce, nil
}
