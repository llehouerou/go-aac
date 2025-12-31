package filterbank

import (
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
