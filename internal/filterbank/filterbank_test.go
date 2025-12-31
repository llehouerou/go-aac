package filterbank

import (
	"math"
	"testing"

	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestNewFilterBank(t *testing.T) {
	fb := NewFilterBank(1024)
	if fb == nil {
		t.Fatal("expected non-nil FilterBank")
	}
	if fb.mdct256 == nil {
		t.Error("expected mdct256 to be initialized")
	}
	if fb.mdct2048 == nil {
		t.Error("expected mdct2048 to be initialized")
	}
}

func TestIFilterBank_OnlyLongSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	// Create test input (1024 frequency coefficients)
	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i % 100) // Simple pattern
	}

	// Output buffer (1024 time samples)
	timeOut := make([]float32, 1024)

	// Overlap buffer (1024 samples, starts at zero)
	overlap := make([]float32, 1024)

	// Process one frame
	fb.IFilterBank(
		syntax.OnlyLongSequence,
		SineWindow, // window_shape
		SineWindow, // window_shape_prev
		freqIn,
		timeOut,
		overlap,
	)

	// After processing, overlap should contain non-zero values
	// (the second half of the windowed IMDCT output)
	allZero := true
	for _, v := range overlap {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("overlap buffer should contain non-zero values after processing")
	}
}

func TestIFilterBank_LongStartSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i % 100)
	}

	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)

	// First, process with ONLY_LONG to initialize overlap
	fb.IFilterBank(syntax.OnlyLongSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// Now test LONG_START_SEQUENCE
	fb.IFilterBank(syntax.LongStartSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// The overlap structure for LONG_START should have:
	// - First nflat_ls samples: direct values (no windowing)
	// - Next nshort samples: windowed with short window
	// - Last nflat_ls samples: zeros
	nshort := 1024 / 8              // 128
	nflat_ls := (1024 - nshort) / 2 // 448

	// Check that the end section is zeros
	for i := nflat_ls + nshort; i < 1024; i++ {
		if overlap[i] != 0 {
			t.Errorf("overlap[%d] = %f, expected 0 (zeros region)", i, overlap[i])
			break
		}
	}
}

func TestIFilterBank_LongStopSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i % 100)
	}

	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)

	// Initialize overlap as if coming from short blocks
	nshort := 1024 / 8              // 128
	nflat_ls := (1024 - nshort) / 2 // 448
	for i := 0; i < nflat_ls; i++ {
		overlap[i] = 0 // zeros before short window region
	}
	for i := nflat_ls; i < nflat_ls+nshort; i++ {
		overlap[i] = float32(i) // some values in short window region
	}
	for i := nflat_ls + nshort; i < 1024; i++ {
		overlap[i] = float32(i) // values in flat region after short
	}

	fb.IFilterBank(syntax.LongStopSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// After LONG_STOP, the overlap should be full long window style
	allZero := true
	for _, v := range overlap {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("overlap should have non-zero values")
	}
}

func TestIFilterBank_EightShortSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	// Input is 1024 coefficients, but treated as 8x128 short blocks
	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i % 50)
	}

	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)

	// Set up overlap as if coming from LONG_START
	nshort := 1024 / 8              // 128
	nflat_ls := (1024 - nshort) / 2 // 448
	for i := 0; i < nflat_ls; i++ {
		overlap[i] = float32(i)
	}
	for i := nflat_ls; i < 1024; i++ {
		overlap[i] = float32(i % 100)
	}

	fb.IFilterBank(syntax.EightShortSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// timeOut should have valid data
	allZero := true
	for _, v := range timeOut {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("timeOut should have non-zero values after EIGHT_SHORT_SEQUENCE")
	}
}

func TestIFilterBank_WindowTransitionLongToShort(t *testing.T) {
	fb := NewFilterBank(1024)

	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i%100) * 0.01
	}

	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)

	// Simulate: ONLY_LONG -> LONG_START -> EIGHT_SHORT

	// Frame 1: ONLY_LONG
	fb.IFilterBank(syntax.OnlyLongSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// Frame 2: LONG_START (transition to short)
	fb.IFilterBank(syntax.LongStartSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// Frame 3: EIGHT_SHORT
	fb.IFilterBank(syntax.EightShortSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// Verify no panics and output is reasonable
	if len(timeOut) != 1024 {
		t.Errorf("timeOut length = %d, expected 1024", len(timeOut))
	}
}

func TestIFilterBank_WindowTransitionShortToLong(t *testing.T) {
	fb := NewFilterBank(1024)

	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i%100) * 0.01
	}

	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)

	// Simulate: ONLY_LONG -> LONG_START -> EIGHT_SHORT -> LONG_STOP -> ONLY_LONG

	fb.IFilterBank(syntax.OnlyLongSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.LongStartSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.EightShortSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.LongStopSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.OnlyLongSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// Verify no panics and output is reasonable
	if len(timeOut) != 1024 {
		t.Errorf("timeOut length = %d, expected 1024", len(timeOut))
	}
}

func TestIFilterBank_MixedWindowShapes(t *testing.T) {
	fb := NewFilterBank(1024)

	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i%100) * 0.01
	}

	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)

	// Test transitioning between sine and KBD windows
	fb.IFilterBank(syntax.OnlyLongSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.OnlyLongSequence, KBDWindow, SineWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.OnlyLongSequence, KBDWindow, KBDWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.OnlyLongSequence, SineWindow, KBDWindow, freqIn, timeOut, overlap)

	// Verify no panics
	if len(timeOut) != 1024 {
		t.Errorf("timeOut length = %d, expected 1024", len(timeOut))
	}
}

func TestFilterBankLTP_OnlyLongSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	// Input: 2*frameLength time samples (2048)
	inData := make([]float32, 2048)
	for i := range inData {
		inData[i] = float32(i%100) * 0.01
	}

	// Output: frameLength MDCT coefficients (1024)
	outMDCT := make([]float32, 1024)

	// Call FilterBankLTP
	fb.FilterBankLTP(
		syntax.OnlyLongSequence,
		SineWindow, // window_shape
		SineWindow, // window_shape_prev
		inData,
		outMDCT,
	)

	// Output should not be all zeros
	allZero := true
	for _, v := range outMDCT {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("outMDCT should contain non-zero values after FilterBankLTP")
	}

	// Verify no NaN or Inf values
	for i, v := range outMDCT {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Errorf("outMDCT[%d] = %v (invalid)", i, v)
		}
	}
}

func TestFilterBankLTP_LongStartSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	inData := make([]float32, 2048)
	for i := range inData {
		inData[i] = float32(i%100) * 0.01
	}

	outMDCT := make([]float32, 1024)

	// Should not panic
	fb.FilterBankLTP(
		syntax.LongStartSequence,
		SineWindow,
		SineWindow,
		inData,
		outMDCT,
	)

	// Output should not be all zeros
	allZero := true
	for _, v := range outMDCT {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("outMDCT should contain non-zero values")
	}

	// Verify no NaN/Inf
	for i, v := range outMDCT {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Errorf("outMDCT[%d] = %v (invalid)", i, v)
		}
	}
}
