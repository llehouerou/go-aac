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

func TestClip32(t *testing.T) {
	tests := []struct {
		name  string
		input float32
		want  int32
	}{
		// Normal range (input scaled by 65536)
		{"zero", 0.0, 0},
		{"positive", 1000000.5, 1000000},   // Rounds to even
		{"negative", -1000000.5, -1000000}, // Rounds to even

		// Edge cases at 32-bit boundaries
		{"max_boundary", 2147483647.0, 2147483647},
		{"min_boundary", -2147483648.0, -2147483648},

		// Clipping cases
		{"clip_positive", 3e9, 2147483647},
		{"clip_negative", -3e9, -2147483648},
		{"clip_max_float", 1e10, 2147483647},
		{"clip_min_float", -1e10, -2147483648},

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
			got := clip32(tt.input)
			if got != tt.want {
				t.Errorf("clip32(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestToPCM16Bit_Mono(t *testing.T) {
	// Single channel input
	input := [][]float32{
		{0.0, 100.0, -100.0, 32767.0, -32768.0, 40000.0, -40000.0},
	}
	channelMap := []uint8{0}

	output := make([]int16, 7)
	ToPCM16Bit(input, channelMap, 1, 7, false, false, output)

	expected := []int16{0, 100, -100, 32767, -32768, 32767, -32768}
	for i, want := range expected {
		if output[i] != want {
			t.Errorf("output[%d] = %d, want %d", i, output[i], want)
		}
	}
}

func TestToPCM16Bit_Stereo(t *testing.T) {
	// Two channel input
	input := [][]float32{
		{100.0, 200.0, 300.0},    // Left
		{-100.0, -200.0, -300.0}, // Right
	}
	channelMap := []uint8{0, 1}

	output := make([]int16, 6) // 3 samples * 2 channels
	ToPCM16Bit(input, channelMap, 2, 3, false, false, output)

	// Expected: L0, R0, L1, R1, L2, R2
	expected := []int16{100, -100, 200, -200, 300, -300}
	for i, want := range expected {
		if output[i] != want {
			t.Errorf("output[%d] = %d, want %d", i, output[i], want)
		}
	}
}

func TestToPCM16Bit_StereoUpMatrix(t *testing.T) {
	// Single channel input, upmixed to stereo
	input := [][]float32{
		{100.0, 200.0, 300.0},
	}
	channelMap := []uint8{0}

	output := make([]int16, 6) // 3 samples * 2 channels
	ToPCM16Bit(input, channelMap, 2, 3, false, true, output)

	// Expected: L0=R0, L1=R1, L2=R2 (mono duplicated to both channels)
	expected := []int16{100, 100, 200, 200, 300, 300}
	for i, want := range expected {
		if output[i] != want {
			t.Errorf("output[%d] = %d, want %d", i, output[i], want)
		}
	}
}

func TestGetSample_NoDownmix(t *testing.T) {
	input := [][]float32{
		{100.0, 200.0},
		{-100.0, -200.0},
	}
	channelMap := []uint8{0, 1}

	// Without downmix, just returns the requested channel
	got := getSample(input, 0, 0, false, channelMap)
	if got != 100.0 {
		t.Errorf("getSample(ch0, s0) = %v, want 100.0", got)
	}

	got = getSample(input, 1, 1, false, channelMap)
	if got != -200.0 {
		t.Errorf("getSample(ch1, s1) = %v, want -200.0", got)
	}
}

func TestGetSample_Downmix5_1ToStereo(t *testing.T) {
	// 5.1 channel layout: C, L, R, Ls, Rs (indices 0-4)
	// FAAD2 internal_channel order: [0]=C, [1]=L, [2]=R, [3]=Ls, [4]=Rs
	input := [][]float32{
		{1000.0}, // Center
		{500.0},  // Left
		{600.0},  // Right
		{200.0},  // Left Surround
		{300.0},  // Right Surround
	}
	channelMap := []uint8{0, 1, 2, 3, 4}

	// Left output = L + C*RSQRT2 + Ls*RSQRT2, all scaled by DM_MUL
	// Expected: DM_MUL * (500 + 1000*0.7071 + 200*0.7071) = 0.3204 * 1348.5 ~ 432.0
	gotL := getSample(input, 0, 0, true, channelMap)
	expectedL := DMMul * (input[1][0] + input[0][0]*RSQRT2 + input[3][0]*RSQRT2)
	if math.Abs(float64(gotL-expectedL)) > 0.01 {
		t.Errorf("getSample(ch0, downmix) = %v, want %v", gotL, expectedL)
	}

	// Right output = R + C*RSQRT2 + Rs*RSQRT2, all scaled by DM_MUL
	gotR := getSample(input, 1, 0, true, channelMap)
	expectedR := DMMul * (input[2][0] + input[0][0]*RSQRT2 + input[4][0]*RSQRT2)
	if math.Abs(float64(gotR-expectedR)) > 0.01 {
		t.Errorf("getSample(ch1, downmix) = %v, want %v", gotR, expectedR)
	}
}

func TestToPCM16Bit_Downmix(t *testing.T) {
	// 5.1 input: C, L, R, Ls, Rs (5 channels)
	input := [][]float32{
		{1000.0, 2000.0}, // Center
		{500.0, 1000.0},  // Left
		{600.0, 1200.0},  // Right
		{200.0, 400.0},   // Left Surround
		{300.0, 600.0},   // Right Surround
	}
	channelMap := []uint8{0, 1, 2, 3, 4}

	output := make([]int16, 4) // 2 samples * 2 channels
	ToPCM16Bit(input, channelMap, 2, 2, true, false, output)

	// Calculate expected left output for sample 0
	expectedL0 := DMMul * (input[1][0] + input[0][0]*RSQRT2 + input[3][0]*RSQRT2)
	// Calculate expected right output for sample 0
	expectedR0 := DMMul * (input[2][0] + input[0][0]*RSQRT2 + input[4][0]*RSQRT2)

	if output[0] != clip16(expectedL0) {
		t.Errorf("output[0] = %d, want %d", output[0], clip16(expectedL0))
	}
	if output[1] != clip16(expectedR0) {
		t.Errorf("output[1] = %d, want %d", output[1], clip16(expectedR0))
	}
}

func TestToPCM24Bit_Mono(t *testing.T) {
	// Input in 16-bit range, will be scaled to 24-bit
	input := [][]float32{
		{0.0, 100.0, -100.0, 32767.0, -32768.0},
	}
	channelMap := []uint8{0}

	output := make([]int32, 5)
	ToPCM24Bit(input, channelMap, 1, 5, false, false, output)

	// Values are scaled by 256
	expected := []int32{0, 25600, -25600, 8388352, -8388608}
	for i, want := range expected {
		if output[i] != want {
			t.Errorf("output[%d] = %d, want %d", i, output[i], want)
		}
	}
}

func TestToPCM24Bit_Stereo(t *testing.T) {
	input := [][]float32{
		{100.0, 200.0},
		{-100.0, -200.0},
	}
	channelMap := []uint8{0, 1}

	output := make([]int32, 4)
	ToPCM24Bit(input, channelMap, 2, 2, false, false, output)

	// L0, R0, L1, R1 scaled by 256
	expected := []int32{25600, -25600, 51200, -51200}
	for i, want := range expected {
		if output[i] != want {
			t.Errorf("output[%d] = %d, want %d", i, output[i], want)
		}
	}
}

func TestToPCM24Bit_StereoUpMatrix(t *testing.T) {
	// Single channel input, upmixed to stereo
	input := [][]float32{
		{100.0, 200.0, 300.0},
	}
	channelMap := []uint8{0}

	output := make([]int32, 6) // 3 samples * 2 channels
	ToPCM24Bit(input, channelMap, 2, 3, false, true, output)

	// Expected: L0=R0, L1=R1, L2=R2 (mono duplicated to both channels, scaled by 256)
	expected := []int32{25600, 25600, 51200, 51200, 76800, 76800}
	for i, want := range expected {
		if output[i] != want {
			t.Errorf("output[%d] = %d, want %d", i, output[i], want)
		}
	}
}

func TestToPCM24Bit_Clipping(t *testing.T) {
	// Test clipping at 24-bit boundaries
	input := [][]float32{
		{40000.0, -40000.0}, // Will exceed 24-bit range when scaled by 256
	}
	channelMap := []uint8{0}

	output := make([]int32, 2)
	ToPCM24Bit(input, channelMap, 1, 2, false, false, output)

	// 40000 * 256 = 10,240,000 > 8388607, so clips to max
	// -40000 * 256 = -10,240,000 < -8388608, so clips to min
	expected := []int32{8388607, -8388608}
	for i, want := range expected {
		if output[i] != want {
			t.Errorf("output[%d] = %d, want %d", i, output[i], want)
		}
	}
}

func TestToPCM24Bit_Downmix(t *testing.T) {
	// 5.1 input: C, L, R, Ls, Rs (5 channels)
	input := [][]float32{
		{1000.0, 2000.0}, // Center
		{500.0, 1000.0},  // Left
		{600.0, 1200.0},  // Right
		{200.0, 400.0},   // Left Surround
		{300.0, 600.0},   // Right Surround
	}
	channelMap := []uint8{0, 1, 2, 3, 4}

	output := make([]int32, 4) // 2 samples * 2 channels
	ToPCM24Bit(input, channelMap, 2, 2, true, false, output)

	// Calculate expected left output for sample 0, scaled by 256
	expectedL0 := DMMul * (input[1][0] + input[0][0]*RSQRT2 + input[3][0]*RSQRT2) * 256
	// Calculate expected right output for sample 0, scaled by 256
	expectedR0 := DMMul * (input[2][0] + input[0][0]*RSQRT2 + input[4][0]*RSQRT2) * 256

	if output[0] != clip24(expectedL0) {
		t.Errorf("output[0] = %d, want %d", output[0], clip24(expectedL0))
	}
	if output[1] != clip24(expectedR0) {
		t.Errorf("output[1] = %d, want %d", output[1], clip24(expectedR0))
	}
}

func TestToPCM32Bit_Mono(t *testing.T) {
	// Input in 16-bit range, will be scaled to 32-bit
	input := [][]float32{
		{0.0, 100.0, -100.0, 32767.0, -32768.0},
	}
	channelMap := []uint8{0}

	output := make([]int32, 5)
	ToPCM32Bit(input, channelMap, 1, 5, false, false, output)

	// Values are scaled by 65536
	expected := []int32{0, 6553600, -6553600, 2147418112, -2147483648}
	for i, want := range expected {
		if output[i] != want {
			t.Errorf("output[%d] = %d, want %d", i, output[i], want)
		}
	}
}

func TestToPCM32Bit_Stereo(t *testing.T) {
	input := [][]float32{
		{100.0, 200.0},
		{-100.0, -200.0},
	}
	channelMap := []uint8{0, 1}

	output := make([]int32, 4)
	ToPCM32Bit(input, channelMap, 2, 2, false, false, output)

	// L0, R0, L1, R1 scaled by 65536
	expected := []int32{6553600, -6553600, 13107200, -13107200}
	for i, want := range expected {
		if output[i] != want {
			t.Errorf("output[%d] = %d, want %d", i, output[i], want)
		}
	}
}

func TestToPCM32Bit_StereoUpMatrix(t *testing.T) {
	// Single channel input, upmixed to stereo
	input := [][]float32{
		{100.0, 200.0, 300.0},
	}
	channelMap := []uint8{0}

	output := make([]int32, 6) // 3 samples * 2 channels
	ToPCM32Bit(input, channelMap, 2, 3, false, true, output)

	// Expected: L0=R0, L1=R1, L2=R2 (mono duplicated to both channels, scaled by 65536)
	expected := []int32{6553600, 6553600, 13107200, 13107200, 19660800, 19660800}
	for i, want := range expected {
		if output[i] != want {
			t.Errorf("output[%d] = %d, want %d", i, output[i], want)
		}
	}
}

func TestToPCM32Bit_Clipping(t *testing.T) {
	// Test clipping at 32-bit boundaries
	input := [][]float32{
		{40000.0, -40000.0}, // Exceeds 16-bit range, will clip at 32-bit
	}
	channelMap := []uint8{0}

	output := make([]int32, 2)
	ToPCM32Bit(input, channelMap, 1, 2, false, false, output)

	// 40000 * 65536 = 2621440000 > 2147483647, clips to max
	// -40000 * 65536 = -2621440000 < -2147483648, clips to min
	expected := []int32{2147483647, -2147483648}
	for i, want := range expected {
		if output[i] != want {
			t.Errorf("output[%d] = %d, want %d", i, output[i], want)
		}
	}
}

func TestToPCM32Bit_Downmix(t *testing.T) {
	// 5.1 input: C, L, R, Ls, Rs (5 channels)
	input := [][]float32{
		{1000.0, 2000.0}, // Center
		{500.0, 1000.0},  // Left
		{600.0, 1200.0},  // Right
		{200.0, 400.0},   // Left Surround
		{300.0, 600.0},   // Right Surround
	}
	channelMap := []uint8{0, 1, 2, 3, 4}

	output := make([]int32, 4) // 2 samples * 2 channels
	ToPCM32Bit(input, channelMap, 2, 2, true, false, output)

	// Calculate expected left output for sample 0, scaled by 65536
	expectedL0 := DMMul * (input[1][0] + input[0][0]*RSQRT2 + input[3][0]*RSQRT2) * 65536
	// Calculate expected right output for sample 0, scaled by 65536
	expectedR0 := DMMul * (input[2][0] + input[0][0]*RSQRT2 + input[4][0]*RSQRT2) * 65536

	if output[0] != clip32(expectedL0) {
		t.Errorf("output[0] = %d, want %d", output[0], clip32(expectedL0))
	}
	if output[1] != clip32(expectedR0) {
		t.Errorf("output[1] = %d, want %d", output[1], clip32(expectedR0))
	}
}

func TestToPCMFloat_Mono(t *testing.T) {
	// Input in 16-bit range, will be normalized to [-1.0, 1.0]
	input := [][]float32{
		{0.0, 32768.0, -32768.0},
	}
	channelMap := []uint8{0}

	output := make([]float32, 3)
	ToPCMFloat(input, channelMap, 1, 3, false, false, output)

	// Values scaled by FloatScale = 1/32768
	expected := []float32{0.0, 1.0, -1.0}
	for i, want := range expected {
		if math.Abs(float64(output[i]-want)) > 1e-6 {
			t.Errorf("output[%d] = %v, want %v", i, output[i], want)
		}
	}
}

func TestToPCMFloat_Stereo(t *testing.T) {
	input := [][]float32{
		{16384.0, 32768.0}, // 0.5, 1.0 after scaling
		{-16384.0, -32768.0},
	}
	channelMap := []uint8{0, 1}

	output := make([]float32, 4)
	ToPCMFloat(input, channelMap, 2, 2, false, false, output)

	expected := []float32{0.5, -0.5, 1.0, -1.0}
	for i, want := range expected {
		if math.Abs(float64(output[i]-want)) > 1e-6 {
			t.Errorf("output[%d] = %v, want %v", i, output[i], want)
		}
	}
}

func TestToPCMFloat_StereoUpMatrix(t *testing.T) {
	// Single channel input, upmixed to stereo
	input := [][]float32{
		{16384.0, 32768.0}, // 0.5, 1.0 after scaling
	}
	channelMap := []uint8{0}

	output := make([]float32, 4) // 2 samples * 2 channels
	ToPCMFloat(input, channelMap, 2, 2, false, true, output)

	// Expected: L0=R0, L1=R1 (mono duplicated to both channels)
	expected := []float32{0.5, 0.5, 1.0, 1.0}
	for i, want := range expected {
		if math.Abs(float64(output[i]-want)) > 1e-6 {
			t.Errorf("output[%d] = %v, want %v", i, output[i], want)
		}
	}
}

func TestToPCMFloat_Downmix(t *testing.T) {
	// 5.1 input: C, L, R, Ls, Rs (5 channels)
	input := [][]float32{
		{1000.0, 2000.0}, // Center
		{500.0, 1000.0},  // Left
		{600.0, 1200.0},  // Right
		{200.0, 400.0},   // Left Surround
		{300.0, 600.0},   // Right Surround
	}
	channelMap := []uint8{0, 1, 2, 3, 4}

	output := make([]float32, 4) // 2 samples * 2 channels
	ToPCMFloat(input, channelMap, 2, 2, true, false, output)

	// Calculate expected left output for sample 0, scaled by FloatScale
	expectedL0 := DMMul * (input[1][0] + input[0][0]*RSQRT2 + input[3][0]*RSQRT2) * FloatScale
	// Calculate expected right output for sample 0, scaled by FloatScale
	expectedR0 := DMMul * (input[2][0] + input[0][0]*RSQRT2 + input[4][0]*RSQRT2) * FloatScale

	if math.Abs(float64(output[0]-expectedL0)) > 1e-6 {
		t.Errorf("output[0] = %v, want %v", output[0], expectedL0)
	}
	if math.Abs(float64(output[1]-expectedR0)) > 1e-6 {
		t.Errorf("output[1] = %v, want %v", output[1], expectedR0)
	}
}

func TestToPCMFloat_NoClipping(t *testing.T) {
	// Float output doesn't clip - values can exceed [-1.0, 1.0]
	input := [][]float32{
		{65536.0, -65536.0}, // 2.0, -2.0 after scaling (exceeds normalized range)
	}
	channelMap := []uint8{0}

	output := make([]float32, 2)
	ToPCMFloat(input, channelMap, 1, 2, false, false, output)

	// Values should be 2.0 and -2.0, not clipped
	expected := []float32{2.0, -2.0}
	for i, want := range expected {
		if math.Abs(float64(output[i]-want)) > 1e-6 {
			t.Errorf("output[%d] = %v, want %v", i, output[i], want)
		}
	}
}
