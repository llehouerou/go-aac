package syntax

import (
	"testing"
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
		t.Run("type_"+string(rune('0'+objType/100))+string(rune('0'+(objType/10)%10))+string(rune('0'+objType%10)), func(t *testing.T) {
			got := isObjectTypeSupported(objType)
			if got {
				t.Errorf("isObjectTypeSupported(%d) = true, want false for out-of-range type", objType)
			}
		})
	}
}
