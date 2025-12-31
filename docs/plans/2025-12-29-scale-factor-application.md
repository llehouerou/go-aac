# Scale Factor Application Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement scale factor application to multiply inverse-quantized spectral coefficients by their per-band scale factors.

**Architecture:** Scale factors are applied per scalefactor band (SFB). The formula is `spec[i] *= 2^((sf - SF_OFFSET) / 4)` where SF_OFFSET=100. Intensity stereo and noise (PNS) bands are zeroed during this phase as they are filled later by dedicated tools.

**Tech Stack:** Go, existing `internal/spectrum` and `internal/tables` packages

---

## Background

In FAAD2, scale factor application is combined with inverse quantization in `quant_to_spec()`. In our Go implementation, we separate these concerns:
1. `InverseQuantize()` - applies x^(4/3) (already implemented)
2. `ApplyScaleFactors()` - applies 2^((sf-100)/4) (this task)

The pow2 lookup tables are already ported:
- `tables.Pow2SFTable[64]` - integer powers of 2
- Need: `Pow2FracTable[4]` - fractional powers (2^0, 2^0.25, 2^0.5, 2^0.75)

---

## Task 1: Add Pow2FracTable to tables package

**Files:**
- Modify: `internal/tables/iq_table.go:50-70`
- Test: `internal/tables/iq_table_test.go`

**Step 1: Write the failing test**

Create test in `internal/tables/iq_table_test.go`:

```go
func TestPow2FracTable(t *testing.T) {
	// Expected values from FAAD2 specrec.c:553-559
	expected := [4]float64{
		1.0,                                 // 2^0
		1.1892071150027210667174999705605,   // 2^0.25
		1.4142135623730950488016887242097,   // 2^0.5
		1.6817928305074290860622509524664,   // 2^0.75
	}

	for i, exp := range expected {
		if Pow2FracTable[i] != exp {
			t.Errorf("Pow2FracTable[%d] = %v, want %v", i, Pow2FracTable[i], exp)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tables -run TestPow2FracTable -v`
Expected: FAIL with "undefined: Pow2FracTable"

**Step 3: Write minimal implementation**

Add to `internal/tables/iq_table.go` after `Pow2SFTable`:

```go
// Pow2FracTable contains 2^(i/4) for i in {0,1,2,3}.
// Used for fractional part of scale factor exponent.
//
// Ported from: pow2_table in ~/dev/faad2/libfaad/specrec.c:553-559
var Pow2FracTable = [4]float64{
	1.0,                                 // 2^0
	1.1892071150027210667174999705605,   // 2^0.25
	1.4142135623730950488016887242097,   // 2^0.5
	1.6817928305074290860622509524664,   // 2^0.75
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tables -run TestPow2FracTable -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tables/iq_table.go internal/tables/iq_table_test.go
git commit -m "feat(tables): add Pow2FracTable for scale factor application"
```

---

## Task 2: Add ScaleFactorOffset constant

**Files:**
- Modify: `internal/tables/iq_table.go`

**Step 1: Write the failing test**

Add to `internal/tables/iq_table_test.go`:

```go
func TestScaleFactorOffset(t *testing.T) {
	// From FAAD2: scale factor must be offset by 100 before computing exponent
	if ScaleFactorOffset != 100 {
		t.Errorf("ScaleFactorOffset = %d, want 100", ScaleFactorOffset)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tables -run TestScaleFactorOffset -v`
Expected: FAIL with "undefined: ScaleFactorOffset"

**Step 3: Write minimal implementation**

Add to `internal/tables/iq_table.go`:

```go
// ScaleFactorOffset is subtracted from scale factors before computing the exponent.
// Formula: spec[i] *= 2^((sf - ScaleFactorOffset) / 4)
//
// Ported from: specrec.c:595 "scale_factor -= 100"
const ScaleFactorOffset = 100
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tables -run TestScaleFactorOffset -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tables/iq_table.go internal/tables/iq_table_test.go
git commit -m "feat(tables): add ScaleFactorOffset constant"
```

