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
