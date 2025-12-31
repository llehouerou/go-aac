# PCM Output Conversion Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Convert decoded spectral float samples to PCM output in various formats (16-bit, 24-bit, 32-bit, float, double)

**Architecture:** The PCM converter takes per-channel float32 arrays from the filter bank and produces interleaved PCM samples in the requested format. It handles channel remapping, downmixing (5.1 to stereo), upmixing (mono to stereo), proper clipping, and rounding.

**Tech Stack:** Go standard library only (math, encoding/binary for tests)

---

## Background

### FAAD2 Source Analysis

The PCM output conversion is in `~/dev/faad2/libfaad/output.c` (~563 lines):

```c
// Key constants
#define FLOAT_SCALE (1.0f/(1<<15))  // 1/32768 for normalizing to [-1.0, 1.0]
#define DM_MUL REAL_CONST(0.3203772410170407)   // 1/(1+sqrt(2)+1/sqrt(2))
#define RSQRT2 REAL_CONST(0.7071067811865475244) // 1/sqrt(2)

// Main entry point
void *output_to_PCM(NeAACDecStruct *hDecoder, real_t **input, void *sample_buffer,
                    uint8_t channels, uint16_t frame_len, uint8_t format);

// Per-format converters
static void to_PCM_16bit(...);   // Clips to [-32768, 32767], rounds with lrintf
static void to_PCM_24bit(...);   // Scales by 256, clips to [-8388608, 8388607]
static void to_PCM_32bit(...);   // Scales by 65536, clips to [-2147483648, 2147483647]
static void to_PCM_float(...);   // Scales by FLOAT_SCALE (no clipping)
static void to_PCM_double(...);  // Same as float but double precision
```

### Input Assumptions

- Input: `[][]float32` - Per-channel float32 samples from filter bank
- Values are in range roughly [-32768, 32767] (16-bit equivalent)
- Frame length: typically 1024 (AAC-LC) or 2048 (SBR upsampled)

### Output Behavior

1. **16-bit**: Clip + round to int16, interleave channels
2. **24-bit**: Scale by 256, clip to 24-bit range, store as int32
3. **32-bit**: Scale by 65536, clip to 32-bit range, store as int32
4. **Float**: Scale by 1/32768, output as float32
5. **Double**: Scale by 1/32768, output as float64

---

## Task 1: Add PCM Constants

**Files:**
- Create: `internal/output/pcm.go`
- Test: `internal/output/pcm_test.go`

**Step 1: Write the test for constants**

```go
// internal/output/pcm_test.go
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
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL with "FloatScale not defined"

**Step 3: Write minimal implementation**

```go
// internal/output/pcm.go
package output

// PCM conversion constants.
// Ported from: ~/dev/faad2/libfaad/output.c:39-42

// FloatScale normalizes 16-bit range to [-1.0, 1.0].
// FLOAT_SCALE = 1.0 / (1 << 15)
const FloatScale = float32(1.0 / 32768.0)

// DMMul is the downmix multiplier: 1/(1+sqrt(2)+1/sqrt(2)).
// Used for 5.1 to stereo downmixing.
const DMMul = float32(0.3203772410170407)

// RSQRT2 is 1/sqrt(2), used for downmix calculations.
const RSQRT2 = float32(0.7071067811865475244)
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): add PCM conversion constants

