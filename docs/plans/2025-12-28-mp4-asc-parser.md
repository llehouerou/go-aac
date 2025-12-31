# MP4 AudioSpecificConfig Parser Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement MP4 AudioSpecificConfig (ASC) parser to initialize decoder from MP4/M4A container esds atom.

**Architecture:** Parse AudioSpecificConfig bitstream to extract audio object type, sample rate, channels, and GA-specific extensions. Reuse existing PCE parser for channel configuration = 0 case. Handle SBR explicit and implicit signaling.

**Tech Stack:** Go, bits.Reader, existing tables/sample_rates package, existing syntax/pce package.

---

## Background

The AudioSpecificConfig is embedded in MP4 containers (in the esds atom) and describes:
- Audio Object Type (5 bits)
- Sampling Frequency (4 bits index + optional 24-bit explicit)
- Channel Configuration (4 bits)
- GA-Specific Config (frame length, extension flags)
- Optional SBR/PS extension signaling

**FAAD2 Source Files:**
- `~/dev/faad2/libfaad/mp4.c:127-313` - Main parsing logic
- `~/dev/faad2/libfaad/syntax.c:109-165` - GASpecificConfig
- `~/dev/faad2/include/neaacdec.h:140-161` - mp4AudioSpecificConfig struct

**Existing Go Code:**
- `aac.go:120-141` - AudioSpecificConfig struct (already defined)
- `internal/tables/sample_rates.go` - GetSampleRate, CanDecodeOT
- `internal/syntax/pce.go` - ParsePCE

---

## Task 1: Add ASC Error Types

**Files:**
- Modify: `internal/syntax/asc.go` (create new file)

**Step 1: Create asc.go with error definitions**

```go
// internal/syntax/asc.go
package syntax

import (
	"errors"
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
```

**Step 2: Run test to verify compilation**

Run: `go build ./internal/syntax/`
Expected: Success (no errors)

**Step 3: Commit**

```bash
git add internal/syntax/asc.go
git commit -m "feat(syntax): add ASC error definitions

Ported from: ~/dev/faad2/libfaad/mp4.c error codes"
```

---

## Task 2: Add ObjectTypesTable for Validation

**Files:**
- Modify: `internal/syntax/asc.go`

**Step 1: Write test for object type validation**

Create `internal/syntax/asc_test.go`:

```go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac"
)

func TestIsObjectTypeSupported(t *testing.T) {
	tests := []struct {
		name     string
		objType  uint8
		expected bool
	}{
		{"NULL type", 0, false},
		{"AAC Main", 1, true},
		{"AAC LC", 2, true},
		{"AAC SSR", 3, false}, // SSR not supported
		{"AAC LTP", 4, true},
		{"HE-AAC (SBR)", 5, true},
		{"AAC Scalable", 6, false},
		{"ER AAC LC", 17, true},
		{"ER AAC LTP", 19, true},
		{"ER AAC LD", 23, true},
		{"Reserved 28", 28, false},
		{"AAC LC + SBR + PS", 29, true},
		{"Reserved 31", 31, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isObjectTypeSupported(tt.objType)
			if got != tt.expected {
				t.Errorf("isObjectTypeSupported(%d) = %v, want %v", tt.objType, got, tt.expected)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/syntax/ -run TestIsObjectTypeSupported`
Expected: FAIL with "undefined: isObjectTypeSupported"

**Step 3: Implement isObjectTypeSupported**

Add to `internal/syntax/asc.go`:

```go
// objectTypesTable defines which object types can be decoded.
// Source: ~/dev/faad2/libfaad/mp4.c:40-117
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
	false, // 16: Algorithmic Synthesis
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
	true,  // 29: AAC LC + SBR + PS
	false, // 30: Reserved
	false, // 31: Reserved
}

// isObjectTypeSupported returns true if the object type can be decoded.
// Source: ~/dev/faad2/libfaad/mp4.c:40-117
func isObjectTypeSupported(objType uint8) bool {
	if objType >= 32 {
		return false
	}
	return objectTypesTable[objType]
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./internal/syntax/ -run TestIsObjectTypeSupported`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/asc.go internal/syntax/asc_test.go
git commit -m "feat(syntax): add objectTypesTable for ASC validation

