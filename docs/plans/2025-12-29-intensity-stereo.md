# Intensity Stereo Decoding Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement intensity stereo (IS) decoding for AAC stereo channel pairs.

**Architecture:** Intensity stereo is a bandwidth-saving technique where high-frequency bands are encoded as a single channel with a scale factor. The right channel is reconstructed by scaling the left channel. The sign of the right channel depends on both the codebook type (INTENSITY_HCB vs INTENSITY_HCB2) and the M/S stereo mask for that band.

**Tech Stack:** Go 1.25+, TDD approach, FAAD2 reference implementation.

---

## Background

Intensity stereo works by:
1. Encoding only the left channel for certain high-frequency bands
2. Storing a scale factor for each IS band
3. Reconstructing right channel as: `R = L * 0.5^(scale_factor/4) * sign`
4. The sign is determined by comparing `is_intensity()` and `invert_intensity()` results

**FAAD2 Source Files:**
- `is.c` (106 lines) - Main decode function
- `is.h` (67 lines) - Helper inline functions

**Existing Code to Build On:**
- `internal/spectrum/helpers.go` - Already has `IsIntensityICS()` function
- `internal/spectrum/ms.go` - Pattern to follow for loop structure
- `internal/syntax/ics.go` - ICStream struct with all required fields

---

### Task 1: Add InvertIntensity Helper Function

**Files:**
- Modify: `internal/spectrum/helpers.go`
- Modify: `internal/spectrum/helpers_test.go`

**Step 1: Write the failing test**

Add to `helpers_test.go`:

```go
func TestInvertIntensity(t *testing.T) {
	tests := []struct {
		name          string
		msMaskPresent uint8
		msUsed        uint8
		expected      int8
	}{
		{"ms_mask_present=0 (no MS)", 0, 0, 1},
		{"ms_mask_present=0, ms_used=1", 0, 1, 1},
		{"ms_mask_present=1, ms_used=0", 1, 0, 1},
		{"ms_mask_present=1, ms_used=1", 1, 1, -1},
		{"ms_mask_present=2 (all bands)", 2, 0, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ics := &syntax.ICStream{
				MSMaskPresent: tc.msMaskPresent,
			}
			ics.MSUsed[0][0] = tc.msUsed

			got := InvertIntensity(ics, 0, 0)
			if got != tc.expected {
				t.Errorf("InvertIntensity() = %d, want %d", got, tc.expected)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/spectrum`
Expected: FAIL with "undefined: InvertIntensity"

**Step 3: Write minimal implementation**

Add to `helpers.go`:

