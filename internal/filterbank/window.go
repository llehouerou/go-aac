// Package filterbank window.go defines window shape and size constants.
package filterbank

// Window shape constants.
// These match FAAD2's indexing in filtbank.c:70-73.
//
// Ported from: ~/dev/faad2/libfaad/filtbank.c
const (
	// SineWindow is the sine window shape (index 0).
	SineWindow = 0

	// KBDWindow is the Kaiser-Bessel Derived window shape (index 1).
	KBDWindow = 1
)

// Window size constants for standard AAC (1024 samples per frame).
const (
	// LongWindowSize is the size of long windows (1024 samples).
	LongWindowSize = 1024

	// ShortWindowSize is the size of short windows (128 samples = 1024/8).
	ShortWindowSize = 128
)

// GetLongWindow returns the long window (1024 samples) for the given shape.
// shape must be SineWindow (0) or KBDWindow (1).
//
// Ported from: fb->long_window[window_shape] in ~/dev/faad2/libfaad/filtbank.c:197
func GetLongWindow(shape int) []float32 {
	switch shape {
	case SineWindow:
		return sineLong1024[:]
	case KBDWindow:
		return kbdLong1024[:]
	default:
		panic("invalid window shape")
	}
}

// GetShortWindow returns the short window (128 samples) for the given shape.
// shape must be SineWindow (0) or KBDWindow (1).
//
// Ported from: fb->short_window[window_shape] in ~/dev/faad2/libfaad/filtbank.c:199
func GetShortWindow(shape int) []float32 {
	switch shape {
	case SineWindow:
		return sineShort128[:]
	case KBDWindow:
		return kbdShort128[:]
	default:
		panic("invalid window shape")
	}
}
