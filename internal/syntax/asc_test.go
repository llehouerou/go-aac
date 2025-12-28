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