Ported from: ~/dev/faad2/libfaad/mp4.c:40-117"
```

---

## Task 3: Implement GASpecificConfig Parser

**Files:**
- Modify: `internal/syntax/asc.go`
- Modify: `internal/syntax/asc_test.go`

**Step 1: Write test for GASpecificConfig**

Add to `internal/syntax/asc_test.go`:

```go
func TestParseGASpecificConfig(t *testing.T) {
	tests := []struct {
		name             string
		data             []byte
		channelConfig    uint8
		objectType       uint8
		wantFrameLen     bool
		wantDependsCore  bool
		wantExtension    bool
		wantErr          bool
	}{
		{
			name:          "basic LC no extensions",
			data:          []byte{0x00}, // 0b00000000: frameLenFlag=0, dependsOnCore=0, extensionFlag=0
			channelConfig: 2,
			objectType:    2, // LC
			wantFrameLen:  false,
			wantDependsCore: false,
			wantExtension: false,
			wantErr:       false,
		},
		{
			name:          "with 960 frame length",
			data:          []byte{0x80}, // 0b10000000: frameLenFlag=1
			channelConfig: 2,
			objectType:    2,
			wantFrameLen:  true,
			wantDependsCore: false,
			wantExtension: false,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bits.NewReader(tt.data)
			asc := &aac.AudioSpecificConfig{
				ChannelsConfiguration: tt.channelConfig,
				ObjectTypeIndex:       tt.objectType,
			}

			pce, err := parseGASpecificConfig(r, asc)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGASpecificConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if asc.FrameLengthFlag != tt.wantFrameLen {
				t.Errorf("FrameLengthFlag = %v, want %v", asc.FrameLengthFlag, tt.wantFrameLen)
			}
			if asc.DependsOnCoreCoder != tt.wantDependsCore {
				t.Errorf("DependsOnCoreCoder = %v, want %v", asc.DependsOnCoreCoder, tt.wantDependsCore)
			}
			if asc.ExtensionFlag != tt.wantExtension {
				t.Errorf("ExtensionFlag = %v, want %v", asc.ExtensionFlag, tt.wantExtension)
			}
			// PCE should be nil when channelConfig != 0
			if tt.channelConfig != 0 && pce != nil {
				t.Error("Expected nil PCE for channelConfig != 0")
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/syntax/ -run TestParseGASpecificConfig`
Expected: FAIL with "undefined: parseGASpecificConfig"

**Step 3: Implement parseGASpecificConfig**

Add to `internal/syntax/asc.go`:

```go
import (
	"errors"

	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/bits"
)

// parseGASpecificConfig parses the General Audio Specific Config.
// Returns the parsed PCE if channelsConfiguration is 0, otherwise nil.
//
// Source: ~/dev/faad2/libfaad/syntax.c:109-165
func parseGASpecificConfig(r *bits.Reader, asc *aac.AudioSpecificConfig) (*ProgramConfig, error) {
	// 1 bit: frameLengthFlag (0 = 1024, 1 = 960)
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
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./internal/syntax/ -run TestParseGASpecificConfig`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/asc.go internal/syntax/asc_test.go
git commit -m "feat(syntax): implement parseGASpecificConfig

Ported from: ~/dev/faad2/libfaad/syntax.c:109-165"
```

---

## Task 4: Implement Core ASC Parser (ParseASC)

**Files:**
- Modify: `internal/syntax/asc.go`
- Modify: `internal/syntax/asc_test.go`

**Step 1: Write test for ParseASC**

Add to `internal/syntax/asc_test.go`:

```go
func TestParseASC(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		wantObjType    uint8
		wantSRIndex    uint8
		wantSampleRate uint32
		wantChannels   uint8
		wantErr        error
	}{
		{
			name: "AAC-LC 44100Hz stereo",
			// 0x12 0x10 = 0001 0010 0001 0000
			// objType=2 (5 bits: 00010), srIndex=4 (4 bits: 0100), channels=2 (4 bits: 0010), GASpec...
			data:           []byte{0x12, 0x10},
			wantObjType:    2,
			wantSRIndex:    4,
			wantSampleRate: 44100,
			wantChannels:   2,
			wantErr:        nil,
		},
		{
			name: "AAC-LC 48000Hz stereo",
			// objType=2, srIndex=3, channels=2
			// 00010 0011 0010 0... = 0x11 0x90
			data:           []byte{0x11, 0x90},
			wantObjType:    2,
			wantSRIndex:    3,
			wantSampleRate: 48000,
			wantChannels:   2,
			wantErr:        nil,
		},
		{
			name: "unsupported object type (SSR)",
			// objType=3 (SSR), srIndex=4, channels=2
			// 00011 0100 0010 = 0x1A 0x10
			data:        []byte{0x1A, 0x10},
			wantObjType: 3,
			wantErr:     ErrASCUnsupportedObjectType,
		},
		{
			name: "AAC-LC mono",
			// objType=2, srIndex=4, channels=1
			// 00010 0100 0001 0... = 0x12 0x08
			data:           []byte{0x12, 0x08},
			wantObjType:    2,
			wantSRIndex:    4,
			wantSampleRate: 44100,
			wantChannels:   1, // Will be upmatrix to 2 for PS support
			wantErr:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asc, _, err := ParseASC(tt.data)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseASC() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseASC() unexpected error = %v", err)
				return
			}

			if asc.ObjectTypeIndex != tt.wantObjType {
				t.Errorf("ObjectTypeIndex = %d, want %d", asc.ObjectTypeIndex, tt.wantObjType)
			}
			if asc.SamplingFrequencyIndex != tt.wantSRIndex {
				t.Errorf("SamplingFrequencyIndex = %d, want %d", asc.SamplingFrequencyIndex, tt.wantSRIndex)
			}
			if asc.SamplingFrequency != tt.wantSampleRate {
				t.Errorf("SamplingFrequency = %d, want %d", asc.SamplingFrequency, tt.wantSampleRate)
			}
			// Note: mono is upmatrix to stereo for PS support
			expectedChannels := tt.wantChannels
			if tt.wantChannels == 1 {
				expectedChannels = 2
			}
			if asc.ChannelsConfiguration != expectedChannels {
				t.Errorf("ChannelsConfiguration = %d, want %d", asc.ChannelsConfiguration, expectedChannels)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/syntax/ -run TestParseASC`
Expected: FAIL with "undefined: ParseASC"

**Step 3: Implement ParseASC**

Add to `internal/syntax/asc.go`:

```go
import (
	"github.com/llehouerou/go-aac/internal/tables"
)

// ParseASC parses an AudioSpecificConfig from raw bytes.
// Returns the parsed config, optional PCE, and any error.
//
// Source: ~/dev/faad2/libfaad/mp4.c:299-313 (AudioSpecificConfig2)
func ParseASC(data []byte) (*aac.AudioSpecificConfig, *ProgramConfig, error) {
	r := bits.NewReader(data)
	return ParseASCFromBitstream(r, uint32(len(data)), false)
}

// ParseASCFromBitstream parses an AudioSpecificConfig from a bitstream.
// bufferSize is the total size available.
// shortForm disables SBR extension parsing when true.
//
// Source: ~/dev/faad2/libfaad/mp4.c:127-297 (AudioSpecificConfigFromBitfile)
func ParseASCFromBitstream(r *bits.Reader, bufferSize uint32, shortForm bool) (*aac.AudioSpecificConfig, *ProgramConfig, error) {
	asc := &aac.AudioSpecificConfig{}
	startPos := r.GetProcessedBits()

	// 5 bits: objectTypeIndex
	asc.ObjectTypeIndex = uint8(r.GetBits(5))

	// 4 bits: samplingFrequencyIndex
	asc.SamplingFrequencyIndex = uint8(r.GetBits(4))
	if asc.SamplingFrequencyIndex == 0x0f {
		// 24 bits: explicit sampling frequency
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

		if asc.SamplingFrequencyIndex == 15 {
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
		bitsToDecose := int32(bufferSize*8) - int32(r.GetProcessedBits()-startPos)
		if asc.ObjectTypeIndex != 5 && asc.ObjectTypeIndex != 29 && bitsToDecose >= 16 {
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

						if asc.SamplingFrequencyIndex == 15 {
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
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./internal/syntax/ -run TestParseASC`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/asc.go internal/syntax/asc_test.go
git commit -m "feat(syntax): implement ParseASC for AudioSpecificConfig

Ported from: ~/dev/faad2/libfaad/mp4.c:127-313"
```

---

## Task 5: Add ParseASCShortForm Convenience Function

**Files:**
- Modify: `internal/syntax/asc.go`
- Modify: `internal/syntax/asc_test.go`

**Step 1: Write test for ParseASCShortForm**

Add to `internal/syntax/asc_test.go`:

```go
func TestParseASCShortForm(t *testing.T) {
	// AAC-LC 44100Hz stereo - should not apply implicit SBR heuristics
	data := []byte{0x12, 0x10}

	asc, _, err := ParseASCShortForm(data)
	if err != nil {
		t.Fatalf("ParseASCShortForm() error = %v", err)
	}

	// With short form, SBR should remain unknown (-1) and no upsampling
	if asc.SBRPresentFlag != -1 {
		t.Errorf("SBRPresentFlag = %d, want -1", asc.SBRPresentFlag)
	}
	// Sample rate should be original 44100 (no doubling for implicit SBR)
	if asc.SamplingFrequency != 44100 {
		t.Errorf("SamplingFrequency = %d, want 44100", asc.SamplingFrequency)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/syntax/ -run TestParseASCShortForm`
Expected: FAIL with "undefined: ParseASCShortForm"

**Step 3: Implement ParseASCShortForm**

Add to `internal/syntax/asc.go`:

```go
// ParseASCShortForm parses an AudioSpecificConfig without SBR extension detection.
// Use this when you know there's no SBR extension data in the config.
//
// Source: ~/dev/faad2/libfaad/mp4.c short_form parameter
func ParseASCShortForm(data []byte) (*aac.AudioSpecificConfig, *ProgramConfig, error) {
	r := bits.NewReader(data)
	return ParseASCFromBitstream(r, uint32(len(data)), true)
}
```

**Step 4: Fix the test expectation**

Looking at FAAD2 code, when shortForm is true, it skips SBR extension parsing but still applies the implicit SBR heuristics at the end. Update the test:

```go
func TestParseASCShortForm(t *testing.T) {
	// AAC-LC 44100Hz stereo - shortForm skips extension parsing
	data := []byte{0x12, 0x10}

	asc, _, err := ParseASCShortForm(data)
	if err != nil {
		t.Fatalf("ParseASCShortForm() error = %v", err)
	}

	// With short form, SBR remains unknown so implicit heuristics apply
	// 44100 > 24000, so downSampledSBR should be set
	if !asc.DownSampledSBR {
		t.Error("DownSampledSBR should be true for sr > 24000 with unknown SBR")
	}
	// Sample rate stays at 44100 (no doubling because > 24000)
	if asc.SamplingFrequency != 44100 {
		t.Errorf("SamplingFrequency = %d, want 44100", asc.SamplingFrequency)
	}
}
```

**Step 5: Run test to verify it passes**

Run: `go test -v ./internal/syntax/ -run TestParseASCShortForm`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/syntax/asc.go internal/syntax/asc_test.go
git commit -m "feat(syntax): add ParseASCShortForm convenience function"
```

---

## Task 6: Add Integration Tests with Real ASC Data

**Files:**
- Modify: `internal/syntax/asc_test.go`

**Step 1: Write integration test with known ASC bytes**

Add to `internal/syntax/asc_test.go`:

```go
func TestParseASC_RealWorldConfigs(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		wantObjType    uint8
		wantSampleRate uint32
		wantChannels   uint8
		wantSBR        int8
	}{
		{
			// Common iTunes AAC-LC config
			name:           "iTunes AAC-LC 44100 stereo",
			data:           []byte{0x12, 0x10}, // LC, 44100, stereo
			wantObjType:    2,
			wantSampleRate: 44100,
			wantChannels:   2,
			wantSBR:        -1, // unknown -> implicit handling
		},
		{
			// HE-AAC config with explicit SBR
			name:           "HE-AAC 22050->44100 stereo",
			data:           []byte{0x2B, 0x92, 0x08, 0x00}, // SBR type, core 22050, ext 44100
			wantObjType:    5,                              // Becomes SBR type
			wantSampleRate: 44100,                          // Output rate
			wantChannels:   2,
			wantSBR:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asc, _, err := ParseASC(tt.data)
			if err != nil {
				t.Fatalf("ParseASC() error = %v", err)
			}

			if asc.ObjectTypeIndex != tt.wantObjType {
				t.Errorf("ObjectTypeIndex = %d, want %d", asc.ObjectTypeIndex, tt.wantObjType)
			}
			// For implicit SBR handling, sample rate may be doubled
			// Just verify it's reasonable
			if tt.wantSBR == -1 {
				// Implicit handling: either original or doubled
				if asc.SamplingFrequency != tt.wantSampleRate && asc.SamplingFrequency != tt.wantSampleRate*2 {
					t.Errorf("SamplingFrequency = %d, want %d or %d", asc.SamplingFrequency, tt.wantSampleRate, tt.wantSampleRate*2)
				}
			} else {
				if asc.SamplingFrequency != tt.wantSampleRate {
					t.Errorf("SamplingFrequency = %d, want %d", asc.SamplingFrequency, tt.wantSampleRate)
				}
			}
			if asc.ChannelsConfiguration != tt.wantChannels {
				t.Errorf("ChannelsConfiguration = %d, want %d", asc.ChannelsConfiguration, tt.wantChannels)
			}
			if tt.wantSBR != -1 && asc.SBRPresentFlag != tt.wantSBR {
				t.Errorf("SBRPresentFlag = %d, want %d", asc.SBRPresentFlag, tt.wantSBR)
			}
		})
	}
}
```

**Step 2: Run test to verify**

Run: `go test -v ./internal/syntax/ -run TestParseASC_RealWorldConfigs`
Expected: PASS (if implementation is correct) or useful failure info

**Step 3: Commit**

```bash
git add internal/syntax/asc_test.go
git commit -m "test(syntax): add real-world ASC integration tests"
```

---

## Task 7: Run Full Test Suite and Lint

**Files:** None (verification only)

**Step 1: Run all syntax tests**

Run: `go test -v ./internal/syntax/`
Expected: All tests PASS

**Step 2: Run linter**

Run: `make lint`
Expected: No errors

**Step 3: Run full check**

Run: `make check`
Expected: All checks pass

**Step 4: Commit any fixes if needed**

If any lint issues found, fix and commit.

---

## Task 8: Final Verification Against FAAD2

**Files:**
- Modify: `internal/syntax/asc_test.go` (add FAAD2 comparison if possible)

**Step 1: Create FAAD2 reference test (optional)**

If `scripts/faad2_debug` tool supports ASC parsing, add comparison test.
Otherwise, manually verify a few ASC configs parse identically to FAAD2.

**Step 2: Document any differences**

Add comments in `asc.go` for any intentional deviations from FAAD2 behavior.

**Step 3: Final commit**

```bash
git add -A
git commit -m "feat(syntax): complete MP4 AudioSpecificConfig parser

- ParseASC: Parse from raw bytes
- ParseASCFromBitstream: Parse from bits.Reader
- ParseASCShortForm: Skip SBR extension detection
- parseGASpecificConfig: Parse GA-specific fields
- Object type validation table
- SBR implicit/explicit signaling support

Ported from: ~/dev/faad2/libfaad/mp4.c, syntax.c"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Add ASC error types | asc.go |
| 2 | Add objectTypesTable | asc.go, asc_test.go |
| 3 | Implement parseGASpecificConfig | asc.go, asc_test.go |
| 4 | Implement ParseASC | asc.go, asc_test.go |
| 5 | Add ParseASCShortForm | asc.go, asc_test.go |
| 6 | Integration tests | asc_test.go |
| 7 | Full verification | - |
| 8 | FAAD2 comparison | asc_test.go |

**Total new files:** 2 (internal/syntax/asc.go, internal/syntax/asc_test.go)
**Estimated lines:** ~250 implementation + ~150 tests