```go
// InvertIntensity returns the intensity stereo sign inversion factor.
// Returns -1 if the M/S mask indicates inversion, 1 otherwise.
//
// Ported from: invert_intensity() in ~/dev/faad2/libfaad/is.h:56-61
func InvertIntensity(ics *syntax.ICStream, group, sfb uint8) int8 {
	if ics.MSMaskPresent == 1 {
		return 1 - 2*int8(ics.MSUsed[group][sfb])
	}
	return 1
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/helpers.go internal/spectrum/helpers_test.go
git commit -m "feat(spectrum): add InvertIntensity helper for IS decoding

Ported from invert_intensity() in ~/dev/faad2/libfaad/is.h:56-61

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Add ISDecodeConfig and ISDecode Function Signature

**Files:**
- Create: `internal/spectrum/is.go`
- Create: `internal/spectrum/is_test.go`

**Step 1: Write the failing test**

Create `is_test.go`:

```go
package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestISDecode_NoIntensityBands(t *testing.T) {
	// When no intensity stereo bands exist, spectra should be unchanged
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = 1 // Normal codebook, not intensity

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SFBCB[0][0] = 1 // Normal codebook, not intensity

	lSpec := []float64{1.0, 2.0, 3.0, 4.0}
	rSpec := []float64{5.0, 6.0, 7.0, 8.0}

	cfg := &ISDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	ISDecode(lSpec, rSpec, cfg)

	// Should be unchanged
	expectedL := []float64{1.0, 2.0, 3.0, 4.0}
	expectedR := []float64{5.0, 6.0, 7.0, 8.0}

	for i := range lSpec {
		if lSpec[i] != expectedL[i] {
			t.Errorf("lSpec[%d] = %v, want %v", i, lSpec[i], expectedL[i])
		}
		if rSpec[i] != expectedR[i] {
			t.Errorf("rSpec[%d] = %v, want %v", i, rSpec[i], expectedR[i])
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/spectrum`
Expected: FAIL with "undefined: ISDecodeConfig" and "undefined: ISDecode"

**Step 3: Write minimal implementation**

Create `is.go`:

```go
package spectrum

import "github.com/llehouerou/go-aac/internal/syntax"

// ISDecodeConfig holds configuration for intensity stereo decoding.
type ISDecodeConfig struct {
	// ICSL is the left channel's individual channel stream
	ICSL *syntax.ICStream

	// ICSR is the right channel's individual channel stream (contains IS scale factors)
	ICSR *syntax.ICStream

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16
}

// ISDecode applies intensity stereo decoding to spectral coefficients.
// The right channel spectrum is reconstructed from the left channel
// for bands coded with intensity stereo (INTENSITY_HCB or INTENSITY_HCB2).
//
// The left channel is NOT modified; only the right channel is written.
//
// Ported from: is_decode() in ~/dev/faad2/libfaad/is.c:46-106
func ISDecode(lSpec, rSpec []float64, cfg *ISDecodeConfig) {
	// Stub - to be implemented
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/spectrum`
Expected: PASS (stub does nothing, spectra unchanged)

**Step 5: Commit**

```bash
git add internal/spectrum/is.go internal/spectrum/is_test.go
git commit -m "feat(spectrum): add ISDecode stub and config struct

Ported from is_decode() in ~/dev/faad2/libfaad/is.c:46-106

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Implement Basic Intensity Stereo Decoding

**Files:**
- Modify: `internal/spectrum/is.go`
- Modify: `internal/spectrum/is_test.go`

**Step 1: Write the failing test**

Add to `is_test.go`:

```go
func TestISDecode_BasicIntensityStereo(t *testing.T) {
	// Basic intensity stereo: scale factor = 0 means scale = 1.0 (0.5^0)
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = 1 // Normal codebook in left

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SWBOffsetMax = 1024
	icsR.SFBCB[0][0] = uint8(huffman.IntensityHCB) // Intensity stereo
	icsR.ScaleFactors[0][0] = 0                     // scale = 0.5^(0/4) = 1.0

	// Left channel: [10, 20, 30, 40]
	// Right should become: [10, 20, 30, 40] (scale=1.0, same sign)
	lSpec := []float64{10.0, 20.0, 30.0, 40.0}
	rSpec := make([]float64, 4)

	cfg := &ISDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	ISDecode(lSpec, rSpec, cfg)

	expectedR := []float64{10.0, 20.0, 30.0, 40.0}
	for i := range rSpec {
		if rSpec[i] != expectedR[i] {
			t.Errorf("rSpec[%d] = %v, want %v", i, rSpec[i], expectedR[i])
		}
	}

	// Left should be unchanged
	expectedL := []float64{10.0, 20.0, 30.0, 40.0}
	for i := range lSpec {
		if lSpec[i] != expectedL[i] {
			t.Errorf("lSpec[%d] modified: got %v, want %v", i, lSpec[i], expectedL[i])
		}
	}
}
```

Also add import for huffman at the top of the test file:

```go
import (
	"testing"

	"github.com/llehouerou/go-aac/internal/huffman"
	"github.com/llehouerou/go-aac/internal/syntax"
)
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/spectrum`
Expected: FAIL with "rSpec[0] = 0, want 10.0"

**Step 3: Write implementation**

Replace the stub in `is.go`:

```go
package spectrum

import (
	"math"

	"github.com/llehouerou/go-aac/internal/syntax"
)

// ISDecodeConfig holds configuration for intensity stereo decoding.
type ISDecodeConfig struct {
	// ICSL is the left channel's individual channel stream
	ICSL *syntax.ICStream

	// ICSR is the right channel's individual channel stream (contains IS scale factors)
	ICSR *syntax.ICStream

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16
}

// ISDecode applies intensity stereo decoding to spectral coefficients.
// The right channel spectrum is reconstructed from the left channel
// for bands coded with intensity stereo (INTENSITY_HCB or INTENSITY_HCB2).
//
// The left channel is NOT modified; only the right channel is written.
//
// Ported from: is_decode() in ~/dev/faad2/libfaad/is.c:46-106
func ISDecode(lSpec, rSpec []float64, cfg *ISDecodeConfig) {
	icsL := cfg.ICSL
	icsR := cfg.ICSR

	nshort := cfg.FrameLength / 8
	group := uint16(0)

	for g := uint8(0); g < icsR.NumWindowGroups; g++ {
		for b := uint8(0); b < icsR.WindowGroupLength[g]; b++ {
			for sfb := uint8(0); sfb < icsR.MaxSFB; sfb++ {
				isDir := IsIntensityICS(icsR, g, sfb)
				if isDir != 0 {
					// Get scale factor and clamp to valid range
					scaleFactor := icsR.ScaleFactors[g][sfb]
					if scaleFactor < -120 {
						scaleFactor = -120
					} else if scaleFactor > 120 {
						scaleFactor = 120
					}

					// Calculate scale: 0.5^(scaleFactor/4)
					scale := math.Pow(0.5, 0.25*float64(scaleFactor))

					// Determine sign inversion
					invertSign := isDir != InvertIntensity(icsL, g, sfb)

					// Calculate SFB bounds, clamped to swb_offset_max
					start := icsR.SWBOffset[sfb]
					end := icsR.SWBOffset[sfb+1]
					if end > icsL.SWBOffsetMax {
						end = icsL.SWBOffsetMax
					}

					// Copy scaled left to right
					for i := start; i < end; i++ {
						k := group*nshort + i
						rSpec[k] = lSpec[k] * scale
						if invertSign {
							rSpec[k] = -rSpec[k]
						}
					}
				}
			}
			group++
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/is.go internal/spectrum/is_test.go
git commit -m "feat(spectrum): implement basic intensity stereo decoding

Implements is_decode() from ~/dev/faad2/libfaad/is.c:46-106
- Reconstructs right channel from scaled left channel
- Applies scale factor: 0.5^(sf/4)
- Handles sign inversion based on codebook and MS mask

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Add Test for Scale Factor Scaling

**Files:**
- Modify: `internal/spectrum/is_test.go`

**Step 1: Write the test**

Add to `is_test.go`:

```go
func TestISDecode_ScaleFactorScaling(t *testing.T) {
	// Test that scale factor correctly scales the output
	// scale = 0.5^(sf/4)
	// sf=4 -> scale = 0.5^1 = 0.5
	// sf=-4 -> scale = 0.5^(-1) = 2.0
	tests := []struct {
		name        string
		scaleFactor int16
		expected    float64
	}{
		{"sf=0 -> scale=1.0", 0, 1.0},
		{"sf=4 -> scale=0.5", 4, 0.5},
		{"sf=-4 -> scale=2.0", -4, 2.0},
		{"sf=8 -> scale=0.25", 8, 0.25},
		{"sf=2 -> scale~0.707", 2, 0.7071067811865476}, // 0.5^0.5
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			icsL := &syntax.ICStream{
				NumWindowGroups: 1,
				MaxSFB:          1,
				NumSWB:          1,
				WindowSequence:  syntax.OnlyLongSequence,
			}
			icsL.WindowGroupLength[0] = 1
			icsL.SWBOffset[0] = 0
			icsL.SWBOffset[1] = 1
			icsL.SWBOffsetMax = 1024
			icsL.SFBCB[0][0] = 1

			icsR := &syntax.ICStream{
				NumWindowGroups: 1,
				MaxSFB:          1,
				NumSWB:          1,
				WindowSequence:  syntax.OnlyLongSequence,
			}
			icsR.WindowGroupLength[0] = 1
			icsR.SWBOffset[0] = 0
			icsR.SWBOffset[1] = 1
			icsR.SWBOffsetMax = 1024
			icsR.SFBCB[0][0] = uint8(huffman.IntensityHCB)
			icsR.ScaleFactors[0][0] = tc.scaleFactor

			lSpec := []float64{1.0} // Use 1.0 so result equals scale
			rSpec := make([]float64, 1)

			cfg := &ISDecodeConfig{
				ICSL:        icsL,
				ICSR:        icsR,
				FrameLength: 1024,
			}

			ISDecode(lSpec, rSpec, cfg)

			const epsilon = 1e-10
			if diff := math.Abs(rSpec[0] - tc.expected); diff > epsilon {
				t.Errorf("rSpec[0] = %v, want %v (diff=%v)", rSpec[0], tc.expected, diff)
			}
		})
	}
}
```

Also add `"math"` to imports.

**Step 2: Run test to verify it passes**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/is_test.go
git commit -m "test(spectrum): add scale factor scaling tests for ISDecode

Tests various scale factor values and their expected scaling:
- sf=0 -> scale=1.0
- sf=4 -> scale=0.5
- sf=-4 -> scale=2.0

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Add Test for Sign Inversion (INTENSITY_HCB2)

**Files:**
- Modify: `internal/spectrum/is_test.go`

**Step 1: Write the test**

Add to `is_test.go`:

```go
func TestISDecode_IntensityHCB2_InvertsSign(t *testing.T) {
	// INTENSITY_HCB2 with ms_mask_present=0 should invert the sign
	// is_intensity() returns -1, invert_intensity() returns 1
	// Since -1 != 1, sign is inverted
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		MSMaskPresent:   0, // No M/S -> invert_intensity returns 1
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = 1

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SWBOffsetMax = 1024
	icsR.SFBCB[0][0] = uint8(huffman.IntensityHCB2) // Out-of-phase: is_intensity=-1
	icsR.ScaleFactors[0][0] = 0                      // scale = 1.0

	lSpec := []float64{10.0, -20.0, 30.0, -40.0}
	rSpec := make([]float64, 4)

	cfg := &ISDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	ISDecode(lSpec, rSpec, cfg)

	// is_intensity=-1, invert_intensity=1, so signs are inverted
	expectedR := []float64{-10.0, 20.0, -30.0, 40.0}
	for i := range rSpec {
		if rSpec[i] != expectedR[i] {
			t.Errorf("rSpec[%d] = %v, want %v", i, rSpec[i], expectedR[i])
		}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/is_test.go
git commit -m "test(spectrum): add INTENSITY_HCB2 sign inversion test

Verifies that INTENSITY_HCB2 (out-of-phase) inverts the sign
when ms_mask_present=0

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Add Test for M/S Mask Interaction

**Files:**
- Modify: `internal/spectrum/is_test.go`

**Step 1: Write the test**

Add to `is_test.go`:

```go
func TestISDecode_MSMaskInteraction(t *testing.T) {
	// When ms_mask_present=1 and ms_used=1, invert_intensity returns -1
	// Combined with INTENSITY_HCB (is_intensity=1), sign is inverted
	// Combined with INTENSITY_HCB2 (is_intensity=-1), sign is NOT inverted
	tests := []struct {
		name       string
		codebook   uint8
		msUsed     uint8
		expectSign float64 // Expected sign: 1.0 or -1.0
	}{
		// INTENSITY_HCB (is=1) with ms_used=0 (inv=1): 1 != 1? No -> no invert
		{"HCB, ms_used=0", uint8(huffman.IntensityHCB), 0, 1.0},
		// INTENSITY_HCB (is=1) with ms_used=1 (inv=-1): 1 != -1? Yes -> invert
		{"HCB, ms_used=1", uint8(huffman.IntensityHCB), 1, -1.0},
		// INTENSITY_HCB2 (is=-1) with ms_used=0 (inv=1): -1 != 1? Yes -> invert
		{"HCB2, ms_used=0", uint8(huffman.IntensityHCB2), 0, -1.0},
		// INTENSITY_HCB2 (is=-1) with ms_used=1 (inv=-1): -1 != -1? No -> no invert
		{"HCB2, ms_used=1", uint8(huffman.IntensityHCB2), 1, 1.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			icsL := &syntax.ICStream{
				NumWindowGroups: 1,
				MaxSFB:          1,
				NumSWB:          1,
				MSMaskPresent:   1, // Per-band M/S
				WindowSequence:  syntax.OnlyLongSequence,
			}
			icsL.WindowGroupLength[0] = 1
			icsL.SWBOffset[0] = 0
			icsL.SWBOffset[1] = 1
			icsL.SWBOffsetMax = 1024
			icsL.SFBCB[0][0] = 1
			icsL.MSUsed[0][0] = tc.msUsed

			icsR := &syntax.ICStream{
				NumWindowGroups: 1,
				MaxSFB:          1,
				NumSWB:          1,
				WindowSequence:  syntax.OnlyLongSequence,
			}
			icsR.WindowGroupLength[0] = 1
			icsR.SWBOffset[0] = 0
			icsR.SWBOffset[1] = 1
			icsR.SWBOffsetMax = 1024
			icsR.SFBCB[0][0] = tc.codebook
			icsR.ScaleFactors[0][0] = 0 // scale = 1.0

			lSpec := []float64{10.0}
			rSpec := make([]float64, 1)

			cfg := &ISDecodeConfig{
				ICSL:        icsL,
				ICSR:        icsR,
				FrameLength: 1024,
			}

			ISDecode(lSpec, rSpec, cfg)

			expected := 10.0 * tc.expectSign
			if rSpec[0] != expected {
				t.Errorf("rSpec[0] = %v, want %v", rSpec[0], expected)
			}
		})
	}
}
```

**Step 2: Run test to verify it passes**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/is_test.go
git commit -m "test(spectrum): add M/S mask interaction tests for ISDecode

Tests the 4 combinations of INTENSITY_HCB/HCB2 with ms_used=0/1
to verify correct sign determination

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 7: Add Test for Short Blocks

**Files:**
- Modify: `internal/spectrum/is_test.go`

**Step 1: Write the test**

Add to `is_test.go`:

```go
func TestISDecode_ShortBlocks(t *testing.T) {
	// Test 8 short windows grouped into 2 groups of 4
	icsL := &syntax.ICStream{
		NumWindowGroups: 2,
		NumWindows:      8,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.EightShortSequence,
	}
	icsL.WindowGroupLength[0] = 4
	icsL.WindowGroupLength[1] = 4
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffsetMax = 128
	icsL.SFBCB[0][0] = 1
	icsL.SFBCB[1][0] = 1

	icsR := &syntax.ICStream{
		NumWindowGroups: 2,
		NumWindows:      8,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.EightShortSequence,
	}
	icsR.WindowGroupLength[0] = 4
	icsR.WindowGroupLength[1] = 4
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SWBOffsetMax = 128
	icsR.SFBCB[0][0] = uint8(huffman.IntensityHCB) // IS in first group
	icsR.SFBCB[1][0] = uint8(huffman.IntensityHCB) // IS in second group
	icsR.ScaleFactors[0][0] = 0                     // scale = 1.0
	icsR.ScaleFactors[1][0] = 4                     // scale = 0.5

	// FrameLength=1024, nshort=128
	lSpec := make([]float64, 1024)
	rSpec := make([]float64, 1024)
	for i := 0; i < 1024; i++ {
		lSpec[i] = 10.0
	}

	cfg := &ISDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	ISDecode(lSpec, rSpec, cfg)

	// Check first group (windows 0-3): scale=1.0
	for win := 0; win < 4; win++ {
		base := win * 128
		for i := 0; i < 4; i++ {
			idx := base + i
			if rSpec[idx] != 10.0 {
				t.Errorf("rSpec[%d] (group 0, win=%d) = %v, want 10.0", idx, win, rSpec[idx])
			}
		}
	}

	// Check second group (windows 4-7): scale=0.5
	for win := 4; win < 8; win++ {
		base := win * 128
		for i := 0; i < 4; i++ {
			idx := base + i
			if rSpec[idx] != 5.0 {
				t.Errorf("rSpec[%d] (group 1, win=%d) = %v, want 5.0", idx, win, rSpec[idx])
			}
		}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/is_test.go
git commit -m "test(spectrum): add short blocks test for ISDecode

Verifies intensity stereo works correctly with 8 short windows
grouped into 2 groups with different scale factors

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 8: Add Test for SWBOffsetMax Clamping

**Files:**
- Modify: `internal/spectrum/is_test.go`

**Step 1: Write the test**

Add to `is_test.go`:

```go
func TestISDecode_SWBOffsetMaxClamping(t *testing.T) {
	// Test that SFB bounds are clamped to SWBOffsetMax
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 100 // SFB would go to 100
	icsL.SWBOffsetMax = 50  // But max is 50
	icsL.SFBCB[0][0] = 1

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 100
	icsR.SWBOffsetMax = 128
	icsR.SFBCB[0][0] = uint8(huffman.IntensityHCB)
	icsR.ScaleFactors[0][0] = 0 // scale = 1.0

	lSpec := make([]float64, 100)
	rSpec := make([]float64, 100)
	for i := 0; i < 100; i++ {
		lSpec[i] = 10.0
	}

	cfg := &ISDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	ISDecode(lSpec, rSpec, cfg)

	// First 50: IS applied
	for i := 0; i < 50; i++ {
		if rSpec[i] != 10.0 {
			t.Errorf("rSpec[%d] = %v, want 10.0", i, rSpec[i])
		}
	}

	// 50-99: Not touched (beyond SWBOffsetMax)
	for i := 50; i < 100; i++ {
		if rSpec[i] != 0.0 {
			t.Errorf("rSpec[%d] = %v, want 0.0 (beyond SWBOffsetMax)", i, rSpec[i])
		}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/is_test.go
git commit -m "test(spectrum): add SWBOffsetMax clamping test for ISDecode

Verifies that SFB bounds are clamped to the left channel's SWBOffsetMax

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 9: Add Test for Scale Factor Clamping

**Files:**
- Modify: `internal/spectrum/is_test.go`

**Step 1: Write the test**

Add to `is_test.go`:

```go
func TestISDecode_ScaleFactorClamping(t *testing.T) {
	// FAAD2 clamps scale factor to [-120, 120]
	// Test extreme values don't cause overflow
	tests := []struct {
		name        string
		scaleFactor int16
	}{
		{"extreme negative", -200},
		{"at min clamp", -120},
		{"extreme positive", 200},
		{"at max clamp", 120},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			icsL := &syntax.ICStream{
				NumWindowGroups: 1,
				MaxSFB:          1,
				NumSWB:          1,
				WindowSequence:  syntax.OnlyLongSequence,
			}
			icsL.WindowGroupLength[0] = 1
			icsL.SWBOffset[0] = 0
			icsL.SWBOffset[1] = 1
			icsL.SWBOffsetMax = 1024
			icsL.SFBCB[0][0] = 1

			icsR := &syntax.ICStream{
				NumWindowGroups: 1,
				MaxSFB:          1,
				NumSWB:          1,
				WindowSequence:  syntax.OnlyLongSequence,
			}
			icsR.WindowGroupLength[0] = 1
			icsR.SWBOffset[0] = 0
			icsR.SWBOffset[1] = 1
			icsR.SWBOffsetMax = 1024
			icsR.SFBCB[0][0] = uint8(huffman.IntensityHCB)
			icsR.ScaleFactors[0][0] = tc.scaleFactor

			lSpec := []float64{1.0}
			rSpec := make([]float64, 1)

			cfg := &ISDecodeConfig{
				ICSL:        icsL,
				ICSR:        icsR,
				FrameLength: 1024,
			}

			// Should not panic
			ISDecode(lSpec, rSpec, cfg)

			// Result should be finite (not Inf or NaN)
			if math.IsInf(rSpec[0], 0) || math.IsNaN(rSpec[0]) {
				t.Errorf("rSpec[0] = %v, expected finite value", rSpec[0])
			}
		})
	}
}
```

**Step 2: Run test to verify it passes**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/is_test.go
git commit -m "test(spectrum): add scale factor clamping tests for ISDecode

Verifies extreme scale factors don't cause overflow/underflow

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 10: Add Test for Mixed IS and Non-IS Bands

**Files:**
- Modify: `internal/spectrum/is_test.go`

**Step 1: Write the test**

Add to `is_test.go`:

```go
func TestISDecode_MixedBands(t *testing.T) {
	// Test that only IS bands are modified, others are untouched
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          3,
		NumSWB:          3,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 4
	icsL.SWBOffset[2] = 8
	icsL.SWBOffset[3] = 12
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = 1
	icsL.SFBCB[0][1] = 1
	icsL.SFBCB[0][2] = 1

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          3,
		NumSWB:          3,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 4
	icsR.SWBOffset[2] = 8
	icsR.SWBOffset[3] = 12
	icsR.SWBOffsetMax = 1024
	icsR.SFBCB[0][0] = 1                            // Normal (not IS)
	icsR.SFBCB[0][1] = uint8(huffman.IntensityHCB)  // IS
	icsR.SFBCB[0][2] = 1                            // Normal (not IS)
	icsR.ScaleFactors[0][1] = 0                     // scale = 1.0

	lSpec := make([]float64, 12)
	rSpec := make([]float64, 12)
	for i := 0; i < 12; i++ {
		lSpec[i] = 10.0
		rSpec[i] = 99.0 // Initial value for non-IS bands
	}

	cfg := &ISDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
	}

	ISDecode(lSpec, rSpec, cfg)

	// SFB 0 (0-3): Not IS, should be unchanged
	for i := 0; i < 4; i++ {
		if rSpec[i] != 99.0 {
			t.Errorf("rSpec[%d] = %v, want 99.0 (non-IS band)", i, rSpec[i])
		}
	}

	// SFB 1 (4-7): IS, should be copied from left
	for i := 4; i < 8; i++ {
		if rSpec[i] != 10.0 {
			t.Errorf("rSpec[%d] = %v, want 10.0 (IS band)", i, rSpec[i])
		}
	}

	// SFB 2 (8-11): Not IS, should be unchanged
	for i := 8; i < 12; i++ {
		if rSpec[i] != 99.0 {
			t.Errorf("rSpec[%d] = %v, want 99.0 (non-IS band)", i, rSpec[i])
		}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/is_test.go
git commit -m "test(spectrum): add mixed IS/non-IS bands test

Verifies that only intensity stereo bands are modified and
other bands remain untouched

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 11: Run Full Test Suite and Lint

**Files:**
- None (validation only)

**Step 1: Run make check**

Run: `make check`
Expected: All tests pass, no lint errors

**Step 2: Fix any issues**

If there are any lint issues or test failures, fix them before proceeding.

**Step 3: Commit any fixes**

If fixes were needed:
```bash
git add -A
git commit -m "fix(spectrum): address lint issues in intensity stereo

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Summary

**Files created:**
- `internal/spectrum/is.go` - Intensity stereo decoding (~60 lines)
- `internal/spectrum/is_test.go` - Comprehensive tests (~300 lines)

**Files modified:**
- `internal/spectrum/helpers.go` - Added InvertIntensity() helper
- `internal/spectrum/helpers_test.go` - Added InvertIntensity tests

**Key implementation details:**
1. `IsIntensityICS()` already exists - detects IS bands
2. `InvertIntensity()` determines sign based on M/S mask
3. `ISDecode()` reconstructs right channel from scaled left channel
4. Scale factor clamped to [-120, 120] to prevent overflow
5. SFB bounds clamped to SWBOffsetMax
6. Left channel is NOT modified; only right channel is written

---

Plan complete and saved to `docs/plans/2025-12-29-intensity-stereo.md`. Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