---

## Task 3: Add helper functions for codebook detection

**Files:**
- Create: `internal/spectrum/helpers.go`
- Test: `internal/spectrum/helpers_test.go`

**Step 1: Write the failing test**

Create `internal/spectrum/helpers_test.go`:

```go
package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/huffman"
)

func TestIsIntensity(t *testing.T) {
	tests := []struct {
		cb       huffman.Codebook
		expected int8
	}{
		{huffman.ZeroHCB, 0},
		{huffman.Codebook(1), 0},
		{huffman.EscHCB, 0},
		{huffman.NoiseHCB, 0},
		{huffman.IntensityHCB, 1},
		{huffman.IntensityHCB2, -1},
	}

	for _, tc := range tests {
		got := IsIntensity(tc.cb)
		if got != tc.expected {
			t.Errorf("IsIntensity(%d) = %d, want %d", tc.cb, got, tc.expected)
		}
	}
}

func TestIsNoise(t *testing.T) {
	tests := []struct {
		cb       huffman.Codebook
		expected bool
	}{
		{huffman.ZeroHCB, false},
		{huffman.Codebook(1), false},
		{huffman.EscHCB, false},
		{huffman.NoiseHCB, true},
		{huffman.IntensityHCB, false},
		{huffman.IntensityHCB2, false},
	}

	for _, tc := range tests {
		got := IsNoise(tc.cb)
		if got != tc.expected {
			t.Errorf("IsNoise(%d) = %v, want %v", tc.cb, got, tc.expected)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run "TestIsIntensity|TestIsNoise" -v`
Expected: FAIL with "undefined: IsIntensity"

**Step 3: Write minimal implementation**

Create `internal/spectrum/helpers.go`:

```go
package spectrum

import "github.com/llehouerou/go-aac/internal/huffman"

// IsIntensity returns the intensity stereo direction for a codebook.
// Returns 1 for in-phase (INTENSITY_HCB), -1 for out-of-phase (INTENSITY_HCB2), 0 otherwise.
//
// Ported from: is_intensity() in ~/dev/faad2/libfaad/is.h:43-54
func IsIntensity(cb huffman.Codebook) int8 {
	switch cb {
	case huffman.IntensityHCB:
		return 1
	case huffman.IntensityHCB2:
		return -1
	default:
		return 0
	}
}

// IsNoise returns true if the codebook indicates a PNS (noise) band.
//
// Ported from: is_noise() in ~/dev/faad2/libfaad/pns.h:47-52
func IsNoise(cb huffman.Codebook) bool {
	return cb == huffman.NoiseHCB
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/spectrum -run "TestIsIntensity|TestIsNoise" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/helpers.go internal/spectrum/helpers_test.go
git commit -m "feat(spectrum): add IsIntensity and IsNoise helper functions"
```

---

## Task 4: Define ApplyScaleFactorsConfig struct

**Files:**
- Create: `internal/spectrum/scalefac.go`
- Test: `internal/spectrum/scalefac_test.go`

**Step 1: Write the config struct**

Create `internal/spectrum/scalefac.go`:

```go
// Package spectrum implements spectral processing for AAC decoding.
package spectrum

import "github.com/llehouerou/go-aac/internal/syntax"

// ApplyScaleFactorsConfig holds configuration for scale factor application.
type ApplyScaleFactorsConfig struct {
	// ICS contains window and scale factor information
	ICS *syntax.ICStream

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16
}
```

**Step 2: Run make check to verify compilation**

Run: `go build ./internal/spectrum`
Expected: PASS (compiles without error)

**Step 3: Commit**

```bash
git add internal/spectrum/scalefac.go
git commit -m "feat(spectrum): add ApplyScaleFactorsConfig struct"
```

---

## Task 5: Implement ApplyScaleFactors for long blocks

**Files:**
- Modify: `internal/spectrum/scalefac.go`
- Test: `internal/spectrum/scalefac_test.go`

**Step 1: Write the failing test**

