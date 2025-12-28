package syntax

import (
	"fmt"
	"testing"

	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/bits"
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
		{"TwinVQ", 7, false},
		{"CELP", 8, false},
		{"HVXC", 9, false},
		{"Reserved 10", 10, false},
		{"Reserved 11", 11, false},
		{"TTSI", 12, false},
		{"Main synthetic", 13, false},
		{"Wavetable synthesis", 14, false},
		{"General MIDI", 15, false},
		{"Algorithmic Synthesis", 16, false},
		{"ER AAC LC", 17, true},
		{"Reserved 18", 18, false},
		{"ER AAC LTP", 19, true},
		{"ER AAC scalable", 20, false},
		{"ER TwinVQ", 21, false},
		{"ER BSAC", 22, false},
		{"ER AAC LD", 23, true},
		{"ER CELP", 24, false},
		{"ER HVXC", 25, false},
		{"ER HILN", 26, false},
		{"ER Parametric", 27, false},
		{"Reserved 28", 28, false},
		{"AAC LC + SBR + PS", 29, true},
		{"Reserved 30", 30, false},
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

func TestIsObjectTypeSupportedOutOfRange(t *testing.T) {
	// Object types >= 32 should return false
	tests := []uint8{32, 33, 64, 100, 255}
	for _, objType := range tests {
		t.Run(fmt.Sprintf("type_%d", objType), func(t *testing.T) {
			got := isObjectTypeSupported(objType)
			if got {
				t.Errorf("isObjectTypeSupported(%d) = true, want false for out-of-range type", objType)
			}
		})
	}
}

func TestParseGASpecificConfig(t *testing.T) {
	tests := []struct {
		name            string
		data            []byte
		channelConfig   uint8
		objectType      uint8
		wantFrameLen    bool
		wantDependsCore bool
		wantExtension   bool
		wantErr         bool
	}{
		{
			name:            "basic LC no extensions",
			data:            []byte{0x00}, // 0b00000000: frameLenFlag=0, dependsOnCore=0, extensionFlag=0
			channelConfig:   2,
			objectType:      2, // LC
			wantFrameLen:    false,
			wantDependsCore: false,
			wantExtension:   false,
			wantErr:         false,
		},
		{
			name:            "with 960 frame length",
			data:            []byte{0x80}, // 0b10000000: frameLenFlag=1
			channelConfig:   2,
			objectType:      2,
			wantFrameLen:    true,
			wantDependsCore: false,
			wantExtension:   false,
			wantErr:         false,
		},
		{
			name:            "depends on core coder",
			data:            []byte{0x40, 0x10, 0x00}, // 0b01000000 0b00010000 0b00000000: frameLenFlag=0, dependsOnCore=1, coreCoderDelay=256, extensionFlag=0
			channelConfig:   2,
			objectType:      2,
			wantFrameLen:    false,
			wantDependsCore: true,
			wantExtension:   false,
			wantErr:         false,
		},
		{
			name:            "extension flag set for non-ER type",
			data:            []byte{0x20}, // 0b00100000: frameLenFlag=0, dependsOnCore=0, extensionFlag=1, extensionFlag3=0
			channelConfig:   2,
			objectType:      2, // LC (not ER)
			wantFrameLen:    false,
			wantDependsCore: false,
			wantExtension:   true,
			wantErr:         false,
		},
		{
			name:            "extension flag set for ER type",
			data:            []byte{0x20}, // 0b00100000: frameLenFlag=0, dependsOnCore=0, extensionFlag=1, then 3 resilience bits + extensionFlag3
			channelConfig:   2,
			objectType:      17, // ER AAC LC
			wantFrameLen:    false,
			wantDependsCore: false,
			wantExtension:   true,
			wantErr:         false,
		},
		{
			name: "channelConfig 0 triggers PCE parsing",
			// Build data: GASpec (3 bits: frameLenFlag=0, dependsOnCore=0, extensionFlag=0)
			// Then PCE data follows:
			//   element_instance_tag: 4 bits = 5 (0101)
			//   object_type: 2 bits = 1 (01, AAC Main)
			//   sf_index: 4 bits = 4 (0100, 44100 Hz)
			//   numFrontChannelElements: 4 bits = 1 (0001)
			//   numSideChannelElements: 4 bits = 0 (0000)
			//   numBackChannelElements: 4 bits = 0 (0000)
			//   numLFEChannelElements: 2 bits = 0 (00)
			//   numAssocDataElements: 3 bits = 0 (000)
			//   numValidCCElements: 4 bits = 0 (0000)
			//   monoMixdownPresent: 1 bit = 0
			//   stereoMixdownPresent: 1 bit = 0
			//   matrixMixdownIdxPresent: 1 bit = 0
			//   frontElement[0]: isCPE=1 (1), tagSelect=0 (0000) => 2 channels via CPE
			//   byte align (3 bits padding to reach byte boundary)
			//   commentFieldBytes: 8 bits = 0
			//
			// Bit layout:
			// Byte 0: 000_0101_0 = GASpec(3) + tag(4) + objType(1/2) = 0x0A
			// Byte 1: 1_0100_000 = objType(1/2) + sfIdx(4) + frontCh(3/4) = 0xA0
			// Byte 2: 1_0000_00_ = frontCh(1/4) + sideCh(4) + backCh(3/4) = 0x80
			// Byte 3: 00_00_000_ = backCh(1/4) + lfeCh(2) + assoc(3) + cc(2/4) = 0x00
			// Byte 4: 00_0_0_0_1 = cc(2/4) + monoMix(1) + stereoMix(1) + matrixMix(1) + frontEl[0].isCPE(1) + frontEl[0].tag(2/4) = 0x04
			// Byte 5: 00_000_00 = frontEl[0].tag(2/4) + padding(3) + comment(3/8) = 0x00
			// Byte 6: 00000_xxx = comment(5/8) + padding = 0x00
			data:            []byte{0x0A, 0xA0, 0x80, 0x00, 0x04, 0x00, 0x00},
			channelConfig:   0, // Triggers PCE parsing
			objectType:      2, // LC
			wantFrameLen:    false,
			wantDependsCore: false,
			wantExtension:   false,
			wantErr:         false,
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

// TestParseGASpecificConfigPCE verifies that channelConfig=0 triggers PCE parsing
// and PCE fields are correctly populated.
func TestParseGASpecificConfigPCE(t *testing.T) {
	// Build data: GASpec (3 bits: frameLenFlag=0, dependsOnCore=0, extensionFlag=0)
	// Then PCE data follows:
	//   element_instance_tag: 4 bits = 5 (0101)
	//   object_type: 2 bits = 1 (01, AAC Main)
	//   sf_index: 4 bits = 4 (0100, 44100 Hz)
	//   numFrontChannelElements: 4 bits = 1 (0001)
	//   numSideChannelElements: 4 bits = 0 (0000)
	//   numBackChannelElements: 4 bits = 0 (0000)
	//   numLFEChannelElements: 2 bits = 0 (00)
	//   numAssocDataElements: 3 bits = 0 (000)
	//   numValidCCElements: 4 bits = 0 (0000)
	//   monoMixdownPresent: 1 bit = 0
	//   stereoMixdownPresent: 1 bit = 0
	//   matrixMixdownIdxPresent: 1 bit = 0
	//   frontElement[0]: isCPE=1 (1), tagSelect=0 (0000) => 2 channels via CPE
	//   byte align (padding to reach byte boundary)
	//   commentFieldBytes: 8 bits = 0
	//
	// Bit layout (reading from MSB):
	// Bits 0-2:   GASpec = 000 (frameLenFlag=0, dependsOnCore=0, extensionFlag=0)
	// Bits 3-6:   element_instance_tag = 0101 = 5
	// Bits 7-8:   object_type = 01 = 1
	// Bits 9-12:  sf_index = 0100 = 4
	// Bits 13-16: numFrontChannelElements = 0001 = 1
	// Bits 17-20: numSideChannelElements = 0000 = 0
	// Bits 21-24: numBackChannelElements = 0000 = 0
	// Bits 25-26: numLFEChannelElements = 00 = 0
	// Bits 27-29: numAssocDataElements = 000 = 0
	// Bits 30-33: numValidCCElements = 0000 = 0
	// Bit 34:     monoMixdownPresent = 0
	// Bit 35:     stereoMixdownPresent = 0
	// Bit 36:     matrixMixdownIdxPresent = 0
	// Bit 37:     frontElement[0].isCPE = 1
	// Bits 38-41: frontElement[0].tagSelect = 0000 = 0
	// Bits 42-47: byte alignment padding (6 bits to reach byte 6)
	// Bits 48-55: commentFieldBytes = 00000000 = 0
	//
	// Byte 0: 000_0101_0 = 0x0A
	// Byte 1: 1_0100_000 = 0xA0
	// Byte 2: 1_0000_000 = 0x80
	// Byte 3: 0_00_000_00 = 0x00
	// Byte 4: 00_0_0_0_1_0 = 0x04
	// Byte 5: 000_00000 = 0x00 (padding + part of comment length)
	// Byte 6: 0 = 0x00 (rest of comment length)

	data := []byte{0x0A, 0xA0, 0x80, 0x00, 0x04, 0x00, 0x00}
	r := bits.NewReader(data)
	asc := &aac.AudioSpecificConfig{
		ChannelsConfiguration: 0, // Triggers PCE parsing
		ObjectTypeIndex:       2, // LC
	}

	pce, err := parseGASpecificConfig(r, asc)
	if err != nil {
		t.Fatalf("parseGASpecificConfig() error = %v", err)
	}

	// Verify PCE was parsed (non-nil)
	if pce == nil {
		t.Fatal("Expected non-nil PCE for channelConfig=0")
	}

	// Verify PCE fields
	if pce.ElementInstanceTag != 5 {
		t.Errorf("PCE.ElementInstanceTag = %d, want 5", pce.ElementInstanceTag)
	}
	if pce.ObjectType != 1 {
		t.Errorf("PCE.ObjectType = %d, want 1 (AAC Main)", pce.ObjectType)
	}
	if pce.SFIndex != 4 {
		t.Errorf("PCE.SFIndex = %d, want 4 (44100 Hz)", pce.SFIndex)
	}
	if pce.NumFrontChannelElements != 1 {
		t.Errorf("PCE.NumFrontChannelElements = %d, want 1", pce.NumFrontChannelElements)
	}
	if pce.NumSideChannelElements != 0 {
		t.Errorf("PCE.NumSideChannelElements = %d, want 0", pce.NumSideChannelElements)
	}
	if pce.NumBackChannelElements != 0 {
		t.Errorf("PCE.NumBackChannelElements = %d, want 0", pce.NumBackChannelElements)
	}
	if pce.NumLFEChannelElements != 0 {
		t.Errorf("PCE.NumLFEChannelElements = %d, want 0", pce.NumLFEChannelElements)
	}
	if !pce.FrontElementIsCPE[0] {
		t.Error("PCE.FrontElementIsCPE[0] = false, want true")
	}
	if pce.FrontElementTagSelect[0] != 0 {
		t.Errorf("PCE.FrontElementTagSelect[0] = %d, want 0", pce.FrontElementTagSelect[0])
	}
	// CPE adds 2 channels
	if pce.Channels != 2 {
		t.Errorf("PCE.Channels = %d, want 2 (1 CPE = 2 channels)", pce.Channels)
	}
	if pce.NumFrontChannels != 2 {
		t.Errorf("PCE.NumFrontChannels = %d, want 2", pce.NumFrontChannels)
	}
	if pce.CommentFieldBytes != 0 {
		t.Errorf("PCE.CommentFieldBytes = %d, want 0", pce.CommentFieldBytes)
	}
}

// TestParseGASpecificConfigERResilienceFlags verifies that resilience flags
// are correctly parsed for ER (Error Resilient) object types.
func TestParseGASpecificConfigERResilienceFlags(t *testing.T) {
	// Data layout for ER object type with extensionFlag=1:
	// Bits are read from MSB to LSB:
	// Bit 7 (MSB): frameLengthFlag = 0
	// Bit 6: dependsOnCoreCoder = 0
	// Bit 5: extensionFlag = 1
	// Since objectType >= 17 (ER), we read 3 resilience flags:
	// Bit 4: AACSectionDataResilienceFlag = 1
	// Bit 3: AACScalefactorDataResilienceFlag = 0
	// Bit 2: AACSpectralDataResilienceFlag = 1
	// Bit 1: extensionFlag3 = 0 (skipped)
	// Bit 0: unused
	//
	// Binary: 0011_0100 = 0x34

	data := []byte{0x34}
	r := bits.NewReader(data)
	asc := &aac.AudioSpecificConfig{
		ChannelsConfiguration: 2,  // Non-zero, no PCE
		ObjectTypeIndex:       17, // ER AAC LC
	}

	_, err := parseGASpecificConfig(r, asc)
	if err != nil {
		t.Fatalf("parseGASpecificConfig() error = %v", err)
	}

	// Verify GASpec fields
	if asc.FrameLengthFlag {
		t.Error("FrameLengthFlag = true, want false")
	}
	if asc.DependsOnCoreCoder {
		t.Error("DependsOnCoreCoder = true, want false")
	}
	if !asc.ExtensionFlag {
		t.Error("ExtensionFlag = false, want true")
	}

	// Verify resilience flags (the main purpose of this test)
	if !asc.AACSectionDataResilienceFlag {
		t.Error("AACSectionDataResilienceFlag = false, want true")
	}
	if asc.AACScalefactorDataResilienceFlag {
		t.Error("AACScalefactorDataResilienceFlag = true, want false")
	}
	if !asc.AACSpectralDataResilienceFlag {
		t.Error("AACSpectralDataResilienceFlag = false, want true")
	}
}
