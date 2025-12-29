// internal/spectrum/ltp_test.go
package spectrum

import (
	"math"
	"testing"

	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/syntax"
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

func TestLTPUpdateState_NonLD(t *testing.T) {
	// Test state update for non-LD object types (LC, LTP, etc.)
	frameLen := uint16(8) // Small for testing

	// State buffer: 4*frameLen = 32 samples
	// Layout: [old_half | time_samples | overlap_samples | zeros]
	state := make([]int16, 4*frameLen)

	// Initialize with some values
	for i := range state {
		state[i] = int16(i + 100)
	}

	// Time domain samples (current frame output)
	time := make([]float64, frameLen)
	for i := range time {
		time[i] = float64(i * 10)
	}

	// Overlap samples from filter bank
	overlap := make([]float64, frameLen)
	for i := range overlap {
		overlap[i] = float64(i * 20)
	}

	LTPUpdateState(state, time, overlap, frameLen, aac.ObjectTypeLTP)

	// Expected layout after update:
	// [0..7] = old state[8..15]
	// [8..15] = realToInt16(time[0..7])
	// [16..23] = realToInt16(overlap[0..7])
	// [24..31] = unchanged (zeros initialized at start)

	// Check shifted values
	for i := uint16(0); i < frameLen; i++ {
		expected := int16(i + 100 + 8) // Original state[i+8]
		if state[i] != expected {
			t.Errorf("state[%d] = %d, want %d (shifted)", i, state[i], expected)
		}
	}

	// Check time values
	for i := uint16(0); i < frameLen; i++ {
		expected := realToInt16(float64(i * 10))
		if state[frameLen+i] != expected {
			t.Errorf("state[%d] = %d, want %d (time)", frameLen+i, state[frameLen+i], expected)
		}
	}

	// Check overlap values
	for i := uint16(0); i < frameLen; i++ {
		expected := realToInt16(float64(i * 20))
		if state[2*frameLen+i] != expected {
			t.Errorf("state[%d] = %d, want %d (overlap)", 2*frameLen+i, state[2*frameLen+i], expected)
		}
	}
}

func TestLTPUpdateState_LD(t *testing.T) {
	// Test state update for LD object type (extra 512 lookback)
	frameLen := uint16(8) // Small for testing

	// State buffer: 4*frameLen = 32 samples
	state := make([]int16, 4*frameLen)

	// Initialize with some values
	for i := range state {
		state[i] = int16(i + 100)
	}

	// Time domain samples
	time := make([]float64, frameLen)
	for i := range time {
		time[i] = float64(i * 10)
	}

	// Overlap samples
	overlap := make([]float64, frameLen)
	for i := range overlap {
		overlap[i] = float64(i * 20)
	}

	LTPUpdateState(state, time, overlap, frameLen, aac.ObjectTypeLD)

	// Expected layout after update (LD mode):
	// [0..7] = old state[8..15]
	// [8..15] = old state[16..23]
	// [16..23] = realToInt16(time[0..7])
	// [24..31] = realToInt16(overlap[0..7])

	// Check first shift
	for i := uint16(0); i < frameLen; i++ {
		expected := int16(i + 100 + 8) // Original state[i+8]
		if state[i] != expected {
			t.Errorf("state[%d] = %d, want %d (first shift)", i, state[i], expected)
		}
	}

	// Check second shift
	for i := uint16(0); i < frameLen; i++ {
		expected := int16(i + 100 + 16) // Original state[i+16]
		if state[frameLen+i] != expected {
			t.Errorf("state[%d] = %d, want %d (second shift)", frameLen+i, state[frameLen+i], expected)
		}
	}

	// Check time values
	for i := uint16(0); i < frameLen; i++ {
		expected := realToInt16(float64(i * 10))
		if state[2*frameLen+i] != expected {
			t.Errorf("state[%d] = %d, want %d (time)", 2*frameLen+i, state[2*frameLen+i], expected)
		}
	}

	// Check overlap values
	for i := uint16(0); i < frameLen; i++ {
		expected := realToInt16(float64(i * 20))
		if state[3*frameLen+i] != expected {
			t.Errorf("state[%d] = %d, want %d (overlap)", 3*frameLen+i, state[3*frameLen+i], expected)
		}
	}
}

func TestLTPPrediction_NoDataPresent(t *testing.T) {
	frameLen := uint16(1024)
	spec := make([]float64, frameLen)
	for i := range spec {
		spec[i] = float64(i)
	}
	original := make([]float64, len(spec))
	copy(original, spec)

	ics := &syntax.ICStream{
		WindowSequence: syntax.OnlyLongSequence,
	}

	ltp := &syntax.LTPInfo{
		DataPresent: false,
	}

	cfg := &LTPConfig{
		ICS:         ics,
		LTP:         ltp,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLTP,
		FrameLength: frameLen,
		// FilterBank is nil - won't be called since DataPresent is false
	}

	LTPPrediction(spec, nil, cfg)

	// No LTP data - spectrum should be unchanged
	for i := range spec {
		if spec[i] != original[i] {
			t.Errorf("sample %d modified without LTP data: got %v, want %v", i, spec[i], original[i])
		}
	}
}

func TestLTPPrediction_ShortBlocks(t *testing.T) {
	frameLen := uint16(1024)
	spec := make([]float64, frameLen)
	for i := range spec {
		spec[i] = float64(i)
	}
	original := make([]float64, len(spec))
	copy(original, spec)

	ics := &syntax.ICStream{
		WindowSequence: syntax.EightShortSequence, // LTP not applied to short blocks
	}

	ltp := &syntax.LTPInfo{
		DataPresent: true,
		Lag:         100,
		Coef:        3,
	}

	cfg := &LTPConfig{
		ICS:         ics,
		LTP:         ltp,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLTP,
		FrameLength: frameLen,
	}

	LTPPrediction(spec, nil, cfg)

	// Short blocks - spectrum should be unchanged
	for i := range spec {
		if spec[i] != original[i] {
			t.Errorf("sample %d modified with short blocks: got %v, want %v", i, spec[i], original[i])
		}
	}
}