Create `internal/spectrum/scalefac_test.go`:

```go
package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/huffman"
	"github.com/llehouerou/go-aac/internal/syntax"
	"github.com/llehouerou/go-aac/internal/tables"
)

func TestApplyScaleFactors_LongBlock_SingleSFB(t *testing.T) {
	// Setup: single window group, single SFB covering 4 coefficients
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4

	// Set codebook to a spectral codebook (not noise/intensity)
	ics.SFBCB[0][0] = uint8(huffman.EscHCB)

	// Scale factor = 100 means multiplier = 2^((100-100)/4) = 2^0 = 1.0
	ics.ScaleFactors[0][0] = 100

	// Input: 4 coefficients with value 1.0
	specData := []float64{1.0, 1.0, 1.0, 1.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// With sf=100, multiplier=1.0, output should equal input
	for i, v := range specData {
		if v != 1.0 {
			t.Errorf("specData[%d] = %v, want 1.0", i, v)
		}
	}
}

func TestApplyScaleFactors_LongBlock_ScaleFactor104(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.EscHCB)

	// Scale factor = 104 means multiplier = 2^((104-100)/4) = 2^1 = 2.0
	ics.ScaleFactors[0][0] = 104

	specData := []float64{1.0, 2.0, 3.0, 4.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	expected := []float64{2.0, 4.0, 6.0, 8.0}
	for i, v := range specData {
		if v != expected[i] {
			t.Errorf("specData[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

func TestApplyScaleFactors_LongBlock_ScaleFactor101(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.EscHCB)

	// Scale factor = 101 means multiplier = 2^((101-100)/4) = 2^0.25 ≈ 1.189
	ics.ScaleFactors[0][0] = 101

	specData := []float64{1.0, 1.0, 1.0, 1.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// Pow2FracTable[1] = 2^0.25
	expected := tables.Pow2FracTable[1]
	for i, v := range specData {
		if v != expected {
			t.Errorf("specData[%d] = %v, want %v", i, v, expected)
		}
	}
}

func TestApplyScaleFactors_NoiseCodebook_ZerosOutput(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	ics.ScaleFactors[0][0] = 120 // Should be ignored

	specData := []float64{1.0, 2.0, 3.0, 4.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// Noise bands should be zeroed (filled later by PNS)
	for i, v := range specData {
		if v != 0.0 {
			t.Errorf("specData[%d] = %v, want 0.0 (noise band)", i, v)
		}
	}
}

func TestApplyScaleFactors_IntensityCodebook_ZerosOutput(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.IntensityHCB)
	ics.ScaleFactors[0][0] = 120 // Should be ignored

	specData := []float64{1.0, 2.0, 3.0, 4.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// Intensity bands should be zeroed (filled later by IS)
	for i, v := range specData {
		if v != 0.0 {
			t.Errorf("specData[%d] = %v, want 0.0 (intensity band)", i, v)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run TestApplyScaleFactors -v`
Expected: FAIL with "undefined: ApplyScaleFactors"

**Step 3: Write minimal implementation**

Add to `internal/spectrum/scalefac.go`:

```go
import (
	"github.com/llehouerou/go-aac/internal/huffman"
	"github.com/llehouerou/go-aac/internal/syntax"
	"github.com/llehouerou/go-aac/internal/tables"
)

// ApplyScaleFactors applies scale factors to spectral coefficients in-place.
// For each scalefactor band: spec[i] *= 2^((sf - 100) / 4)
//
// Intensity stereo and noise (PNS) bands are zeroed, as they are filled
// by dedicated tools (is_decode, pns_decode) later in the pipeline.
//
// Ported from: quant_to_spec() scale factor part in ~/dev/faad2/libfaad/specrec.c:549-693
func ApplyScaleFactors(specData []float64, cfg *ApplyScaleFactorsConfig) {
	ics := cfg.ICS

	// Process each window group
	gindex := uint16(0)
	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		// win_inc is the offset between windows within a group
		winInc := ics.SWBOffset[ics.NumSWB]

		// Process each scalefactor band
		j := uint16(0)
		for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
			cb := huffman.Codebook(ics.SFBCB[g][sfb])
			sf := ics.ScaleFactors[g][sfb]

			width := ics.SWBOffset[sfb+1] - ics.SWBOffset[sfb]

			// Intensity stereo and noise bands: zero the coefficients
			// They will be filled later by dedicated tools
			if IsIntensity(cb) != 0 || IsNoise(cb) {
				for win := uint8(0); win < ics.WindowGroupLength[g]; win++ {
					wa := gindex + uint16(win)*winInc + j
					for bin := uint16(0); bin < width; bin++ {
						specData[wa+bin] = 0.0
					}
				}
			} else {
				// Normal spectral band: apply scale factor
				// Formula: spec[i] *= 2^((sf - 100) / 4)
				sfAdjusted := int(sf) - tables.ScaleFactorOffset
				exp := sfAdjusted >> 2        // Integer part of exponent
				frac := sfAdjusted & 3        // Fractional part (0-3)

				// Compute scale: pow2sf_tab[exp+25] * pow2_table[frac]
				// Pow2SFTable index 25 = 1.0 (2^0)
				expIdx := exp + 25
				if expIdx < 0 {
					expIdx = 0
				} else if expIdx >= len(tables.Pow2SFTable) {
					expIdx = len(tables.Pow2SFTable) - 1
				}

				scf := tables.Pow2SFTable[expIdx] * tables.Pow2FracTable[frac]

				// Apply to all windows in this group
				for win := uint8(0); win < ics.WindowGroupLength[g]; win++ {
					wa := gindex + uint16(win)*winInc + j
					for bin := uint16(0); bin < width; bin++ {
						specData[wa+bin] *= scf
					}
				}
			}

			j += width
		}

		// Advance gindex by the total span of this group
		gindex += uint16(ics.WindowGroupLength[g]) * winInc
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/spectrum -run TestApplyScaleFactors -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/scalefac.go internal/spectrum/scalefac_test.go
git commit -m "feat(spectrum): implement ApplyScaleFactors for long blocks"
```

---

## Task 6: Add tests for short block handling

**Files:**
- Modify: `internal/spectrum/scalefac_test.go`

**Step 1: Write the failing test**

Add to `internal/spectrum/scalefac_test.go`:

```go
func TestApplyScaleFactors_ShortBlock(t *testing.T) {
	// Setup: 8 short windows in 1 group
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      8,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.EightShortSequence,
	}
	ics.WindowGroupLength[0] = 8

	// Short block: each window is 128 samples, SFB covers first 4 of each
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4

	ics.SFBCB[0][0] = uint8(huffman.EscHCB)
	ics.ScaleFactors[0][0] = 104 // multiplier = 2.0

	// 8 windows * 4 coefficients = 32 values
	// Interleaved: win0[0-3], win1[0-3], ..., win7[0-3] = 32 total
	// With winInc = 4 (SWBOffset[NumSWB]), data layout is sequential
	specData := make([]float64, 32)
	for i := range specData {
		specData[i] = 1.0
	}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// All values should be multiplied by 2.0
	for i, v := range specData {
		if v != 2.0 {
			t.Errorf("specData[%d] = %v, want 2.0", i, v)
		}
	}
}

func TestApplyScaleFactors_MultipleSFB(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8

	ics.SFBCB[0][0] = uint8(huffman.EscHCB)
	ics.SFBCB[0][1] = uint8(huffman.EscHCB)

	// SFB 0: sf=104 -> mult=2.0
	// SFB 1: sf=108 -> mult=4.0
	ics.ScaleFactors[0][0] = 104
	ics.ScaleFactors[0][1] = 108

	specData := []float64{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	expected := []float64{2.0, 2.0, 2.0, 2.0, 4.0, 4.0, 4.0, 4.0}
	for i, v := range specData {
		if v != expected[i] {
			t.Errorf("specData[%d] = %v, want %v", i, v, expected[i])
		}
	}
}
```