Ported from ~/dev/faad2/libfaad/output.c:39-42

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Add Clip16 Helper Function

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
func TestClip16(t *testing.T) {
	tests := []struct {
		name  string
		input float32
		want  int16
	}{
		// Normal range
		{"zero", 0.0, 0},
		{"positive", 100.5, 101},   // Rounds to nearest
		{"negative", -100.5, -100}, // Rounds toward zero (per lrintf)

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
		{"round_half_even_up", 1.5, 2},     // 1.5 -> 2 (nearest even)
		{"round_half_even_down", 2.5, 2},   // 2.5 -> 2 (nearest even)
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
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL with "clip16 not defined"

**Step 3: Write minimal implementation**

```go
// clip16 clips and rounds a float32 to int16 range.
// Matches FAAD2's CLIP macro + lrintf behavior.
//
// Ported from: ~/dev/faad2/libfaad/output.c:64-85
func clip16(sample float32) int16 {
	// Clipping
	if sample >= 32767.0 {
		return 32767
	}
	if sample <= -32768.0 {
		return -32768
	}
	// Round to nearest (lrintf behavior)
	return int16(math.RoundToEven(float64(sample)))
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): add clip16 helper function

Clips float32 to int16 range with lrintf-style rounding.
Ported from ~/dev/faad2/libfaad/output.c:64-85

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: Add Clip24 Helper Function

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
func TestClip24(t *testing.T) {
	tests := []struct {
		name  string
		input float32
		want  int32
	}{
		// Normal range (input is already scaled by 256)
		{"zero", 0.0, 0},
		{"positive", 256000.5, 256001},
		{"negative", -256000.5, -256000},

		// Edge cases at 24-bit boundaries
		{"max_boundary", 8388607.0, 8388607},
		{"min_boundary", -8388608.0, -8388608},

		// Clipping cases
		{"clip_positive", 10000000.0, 8388607},
		{"clip_negative", -10000000.0, -8388608},
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
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL with "clip24 not defined"

**Step 3: Write minimal implementation**

```go
// clip24 clips and rounds a float32 to 24-bit signed integer range.
// Input should already be scaled by 256.
//
// Ported from: ~/dev/faad2/libfaad/output.c:154-172 (24-bit section)
func clip24(sample float32) int32 {
	// Clipping to 24-bit signed range
	if sample >= 8388607.0 {
		return 8388607
	}
	if sample <= -8388608.0 {
		return -8388608
	}
	return int32(math.RoundToEven(float64(sample)))
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): add clip24 helper function

Clips scaled float32 to 24-bit signed range.
Ported from ~/dev/faad2/libfaad/output.c:154-172

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Add Clip32 Helper Function

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
func TestClip32(t *testing.T) {
	tests := []struct {
		name  string
		input float64 // Use float64 for 32-bit range precision
		want  int32
	}{
		// Normal range (input scaled by 65536)
		{"zero", 0.0, 0},
		{"positive", 1000000.5, 1000001},
		{"negative", -1000000.5, -1000000},

		// Edge cases at 32-bit boundaries
		{"max_boundary", 2147483647.0, 2147483647},
		{"min_boundary", -2147483648.0, -2147483648},

		// Clipping cases
		{"clip_positive", 3e9, 2147483647},
		{"clip_negative", -3e9, -2147483648},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clip32(float32(tt.input))
			if got != tt.want {
				t.Errorf("clip32(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL with "clip32 not defined"

**Step 3: Write minimal implementation**

```go
// clip32 clips and rounds a float32 to int32 range.
// Input should already be scaled by 65536.
//
// Ported from: ~/dev/faad2/libfaad/output.c:224-243 (32-bit section)
func clip32(sample float32) int32 {
	// Clipping to 32-bit signed range
	if sample >= 2147483647.0 {
		return 2147483647
	}
	if sample <= -2147483648.0 {
		return -2147483648
	}
	return int32(math.RoundToEven(float64(sample)))
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): add clip32 helper function

Clips scaled float32 to int32 range.
Ported from ~/dev/faad2/libfaad/output.c:224-243

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Implement ToPCM16Bit for Mono

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
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
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL with "ToPCM16Bit not defined"

**Step 3: Write minimal implementation**

```go
// ToPCM16Bit converts float32 samples to 16-bit PCM.
//
// Parameters:
//   - input: Per-channel float32 samples (input[channel][sample])
//   - channelMap: Maps output channels to input channels
//   - channels: Number of output channels
//   - frameLen: Number of samples per channel
//   - downMatrix: Enable 5.1 to stereo downmixing
//   - upMatrix: Enable mono to stereo upmixing
//   - output: Destination slice for interleaved int16 samples
//
// Ported from: to_PCM_16bit in ~/dev/faad2/libfaad/output.c:89-152
func ToPCM16Bit(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool, output []int16) {

	switch {
	case channels == 1:
		// Mono: direct copy with clipping
		ch := channelMap[0]
		for i := uint16(0); i < frameLen; i++ {
			output[i] = clip16(input[ch][i])
		}

	default:
		// Generic multichannel (will be expanded later)
		for i := uint16(0); i < frameLen; i++ {
			for ch := uint8(0); ch < channels; ch++ {
				inp := input[channelMap[ch]][i]
				output[int(i)*int(channels)+int(ch)] = clip16(inp)
			}
		}
	}
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): add ToPCM16Bit for mono input

Implements 16-bit PCM conversion for single channel.
Ported from ~/dev/faad2/libfaad/output.c:89-152

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 6: Extend ToPCM16Bit for Stereo

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
func TestToPCM16Bit_Stereo(t *testing.T) {
	// Two channel input
	input := [][]float32{
		{100.0, 200.0, 300.0}, // Left
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
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL (stereo upmix case will fail)

**Step 3: Write implementation**

Update ToPCM16Bit to handle stereo with and without upmix:

```go
func ToPCM16Bit(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool, output []int16) {

	switch {
	case channels == 1:
		// Mono: direct copy with clipping
		ch := channelMap[0]
		for i := uint16(0); i < frameLen; i++ {
			output[i] = clip16(input[ch][i])
		}

	case channels == 2 && !downMatrix:
		if upMatrix {
			// Mono to stereo upmix: duplicate to both channels
			ch := channelMap[0]
			for i := uint16(0); i < frameLen; i++ {
				sample := clip16(input[ch][i])
				output[i*2+0] = sample
				output[i*2+1] = sample
			}
		} else {
			// True stereo
			chL := channelMap[0]
			chR := channelMap[1]
			for i := uint16(0); i < frameLen; i++ {
				output[i*2+0] = clip16(input[chL][i])
				output[i*2+1] = clip16(input[chR][i])
			}
		}

	default:
		// Generic multichannel
		for i := uint16(0); i < frameLen; i++ {
			for ch := uint8(0); ch < channels; ch++ {
				inp := input[channelMap[ch]][i]
				output[int(i)*int(channels)+int(ch)] = clip16(inp)
			}
		}
	}
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): extend ToPCM16Bit for stereo with upmix

Adds stereo output support and mono-to-stereo upmixing.
Ported from ~/dev/faad2/libfaad/output.c:109-137

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 7: Add getSample Helper for Downmix

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
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
	// 5.1 channel layout: C, L, R, Ls, Rs, LFE (indices 0-5)
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
	// Expected: DM_MUL * (500 + 1000*0.7071 + 200*0.7071)
	//         = 0.3204 * (500 + 707.1 + 141.4) = 0.3204 * 1348.5 ≈ 432.0
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
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL with "getSample not defined"

**Step 3: Write implementation**

```go
// getSample retrieves a sample, optionally applying 5.1 to stereo downmix.
//
// When downMatrix is true, channels 0-4 are: C, L, R, Ls, Rs
// Output channel 0 = L + C*RSQRT2 + Ls*RSQRT2, scaled by DM_MUL
// Output channel 1 = R + C*RSQRT2 + Rs*RSQRT2, scaled by DM_MUL
//
// Ported from: get_sample in ~/dev/faad2/libfaad/output.c:45-61
func getSample(input [][]float32, channel uint8, sample uint16,
	downMatrix bool, channelMap []uint8) float32 {

	if !downMatrix {
		return input[channelMap[channel]][sample]
	}

	// 5.1 to stereo downmix
	// channelMap[0] = Center, [1] = Left, [2] = Right, [3] = Ls, [4] = Rs
	if channel == 0 {
		// Left output
		return DMMul * (input[channelMap[1]][sample] +
			input[channelMap[0]][sample]*RSQRT2 +
			input[channelMap[3]][sample]*RSQRT2)
	}
	// Right output
	return DMMul * (input[channelMap[2]][sample] +
		input[channelMap[0]][sample]*RSQRT2 +
		input[channelMap[4]][sample]*RSQRT2)
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): add getSample helper for 5.1 downmix

Implements 5.1 to stereo downmixing per ITU-R BS.775-1.
Ported from ~/dev/faad2/libfaad/output.c:45-61

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 8: Extend ToPCM16Bit for Downmix

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
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
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL

**Step 3: Update implementation**

Update the default case in ToPCM16Bit to use getSample:

```go
func ToPCM16Bit(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool, output []int16) {

	switch {
	case channels == 1 && !downMatrix:
		// Mono: direct copy with clipping
		ch := channelMap[0]
		for i := uint16(0); i < frameLen; i++ {
			output[i] = clip16(input[ch][i])
		}

	case channels == 2 && !downMatrix:
		if upMatrix {
			// Mono to stereo upmix: duplicate to both channels
			ch := channelMap[0]
			for i := uint16(0); i < frameLen; i++ {
				sample := clip16(input[ch][i])
				output[i*2+0] = sample
				output[i*2+1] = sample
			}
		} else {
			// True stereo
			chL := channelMap[0]
			chR := channelMap[1]
			for i := uint16(0); i < frameLen; i++ {
				output[i*2+0] = clip16(input[chL][i])
				output[i*2+1] = clip16(input[chR][i])
			}
		}

	default:
		// Generic multichannel with optional downmix
		for ch := uint8(0); ch < channels; ch++ {
			for i := uint16(0); i < frameLen; i++ {
				inp := getSample(input, ch, i, downMatrix, channelMap)
				output[int(i)*int(channels)+int(ch)] = clip16(inp)
			}
		}
	}
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): extend ToPCM16Bit for 5.1 downmix

Integrates getSample for multichannel downmixing.
Ported from ~/dev/faad2/libfaad/output.c:139-151

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 9: Implement ToPCM24Bit

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
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
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL with "ToPCM24Bit not defined"

**Step 3: Write implementation**

```go
// ToPCM24Bit converts float32 samples to 24-bit PCM (stored in int32).
//
// Input values are scaled by 256 to extend from 16-bit to 24-bit range.
// Output is clipped to [-8388608, 8388607].
//
// Ported from: to_PCM_24bit in ~/dev/faad2/libfaad/output.c:154-222
func ToPCM24Bit(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool, output []int32) {

	switch {
	case channels == 1 && !downMatrix:
		ch := channelMap[0]
		for i := uint16(0); i < frameLen; i++ {
			output[i] = clip24(input[ch][i] * 256.0)
		}

	case channels == 2 && !downMatrix:
		if upMatrix {
			ch := channelMap[0]
			for i := uint16(0); i < frameLen; i++ {
				sample := clip24(input[ch][i] * 256.0)
				output[i*2+0] = sample
				output[i*2+1] = sample
			}
		} else {
			chL := channelMap[0]
			chR := channelMap[1]
			for i := uint16(0); i < frameLen; i++ {
				output[i*2+0] = clip24(input[chL][i] * 256.0)
				output[i*2+1] = clip24(input[chR][i] * 256.0)
			}
		}

	default:
		for ch := uint8(0); ch < channels; ch++ {
			for i := uint16(0); i < frameLen; i++ {
				inp := getSample(input, ch, i, downMatrix, channelMap)
				output[int(i)*int(channels)+int(ch)] = clip24(inp * 256.0)
			}
		}
	}
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): add ToPCM24Bit conversion

Converts float samples to 24-bit PCM stored in int32.
Ported from ~/dev/faad2/libfaad/output.c:154-222

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 10: Implement ToPCM32Bit

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
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
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL with "ToPCM32Bit not defined"

**Step 3: Write implementation**

```go
// ToPCM32Bit converts float32 samples to 32-bit PCM.
//
// Input values are scaled by 65536 to extend from 16-bit to 32-bit range.
// Output is clipped to int32 range.
//
// Ported from: to_PCM_32bit in ~/dev/faad2/libfaad/output.c:224-292
func ToPCM32Bit(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool, output []int32) {

	switch {
	case channels == 1 && !downMatrix:
		ch := channelMap[0]
		for i := uint16(0); i < frameLen; i++ {
			output[i] = clip32(input[ch][i] * 65536.0)
		}

	case channels == 2 && !downMatrix:
		if upMatrix {
			ch := channelMap[0]
			for i := uint16(0); i < frameLen; i++ {
				sample := clip32(input[ch][i] * 65536.0)
				output[i*2+0] = sample
				output[i*2+1] = sample
			}
		} else {
			chL := channelMap[0]
			chR := channelMap[1]
			for i := uint16(0); i < frameLen; i++ {
				output[i*2+0] = clip32(input[chL][i] * 65536.0)
				output[i*2+1] = clip32(input[chR][i] * 65536.0)
			}
		}

	default:
		for ch := uint8(0); ch < channels; ch++ {
			for i := uint16(0); i < frameLen; i++ {
				inp := getSample(input, ch, i, downMatrix, channelMap)
				output[int(i)*int(channels)+int(ch)] = clip32(inp * 65536.0)
			}
		}
	}
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): add ToPCM32Bit conversion

Converts float samples to 32-bit PCM.
Ported from ~/dev/faad2/libfaad/output.c:224-292

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 11: Implement ToPCMFloat

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
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
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL with "ToPCMFloat not defined"

**Step 3: Write implementation**

```go
// ToPCMFloat converts float32 samples to normalized float32 PCM.
//
// Input values are scaled by FloatScale (1/32768) to normalize to [-1.0, 1.0].
// No clipping is applied.
//
// Ported from: to_PCM_float in ~/dev/faad2/libfaad/output.c:294-344
func ToPCMFloat(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool, output []float32) {

	switch {
	case channels == 1 && !downMatrix:
		ch := channelMap[0]
		for i := uint16(0); i < frameLen; i++ {
			output[i] = input[ch][i] * FloatScale
		}

	case channels == 2 && !downMatrix:
		if upMatrix {
			ch := channelMap[0]
			for i := uint16(0); i < frameLen; i++ {
				sample := input[ch][i] * FloatScale
				output[i*2+0] = sample
				output[i*2+1] = sample
			}
		} else {
			chL := channelMap[0]
			chR := channelMap[1]
			for i := uint16(0); i < frameLen; i++ {
				output[i*2+0] = input[chL][i] * FloatScale
				output[i*2+1] = input[chR][i] * FloatScale
			}
		}

	default:
		for ch := uint8(0); ch < channels; ch++ {
			for i := uint16(0); i < frameLen; i++ {
				inp := getSample(input, ch, i, downMatrix, channelMap)
				output[int(i)*int(channels)+int(ch)] = inp * FloatScale
			}
		}
	}
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): add ToPCMFloat conversion

Converts float samples to normalized [-1.0, 1.0] range.
Ported from ~/dev/faad2/libfaad/output.c:294-344

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 12: Implement ToPCMDouble

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
func TestToPCMDouble_Mono(t *testing.T) {
	input := [][]float32{
		{0.0, 32768.0, -32768.0},
	}
	channelMap := []uint8{0}

	output := make([]float64, 3)
	ToPCMDouble(input, channelMap, 1, 3, false, false, output)

	expected := []float64{0.0, 1.0, -1.0}
	for i, want := range expected {
		if math.Abs(output[i]-want) > 1e-6 {
			t.Errorf("output[%d] = %v, want %v", i, output[i], want)
		}
	}
}

func TestToPCMDouble_Stereo(t *testing.T) {
	input := [][]float32{
		{16384.0, 32768.0},
		{-16384.0, -32768.0},
	}
	channelMap := []uint8{0, 1}

	output := make([]float64, 4)
	ToPCMDouble(input, channelMap, 2, 2, false, false, output)

	expected := []float64{0.5, -0.5, 1.0, -1.0}
	for i, want := range expected {
		if math.Abs(output[i]-want) > 1e-6 {
			t.Errorf("output[%d] = %v, want %v", i, output[i], want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL with "ToPCMDouble not defined"

**Step 3: Write implementation**

```go
// ToPCMDouble converts float32 samples to normalized float64 PCM.
//
// Input values are scaled by FloatScale (1/32768) to normalize to [-1.0, 1.0].
// No clipping is applied.
//
// Ported from: to_PCM_double in ~/dev/faad2/libfaad/output.c:346-396
func ToPCMDouble(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool, output []float64) {

	switch {
	case channels == 1 && !downMatrix:
		ch := channelMap[0]
		for i := uint16(0); i < frameLen; i++ {
			output[i] = float64(input[ch][i]) * float64(FloatScale)
		}

	case channels == 2 && !downMatrix:
		if upMatrix {
			ch := channelMap[0]
			for i := uint16(0); i < frameLen; i++ {
				sample := float64(input[ch][i]) * float64(FloatScale)
				output[i*2+0] = sample
				output[i*2+1] = sample
			}
		} else {
			chL := channelMap[0]
			chR := channelMap[1]
			for i := uint16(0); i < frameLen; i++ {
				output[i*2+0] = float64(input[chL][i]) * float64(FloatScale)
				output[i*2+1] = float64(input[chR][i]) * float64(FloatScale)
			}
		}

	default:
		for ch := uint8(0); ch < channels; ch++ {
			for i := uint16(0); i < frameLen; i++ {
				inp := getSample(input, ch, i, downMatrix, channelMap)
				output[int(i)*int(channels)+int(ch)] = float64(inp) * float64(FloatScale)
			}
		}
	}
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): add ToPCMDouble conversion

Converts float samples to normalized float64 range.
Ported from ~/dev/faad2/libfaad/output.c:346-396

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 13: Implement Main OutputToPCM Function

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
func TestOutputToPCM_16Bit(t *testing.T) {
	input := [][]float32{
		{100.0, 200.0},
		{-100.0, -200.0},
	}
	channelMap := []uint8{0, 1}

	result := OutputToPCM(input, channelMap, 2, 2, 1, false, false) // format=1 is 16-bit
	output, ok := result.([]int16)
	if !ok {
		t.Fatalf("expected []int16, got %T", result)
	}

	expected := []int16{100, -100, 200, -200}
	for i, want := range expected {
		if output[i] != want {
			t.Errorf("output[%d] = %d, want %d", i, output[i], want)
		}
	}
}

func TestOutputToPCM_Float(t *testing.T) {
	input := [][]float32{
		{32768.0},
	}
	channelMap := []uint8{0}

	result := OutputToPCM(input, channelMap, 1, 1, 4, false, false) // format=4 is float
	output, ok := result.([]float32)
	if !ok {
		t.Fatalf("expected []float32, got %T", result)
	}

	if math.Abs(float64(output[0]-1.0)) > 1e-6 {
		t.Errorf("output[0] = %v, want 1.0", output[0])
	}
}
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL with "OutputToPCM not defined"

**Step 3: Write implementation**

```go
// OutputToPCM converts float32 samples to the requested PCM format.
//
// Returns a slice of the appropriate type:
//   - format 1 (16-bit): []int16
//   - format 2 (24-bit): []int32
//   - format 3 (32-bit): []int32
//   - format 4 (float):  []float32
//   - format 5 (double): []float64
//
// Ported from: output_to_PCM in ~/dev/faad2/libfaad/output.c:398-437
func OutputToPCM(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, format uint8, downMatrix, upMatrix bool) interface{} {

	totalSamples := int(frameLen) * int(channels)

	switch format {
	case 1: // FAAD_FMT_16BIT
		output := make([]int16, totalSamples)
		ToPCM16Bit(input, channelMap, channels, frameLen, downMatrix, upMatrix, output)
		return output

	case 2: // FAAD_FMT_24BIT
		output := make([]int32, totalSamples)
		ToPCM24Bit(input, channelMap, channels, frameLen, downMatrix, upMatrix, output)
		return output

	case 3: // FAAD_FMT_32BIT
		output := make([]int32, totalSamples)
		ToPCM32Bit(input, channelMap, channels, frameLen, downMatrix, upMatrix, output)
		return output

	case 4: // FAAD_FMT_FLOAT
		output := make([]float32, totalSamples)
		ToPCMFloat(input, channelMap, channels, frameLen, downMatrix, upMatrix, output)
		return output

	case 5: // FAAD_FMT_DOUBLE
		output := make([]float64, totalSamples)
		ToPCMDouble(input, channelMap, channels, frameLen, downMatrix, upMatrix, output)
		return output

	default:
		// Default to 16-bit
		output := make([]int16, totalSamples)
		ToPCM16Bit(input, channelMap, channels, frameLen, downMatrix, upMatrix, output)
		return output
	}
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): add main OutputToPCM dispatcher

Dispatches to format-specific converters based on format code.
Ported from ~/dev/faad2/libfaad/output.c:398-437

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 14: Add Type-Safe OutputToPCM Variants

**Files:**
- Modify: `internal/output/pcm.go`
- Modify: `internal/output/pcm_test.go`

**Step 1: Write the test**

```go
func TestOutputToPCM16(t *testing.T) {
	input := [][]float32{{100.0, 200.0}}
	channelMap := []uint8{0}

	output := OutputToPCM16(input, channelMap, 1, 2, false, false)
	if output[0] != 100 || output[1] != 200 {
		t.Errorf("unexpected output: %v", output)
	}
}

func TestOutputToPCMFloat32(t *testing.T) {
	input := [][]float32{{32768.0}}
	channelMap := []uint8{0}

	output := OutputToPCMFloat32(input, channelMap, 1, 1, false, false)
	if math.Abs(float64(output[0]-1.0)) > 1e-6 {
		t.Errorf("output[0] = %v, want 1.0", output[0])
	}
}
```

**Step 2: Run test to verify it fails**

```bash
make test PKG=./internal/output
```
Expected: FAIL with "OutputToPCM16 not defined"

**Step 3: Write implementation**

```go
// OutputToPCM16 converts float32 samples to 16-bit PCM.
// This is a type-safe wrapper around ToPCM16Bit.
func OutputToPCM16(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool) []int16 {

	output := make([]int16, int(frameLen)*int(channels))
	ToPCM16Bit(input, channelMap, channels, frameLen, downMatrix, upMatrix, output)
	return output
}

// OutputToPCM24 converts float32 samples to 24-bit PCM (stored in int32).
// This is a type-safe wrapper around ToPCM24Bit.
func OutputToPCM24(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool) []int32 {

	output := make([]int32, int(frameLen)*int(channels))
	ToPCM24Bit(input, channelMap, channels, frameLen, downMatrix, upMatrix, output)
	return output
}

// OutputToPCM32 converts float32 samples to 32-bit PCM.
// This is a type-safe wrapper around ToPCM32Bit.
func OutputToPCM32(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool) []int32 {

	output := make([]int32, int(frameLen)*int(channels))
	ToPCM32Bit(input, channelMap, channels, frameLen, downMatrix, upMatrix, output)
	return output
}

// OutputToPCMFloat32 converts float32 samples to normalized float32 PCM.
// This is a type-safe wrapper around ToPCMFloat.
func OutputToPCMFloat32(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool) []float32 {

	output := make([]float32, int(frameLen)*int(channels))
	ToPCMFloat(input, channelMap, channels, frameLen, downMatrix, upMatrix, output)
	return output
}

// OutputToPCMFloat64 converts float32 samples to normalized float64 PCM.
// This is a type-safe wrapper around ToPCMDouble.
func OutputToPCMFloat64(input [][]float32, channelMap []uint8, channels uint8,
	frameLen uint16, downMatrix, upMatrix bool) []float64 {

	output := make([]float64, int(frameLen)*int(channels))
	ToPCMDouble(input, channelMap, channels, frameLen, downMatrix, upMatrix, output)
	return output
}
```

**Step 4: Run test to verify it passes**

```bash
make test PKG=./internal/output
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/pcm.go internal/output/pcm_test.go
git commit -m "feat(output): add type-safe OutputToPCM variants

Provides OutputToPCM16, OutputToPCM24, OutputToPCM32,
OutputToPCMFloat32, and OutputToPCMFloat64 for type-safe usage.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 15: Run Full Test Suite and Verify

**Files:**
- None (verification only)

**Step 1: Run all tests for the output package**

```bash
make test PKG=./internal/output
```
Expected: All tests pass

**Step 2: Run linter**

```bash
make lint
```
Expected: No errors

**Step 3: Run full check**

```bash
make check
```
Expected: All checks pass

**Step 4: Commit any cleanup**

If any formatting/linting fixes were needed:
```bash
git add -A
git commit -m "chore(output): cleanup and formatting

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Summary

This plan implements Step 6.2 (PCM Output Conversion) from the migration steps with:

1. **Constants**: FloatScale, DMMul, RSQRT2
2. **Clip functions**: clip16, clip24, clip32
3. **Per-format converters**: ToPCM16Bit, ToPCM24Bit, ToPCM32Bit, ToPCMFloat, ToPCMDouble
4. **Downmix helper**: getSample for 5.1 to stereo
5. **Main dispatcher**: OutputToPCM (interface{} return)
6. **Type-safe wrappers**: OutputToPCM16, OutputToPCM24, etc.

Total estimated lines: ~350-400 (matches migration guide estimate)

### Files Created/Modified

| File | Action | Description |
|------|--------|-------------|
| `internal/output/pcm.go` | Create | PCM conversion implementation |
| `internal/output/pcm_test.go` | Create | Comprehensive tests |

### FAAD2 Validation

The final PCM output should match FAAD2 reference data exactly (bit-perfect for int16).
Use the `scripts/check_faad2` tool to generate reference PCM and compare:

```bash
./scripts/check_faad2 testdata/test.aac
# Compare /tmp/faad2_ref_test/frame_NNNN_pcm.bin against Go output
```
