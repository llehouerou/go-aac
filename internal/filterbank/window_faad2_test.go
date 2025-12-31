// Package filterbank window_faad2_test.go validates window values against FAAD2.
package filterbank

import (
	"math"
	"testing"
)

// TestWindowValues_MatchFAAD2 validates that our window values match FAAD2 exactly.
// This is critical because windows affect the final audio output.
func TestWindowValues_MatchFAAD2(t *testing.T) {
	// Test sine long window - first and last 5 values
	sineLongExpected := []float32{
		0.00076699031874270449,
		0.002300969151425805,
		0.0038349425697062275,
		0.0053689069639963425,
		0.0069028587247297558,
	}

	w := GetLongWindow(SineWindow)
	for i, expected := range sineLongExpected {
		if !closeEnough(w[i], expected, 1e-10) {
			t.Errorf("sineLong1024[%d] = %.20f, want %.20f", i, w[i], expected)
		}
	}

	// Test KBD long window - first 5 values
	kbdLongExpected := []float32{
		0.00029256153896361,
		0.00042998567353047,
		0.00054674074589540,
		0.00065482304299792,
		0.00075870195068747,
	}

	w = GetLongWindow(KBDWindow)
	for i, expected := range kbdLongExpected {
		if !closeEnough(w[i], expected, 1e-10) {
			t.Errorf("kbdLong1024[%d] = %.20f, want %.20f", i, w[i], expected)
		}
	}

	// Verify window lengths
	if len(GetLongWindow(SineWindow)) != LongWindowSize {
		t.Errorf("sineLong1024 length = %d, want %d", len(GetLongWindow(SineWindow)), LongWindowSize)
	}
	if len(GetShortWindow(SineWindow)) != ShortWindowSize {
		t.Errorf("sineShort128 length = %d, want %d", len(GetShortWindow(SineWindow)), ShortWindowSize)
	}
}

// TestWindowTDAC verifies that windows satisfy the TDAC (Time-Domain Aliasing
// Cancellation) property required for perfect reconstruction in MDCT.
// The property is: w[n]^2 + w[N-1-n]^2 = 1 for all n in [0, N/2).
// This is different from simple symmetry - these windows are designed so that
// overlapping windows sum to constant power.
func TestWindowTDAC(t *testing.T) {
	tests := []struct {
		name  string
		shape int
		size  int
	}{
		{"sine_long", SineWindow, LongWindowSize},
		{"sine_short", SineWindow, ShortWindowSize},
		{"kbd_long", KBDWindow, LongWindowSize},
		{"kbd_short", KBDWindow, ShortWindowSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w []float32
			if tt.size == LongWindowSize {
				w = GetLongWindow(tt.shape)
			} else {
				w = GetShortWindow(tt.shape)
			}

			n := len(w)
			for i := 0; i < n/2; i++ {
				// TDAC condition: w[n]^2 + w[N-1-n]^2 = 1
				sum := float64(w[i])*float64(w[i]) + float64(w[n-1-i])*float64(w[n-1-i])
				if math.Abs(sum-1.0) > 1e-5 {
					t.Errorf("TDAC violation at [%d]: w[%d]^2 + w[%d]^2 = %v, want 1.0",
						i, i, n-1-i, sum)
				}
			}
		})
	}
}

// TestWindowRange verifies all window values are in valid range [0, 1].
func TestWindowRange(t *testing.T) {
	windows := []struct {
		name string
		w    []float32
	}{
		{"sine_long", GetLongWindow(SineWindow)},
		{"sine_short", GetShortWindow(SineWindow)},
		{"kbd_long", GetLongWindow(KBDWindow)},
		{"kbd_short", GetShortWindow(KBDWindow)},
	}

	for _, ww := range windows {
		t.Run(ww.name, func(t *testing.T) {
			for i, v := range ww.w {
				if v < 0 || v > 1 {
					t.Errorf("[%d] = %v, want in range [0, 1]", i, v)
				}
			}
		})
	}
}

func closeEnough(a, b float32, tolerance float64) bool {
	return math.Abs(float64(a-b)) < tolerance
}