**Step 2: Run tests**

Run: `go test ./internal/spectrum -run TestApplyScaleFactors -v`
Expected: PASS (implementation should already handle these cases)

**Step 3: Commit**

```bash
git add internal/spectrum/scalefac_test.go
git commit -m "test(spectrum): add short block and multi-SFB tests for ApplyScaleFactors"
```

---

## Task 7: Add edge case tests

**Files:**
- Modify: `internal/spectrum/scalefac_test.go`

**Step 1: Write edge case tests**

Add to `internal/spectrum/scalefac_test.go`:

```go
func TestApplyScaleFactors_ZeroCodebook(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.ZeroHCB)
	ics.ScaleFactors[0][0] = 0 // Zero codebook has sf=0

	// Data should be all zeros from spectral decoding, but test anyway
	specData := []float64{0.0, 0.0, 0.0, 0.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	ApplyScaleFactors(specData, cfg)

	// Should remain zero
	for i, v := range specData {
		if v != 0.0 {
			t.Errorf("specData[%d] = %v, want 0.0", i, v)
		}
	}
}

func TestApplyScaleFactors_LargeScaleFactor(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.EscHCB)
	ics.ScaleFactors[0][0] = 255 // Maximum valid scale factor

	specData := []float64{1.0, 1.0, 1.0, 1.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	// Should not panic
	ApplyScaleFactors(specData, cfg)

	// Just verify it runs without error and produces positive values
	for i, v := range specData {
		if v <= 0 {
			t.Errorf("specData[%d] = %v, want positive value", i, v)
		}
	}
}

func TestApplyScaleFactors_SmallScaleFactor(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SFBCB[0][0] = uint8(huffman.EscHCB)
	ics.ScaleFactors[0][0] = 0 // Minimum valid scale factor

	specData := []float64{1.0, 1.0, 1.0, 1.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	// Should not panic
	ApplyScaleFactors(specData, cfg)

	// Output should be very small but positive (2^(-25) * 1 is tiny)
	for i, v := range specData {
		if v <= 0 || v >= 1.0 {
			t.Errorf("specData[%d] = %v, want small positive value", i, v)
		}
	}
}

func TestApplyScaleFactors_EmptyMaxSFB(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          0, // No scalefactor bands used
		NumSWB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4

	specData := []float64{1.0, 1.0, 1.0, 1.0}

	cfg := &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: 1024,
	}

	// Should not modify anything when MaxSFB=0
	ApplyScaleFactors(specData, cfg)

	for i, v := range specData {
		if v != 1.0 {
			t.Errorf("specData[%d] = %v, want 1.0 (unmodified)", i, v)
		}
	}
}
```

**Step 2: Run tests**

Run: `go test ./internal/spectrum -run TestApplyScaleFactors -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/scalefac_test.go
git commit -m "test(spectrum): add edge case tests for ApplyScaleFactors"
```

---

## Task 8: Run full test suite and lint

**Step 1: Run make check**

Run: `make check`
Expected: All format, lint, and tests pass

**Step 2: Fix any issues**

If any issues arise from linting or tests, fix them.

**Step 3: Commit any fixes**

```bash
git add -A
git commit -m "chore: fix lint issues in scale factor implementation"
```

---

## Summary

After completing all tasks, the scale factor application will:

1. **Tables added:**
   - `Pow2FracTable[4]` - fractional powers of 2
   - `ScaleFactorOffset = 100` - constant for the offset

2. **Helpers added:**
   - `IsIntensity(cb)` - returns intensity stereo direction
   - `IsNoise(cb)` - returns true for noise codebook

3. **Main function:**
   - `ApplyScaleFactors(specData, cfg)` - applies scale factors in-place
   - Handles long and short blocks
   - Zeros intensity stereo and noise bands
   - Uses the formula: `spec[i] *= 2^((sf - 100) / 4)`

**Integration point:** This function will be called after `InverseQuantize()` in the spectral reconstruction pipeline.
