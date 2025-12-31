// Package filterbank window_test.go tests window constants.
package filterbank

import "testing"

func TestWindowShapeConstants(t *testing.T) {
	// Window shapes must match FAAD2 indices (filtbank.c:70-73)
	if SineWindow != 0 {
		t.Errorf("SineWindow = %d, want 0", SineWindow)
	}
	if KBDWindow != 1 {
		t.Errorf("KBDWindow = %d, want 1", KBDWindow)
	}
}

func TestWindowSizeConstants(t *testing.T) {
	// Standard AAC frame uses 1024 long, 128 short
	if LongWindowSize != 1024 {
		t.Errorf("LongWindowSize = %d, want 1024", LongWindowSize)
	}
	if ShortWindowSize != 128 {
		t.Errorf("ShortWindowSize = %d, want 128", ShortWindowSize)
	}
}
