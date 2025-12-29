// internal/spectrum/ltp_test.go
package spectrum

import (
	"math"
	"testing"

	"github.com/llehouerou/go-aac"
)

func TestIsLTPObjectType(t *testing.T) {
	tests := []struct {
		name       string
		objectType aac.ObjectType
		want       bool
	}{
		{"LC is not LTP", aac.ObjectTypeLC, false},
		{"Main is not LTP", aac.ObjectTypeMain, false},
		{"LTP is LTP", aac.ObjectTypeLTP, true},
		{"ER_LTP is LTP", aac.ObjectTypeERLTP, true},
		{"LD is LTP", aac.ObjectTypeLD, true},
		{"SSR is not LTP", aac.ObjectTypeSSR, false},
		{"HE_AAC is not LTP", aac.ObjectTypeHEAAC, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLTPObjectType(tt.objectType)
			if got != tt.want {
				t.Errorf("IsLTPObjectType(%v) = %v, want %v", tt.objectType, got, tt.want)
			}
		})
	}
}

func TestLTPCodebook(t *testing.T) {
	// Verify codebook has correct values from FAAD2
	expected := []float64{
		0.570829,
		0.696616,
		0.813004,
		0.911304,
		0.984900,
		1.067894,
		1.194601,
		1.369533,
	}

	if len(ltpCodebook) != 8 {
		t.Fatalf("ltpCodebook length = %d, want 8", len(ltpCodebook))
	}

	for i, exp := range expected {
		if math.Abs(ltpCodebook[i]-exp) > 1e-6 {
			t.Errorf("ltpCodebook[%d] = %v, want %v", i, ltpCodebook[i], exp)
		}
	}
}

func TestRealToInt16(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  int16
	}{
		{"zero", 0.0, 0},
		{"positive small", 100.5, 101},   // rounds to nearest
		{"negative small", -100.5, -101}, // rounds to nearest (away from zero for .5)
		{"positive large", 32767.0, 32767},
		{"negative large", -32768.0, -32768},
		{"positive overflow", 40000.0, 32767},   // clamp
		{"negative overflow", -40000.0, -32768}, // clamp
		{"positive round down", 100.3, 100},
		{"negative round down", -100.3, -100},
		{"positive round up", 100.7, 101},
		{"negative round up", -100.7, -101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := realToInt16(tt.input)
			if got != tt.want {
				t.Errorf("realToInt16(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
