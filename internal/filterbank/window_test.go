// Package filterbank window_test.go tests window constants.
package filterbank

import (
	"fmt"
	"testing"
)

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

func TestGetLongWindow(t *testing.T) {
	tests := []struct {
		shape     int
		wantFirst float32
		wantLen   int
	}{
		{SineWindow, 0.00076699031874270449, 1024},
		{KBDWindow, 0.00029256153896361, 1024},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("shape=%d", tt.shape), func(t *testing.T) {
			w := GetLongWindow(tt.shape)
			if len(w) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(w), tt.wantLen)
			}
			// Check first value (approximately - full precision check in validation test)
			if w[0] < tt.wantFirst*0.999 || w[0] > tt.wantFirst*1.001 {
				t.Errorf("w[0] = %v, want ~%v", w[0], tt.wantFirst)
			}
		})
	}
}

func TestGetShortWindow(t *testing.T) {
	tests := []struct {
		shape   int
		wantLen int
	}{
		{SineWindow, 128},
		{KBDWindow, 128},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("shape=%d", tt.shape), func(t *testing.T) {
			w := GetShortWindow(tt.shape)
			if len(w) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(w), tt.wantLen)
			}
		})
	}
}

func TestGetLongWindow_InvalidShape(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid shape")
		}
	}()
	GetLongWindow(99) // Should panic
}

func TestGetShortWindow_InvalidShape(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid shape")
		}
	}()
	GetShortWindow(-1) // Should panic
}
