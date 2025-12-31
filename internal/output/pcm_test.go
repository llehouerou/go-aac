// Package output provides PCM output conversion.
// Ported from: ~/dev/faad2/libfaad/output.c
package output

import (
	"math"
	"testing"
)

func TestPCMConstants(t *testing.T) {
	// FLOAT_SCALE = 1.0 / (1 << 15) = 1/32768
	// Source: ~/dev/faad2/libfaad/output.c:39
	expectedFloatScale := float32(1.0 / 32768.0)
	if math.Abs(float64(FloatScale-expectedFloatScale)) > 1e-10 {
		t.Errorf("FloatScale: got %v, want %v", FloatScale, expectedFloatScale)
	}

	// DM_MUL = 1/(1+sqrt(2)+1/sqrt(2)) ≈ 0.3203772410170407
	// Source: ~/dev/faad2/libfaad/output.c:41
	expectedDMMul := float32(0.3203772410170407)
	if math.Abs(float64(DMMul-expectedDMMul)) > 1e-6 {
		t.Errorf("DMMul: got %v, want %v", DMMul, expectedDMMul)
	}

	// RSQRT2 = 1/sqrt(2) ≈ 0.7071067811865475
	// Source: ~/dev/faad2/libfaad/output.c:42
	expectedRSQRT2 := float32(0.7071067811865475244)
	if math.Abs(float64(RSQRT2-expectedRSQRT2)) > 1e-6 {
		t.Errorf("RSQRT2: got %v, want %v", RSQRT2, expectedRSQRT2)
	}
}

func TestClip16(t *testing.T) {
	tests := []struct {
		name  string
		input float32
		want  int16
	}{
		// Normal range
		{"zero", 0.0, 0},
		{"positive", 100.5, 100},   // Rounds to nearest even (100 is even)
		{"negative", -100.5, -100}, // Rounds to nearest even (-100 is even)

		// Edge cases at boundaries
		{"max_boundary", 32767.0, 32767},
		{"min_boundary", -32768.0, -32768},

		// Clipping cases
		{"clip_positive", 40000.0, 32767},
		{"clip_negative", -40000.0, -32768},
		{"clip_max_float", 1e10, 32767},
		{"clip_min_float", -1e10, -32768},

		// Rounding behavior (matches lrintf: round to nearest, ties to even)
		{"round_up", 0.6, 1},
		{"round_down", 0.4, 0},
		{"round_half_even_up", 1.5, 2},   // 1.5 -> 2 (nearest even)
		{"round_half_even_down", 2.5, 2}, // 2.5 -> 2 (nearest even)
		{"round_neg_up", -0.4, 0},
		{"round_neg_down", -0.6, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clip16(tt.input)
			if got != tt.want {
				t.Errorf("clip16(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestClip24(t *testing.T) {
	tests := []struct {
		name  string
		input float32
		want  int32
	}{
		// Normal range (input is already scaled by 256)
		{"zero", 0.0, 0},
		{"positive", 256000.5, 256000}, // Note: 256000.5 rounds to 256000 (even)
		{"negative", -256000.5, -256000},

		// Edge cases at 24-bit boundaries
		{"max_boundary", 8388607.0, 8388607},
		{"min_boundary", -8388608.0, -8388608},

		// Clipping cases
		{"clip_positive", 10000000.0, 8388607},
		{"clip_negative", -10000000.0, -8388608},
		{"clip_max_float", 1e10, 8388607},
		{"clip_min_float", -1e10, -8388608},

		// Rounding behavior (matches lrintf: round to nearest, ties to even)
		{"round_up", 0.6, 1},
		{"round_down", 0.4, 0},
		{"round_half_even_up", 1.5, 2},   // 1.5 -> 2 (nearest even)
		{"round_half_even_down", 2.5, 2}, // 2.5 -> 2 (nearest even)
		{"round_neg_up", -0.4, 0},
		{"round_neg_down", -0.6, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clip24(tt.input)
			if got != tt.want {
				t.Errorf("clip24(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
