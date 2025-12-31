# Dynamic Range Control (DRC) Decoder Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement DRC (Dynamic Range Control) decoding to apply compression/boost gain factors to spectral coefficients.

**Architecture:** DRC parsing is already implemented in `internal/syntax/fill.go`. This plan implements the DRC application logic in `internal/output/drc.go`. DRC applies per-band gain adjustments to spectral data using the formula: `spec[i] *= 2^exp` where exp is computed from the DRC control values and cut/boost parameters.

**Tech Stack:** Pure Go, float32 for spectral processing, math package for power function.

---

## Background

### What DRC Does

Dynamic Range Control (DRC) allows the decoder to adjust the dynamic range of audio for different playback environments:

- **Compression (cut)**: Reduces loud passages for quiet environments (e.g., late-night listening)
- **Boost**: Increases quiet passages for noisy environments (e.g., car audio)

### DRC Data Flow

1. DRC info is parsed from fill elements (already implemented in `internal/syntax/fill.go`)
2. `DRCInfo` struct contains: bands, control values, program reference level
3. `DRC.Decode()` applies gain factors to spectral coefficients before filter bank

### Key Formulas (from FAAD2)

```
DRC_REF_LEVEL = 80  // -20 dB * 4

// Compression (dyn_rng_sgn == 1):
exp = ((-ctrl1 * dyn_rng_ctl) - (DRC_REF_LEVEL - prog_ref_level)) / 24.0

// Boost (dyn_rng_sgn == 0):
exp = ((ctrl2 * dyn_rng_ctl) - (DRC_REF_LEVEL - prog_ref_level)) / 24.0

// Apply:
factor = 2^exp
spec[i] *= factor
```

---

## Task 1: Define DRC Constants

**Files:**
- Create: `internal/output/drc.go`
- Test: `internal/output/drc_test.go`

**Step 1: Write the failing test**

```go
// internal/output/drc_test.go
package output

import "testing"

func TestDRCRefLevel(t *testing.T) {
	// DRC_REF_LEVEL = 20 * 4 = 80 (represents -20 dB)
	// Source: ~/dev/faad2/libfaad/drc.h:38
	if DRCRefLevel != 80 {
		t.Errorf("DRCRefLevel: got %d, want 80", DRCRefLevel)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/output/... -run TestDRCRefLevel -v`
Expected: FAIL with "undefined: DRCRefLevel"

**Step 3: Write minimal implementation**

```go
// internal/output/drc.go
//
// Package output implements PCM output conversion.
//
// This includes sample format conversion, dynamic range control,
// and channel downmixing.
//
// Ported from: ~/dev/faad2/libfaad/output.c, drc.c
package output

// DRCRefLevel is the reference level for DRC calculations.
// Represents -20 dB (20 * 4 = 80 in quarter-dB units).
//
// Ported from: DRC_REF_LEVEL in ~/dev/faad2/libfaad/drc.h:38
const DRCRefLevel = 80
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/output/... -run TestDRCRefLevel -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/drc.go internal/output/drc_test.go
git commit -m "feat(output): add DRC reference level constant"
```

---

## Task 2: Define DRC Type

**Files:**
- Modify: `internal/output/drc.go`
- Modify: `internal/output/drc_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/output/drc_test.go
func TestNewDRC(t *testing.T) {
	drc := NewDRC(0.5, 0.75)

	if drc.Cut != 0.5 {
		t.Errorf("Cut: got %v, want 0.5", drc.Cut)
	}
	if drc.Boost != 0.75 {
		t.Errorf("Boost: got %v, want 0.75", drc.Boost)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/output/... -run TestNewDRC -v`
Expected: FAIL with "undefined: NewDRC"

**Step 3: Write minimal implementation**

```go
// Add to internal/output/drc.go

// DRC holds the Dynamic Range Control state.
//
// Cut and Boost are application-configurable parameters (0.0 to 1.0):
// - Cut: Controls compression (reduces dynamic range)
// - Boost: Controls expansion (increases quiet passages)
//
// Ported from: drc_info in ~/dev/faad2/libfaad/structs.h:85-101
type DRC struct {
	Cut   float32 // Compression control (ctrl1 in FAAD2)
	Boost float32 // Boost control (ctrl2 in FAAD2)
}

// NewDRC creates a new DRC processor with the specified cut and boost factors.
//
// Parameters:
// - cut: Compression factor (0.0 = no compression, 1.0 = full compression)
// - boost: Boost factor (0.0 = no boost, 1.0 = full boost)
//
// Ported from: drc_init() in ~/dev/faad2/libfaad/drc.c:38-52
func NewDRC(cut, boost float32) *DRC {
	return &DRC{
		Cut:   cut,
		Boost: boost,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/output/... -run TestNewDRC -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/drc.go internal/output/drc_test.go
git commit -m "feat(output): add DRC type and constructor"
```

---

## Task 3: Implement DRC Decode for Single Band

**Files:**
- Modify: `internal/output/drc.go`
- Modify: `internal/output/drc_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/output/drc_test.go
import (
	"math"
	"testing"

	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestDRCDecode_SingleBand_NoGain(t *testing.T) {
	// DRC with default values should apply no gain
	drc := NewDRC(0.0, 0.0)

	info := &syntax.DRCInfo{
		NumBands:     1,
		ProgRefLevel: DRCRefLevel, // Same as reference = no adjustment
		DynRngSgn:    [17]uint8{0},
		DynRngCtl:    [17]uint8{0},
	}
	info.BandTop[0] = 1024/4 - 1 // Default: entire frame

	// Input: some spectral coefficients
	spec := []float32{1.0, 2.0, 3.0, 4.0}

	drc.Decode(info, spec)

	// With zero control values and prog_ref_level == DRC_REF_LEVEL,
	// the exponent is 0, so factor = 2^0 = 1.0 (no change)
	expected := []float32{1.0, 2.0, 3.0, 4.0}
	for i, v := range spec {
		if math.Abs(float64(v-expected[i])) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want %v", i, v, expected[i])
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/output/... -run TestDRCDecode_SingleBand_NoGain -v`
Expected: FAIL with "drc.Decode undefined"

**Step 3: Write minimal implementation**

```go
// Add to internal/output/drc.go
import (
	"math"

	"github.com/llehouerou/go-aac/internal/syntax"
)

// Decode applies Dynamic Range Control to spectral coefficients.
//
// The DRC info is parsed from fill elements in the bitstream.
// This function modifies spec in-place.
//
// Ported from: drc_decode() in ~/dev/faad2/libfaad/drc.c:112-172
func (d *DRC) Decode(info *syntax.DRCInfo, spec []float32) {
	if info == nil || info.NumBands == 0 {
		return
	}

	bottom := uint16(0)
	numBands := info.NumBands

	// Default band_top for single band
	if numBands == 1 {
		info.BandTop[0] = 1024/4 - 1
	}

	for bd := uint8(0); bd < numBands; bd++ {
		top := uint16(4 * (uint16(info.BandTop[bd]) + 1))

		// Clamp top to spec length
		if int(top) > len(spec) {
			top = uint16(len(spec))
		}

		// Decode DRC gain factor
		var exp float32
		if info.DynRngSgn[bd] == 1 {
			// Compress
			exp = ((-d.Cut * float32(info.DynRngCtl[bd])) -
				float32(DRCRefLevel-int(info.ProgRefLevel))) / 24.0
		} else {
			// Boost
			exp = ((d.Boost * float32(info.DynRngCtl[bd])) -
				float32(DRCRefLevel-int(info.ProgRefLevel))) / 24.0
		}

		factor := float32(math.Pow(2.0, float64(exp)))

		// Apply gain factor
		for i := bottom; i < top; i++ {
			spec[i] *= factor
		}

		bottom = top
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/output/... -run TestDRCDecode_SingleBand_NoGain -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/drc.go internal/output/drc_test.go
git commit -m "feat(output): implement DRC.Decode for single band"
```

---

## Task 4: Test DRC Decode Compression (Cut)

**Files:**
- Modify: `internal/output/drc_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/output/drc_test.go
func TestDRCDecode_Compression(t *testing.T) {
	// Full cut (1.0) with max control value
	drc := NewDRC(1.0, 0.0)

	info := &syntax.DRCInfo{
		NumBands:     1,
		ProgRefLevel: DRCRefLevel, // No level adjustment
		DynRngSgn:    [17]uint8{1}, // Compress
		DynRngCtl:    [17]uint8{24}, // 24 quarter-dB
	}
	info.BandTop[0] = 1024/4 - 1

	// Input
	spec := []float32{1.0, 1.0, 1.0, 1.0}

	drc.Decode(info, spec)

	// exp = (-1.0 * 24 - 0) / 24 = -1.0
	// factor = 2^(-1) = 0.5
	expected := float32(0.5)
	for i, v := range spec {
		if math.Abs(float64(v-expected)) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want %v", i, v, expected)
		}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/output/... -run TestDRCDecode_Compression -v`
Expected: PASS (implementation already handles this)

**Step 3: Commit**

```bash
git add internal/output/drc_test.go
git commit -m "test(output): add DRC compression test"
```

---

## Task 5: Test DRC Decode Boost

**Files:**
- Modify: `internal/output/drc_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/output/drc_test.go
func TestDRCDecode_Boost(t *testing.T) {
	// Full boost (1.0) with max control value
	drc := NewDRC(0.0, 1.0)

	info := &syntax.DRCInfo{
		NumBands:     1,
		ProgRefLevel: DRCRefLevel, // No level adjustment
		DynRngSgn:    [17]uint8{0}, // Boost
		DynRngCtl:    [17]uint8{24}, // 24 quarter-dB
	}
	info.BandTop[0] = 1024/4 - 1

	// Input
	spec := []float32{1.0, 1.0, 1.0, 1.0}

	drc.Decode(info, spec)

	// exp = (1.0 * 24 - 0) / 24 = 1.0
	// factor = 2^1 = 2.0
	expected := float32(2.0)
	for i, v := range spec {
		if math.Abs(float64(v-expected)) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want %v", i, v, expected)
		}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/output/... -run TestDRCDecode_Boost -v`
Expected: PASS (implementation already handles this)

**Step 3: Commit**

```bash
git add internal/output/drc_test.go
git commit -m "test(output): add DRC boost test"
```

---

## Task 6: Test DRC with Program Reference Level Adjustment

**Files:**
- Modify: `internal/output/drc_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/output/drc_test.go
func TestDRCDecode_ProgRefLevelAdjustment(t *testing.T) {
	// Test with prog_ref_level different from DRC_REF_LEVEL
	drc := NewDRC(1.0, 1.0)

	info := &syntax.DRCInfo{
		NumBands:     1,
		ProgRefLevel: 56, // -14 dB (56/4 = 14), differs from -20 dB
		DynRngSgn:    [17]uint8{0}, // Boost
		DynRngCtl:    [17]uint8{0}, // No control signal
	}
	info.BandTop[0] = 1024/4 - 1

	// Input
	spec := []float32{1.0, 1.0, 1.0, 1.0}

	drc.Decode(info, spec)

	// exp = (1.0 * 0 - (80 - 56)) / 24 = -24/24 = -1.0
	// factor = 2^(-1) = 0.5
	expected := float32(0.5)
	for i, v := range spec {
		if math.Abs(float64(v-expected)) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want %v", i, v, expected)
		}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/output/... -run TestDRCDecode_ProgRefLevelAdjustment -v`
Expected: PASS (implementation already handles this)

**Step 3: Commit**

```bash
git add internal/output/drc_test.go
git commit -m "test(output): add DRC program reference level test"
```

---

## Task 7: Test DRC with Multiple Bands

**Files:**
- Modify: `internal/output/drc_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/output/drc_test.go
func TestDRCDecode_MultipleBands(t *testing.T) {
	drc := NewDRC(1.0, 1.0)

	// Two bands with different settings
	info := &syntax.DRCInfo{
		NumBands:     2,
		ProgRefLevel: DRCRefLevel,
		DynRngSgn:    [17]uint8{1, 0}, // Band 0: compress, Band 1: boost
		DynRngCtl:    [17]uint8{24, 24}, // Same control value
	}
	// Band 0: samples 0-3 (top=0 means 4 samples: 4*(0+1))
	// Band 1: samples 4-7 (top=1 means 8 samples: 4*(1+1))
	info.BandTop[0] = 0 // Covers samples 0-3
	info.BandTop[1] = 1 // Covers samples 4-7

	// Input: 8 samples
	spec := []float32{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0}

	drc.Decode(info, spec)

	// Band 0: compress, factor = 0.5
	for i := 0; i < 4; i++ {
		if math.Abs(float64(spec[i]-0.5)) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want 0.5", i, spec[i])
		}
	}

	// Band 1: boost, factor = 2.0
	for i := 4; i < 8; i++ {
		if math.Abs(float64(spec[i]-2.0)) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want 2.0", i, spec[i])
		}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/output/... -run TestDRCDecode_MultipleBands -v`
Expected: PASS (implementation already handles this)

**Step 3: Commit**

```bash
git add internal/output/drc_test.go
git commit -m "test(output): add DRC multiple bands test"
```

---

## Task 8: Test DRC with Nil Info

**Files:**
- Modify: `internal/output/drc_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/output/drc_test.go
func TestDRCDecode_NilInfo(t *testing.T) {
	drc := NewDRC(1.0, 1.0)

	// Input should not be modified
	spec := []float32{1.0, 2.0, 3.0, 4.0}
	original := make([]float32, len(spec))
	copy(original, spec)

	// Should not panic
	drc.Decode(nil, spec)

	// Should be unchanged
	for i, v := range spec {
		if v != original[i] {
			t.Errorf("spec[%d]: got %v, want %v", i, v, original[i])
		}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/output/... -run TestDRCDecode_NilInfo -v`
Expected: PASS (implementation already handles nil)

**Step 3: Commit**

```bash
git add internal/output/drc_test.go
git commit -m "test(output): add DRC nil info test"
```

---

## Task 9: Test DRC with Zero Bands

**Files:**
- Modify: `internal/output/drc_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/output/drc_test.go
func TestDRCDecode_ZeroBands(t *testing.T) {
	drc := NewDRC(1.0, 1.0)

	info := &syntax.DRCInfo{
		NumBands: 0, // No bands
	}

	// Input should not be modified
	spec := []float32{1.0, 2.0, 3.0, 4.0}
	original := make([]float32, len(spec))
	copy(original, spec)

	drc.Decode(info, spec)

	// Should be unchanged
	for i, v := range spec {
		if v != original[i] {
			t.Errorf("spec[%d]: got %v, want %v", i, v, original[i])
		}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/output/... -run TestDRCDecode_ZeroBands -v`
Expected: PASS (implementation already handles zero bands)

**Step 3: Commit**

```bash
git add internal/output/drc_test.go
git commit -m "test(output): add DRC zero bands test"
```

---

## Task 10: Test DRC with Short Spec Array

**Files:**
- Modify: `internal/output/drc_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/output/drc_test.go
func TestDRCDecode_ShortSpec(t *testing.T) {
	// Test that DRC correctly handles spec arrays shorter than band_top suggests
	drc := NewDRC(1.0, 1.0)

	info := &syntax.DRCInfo{
		NumBands:     1,
		ProgRefLevel: DRCRefLevel,
		DynRngSgn:    [17]uint8{0}, // Boost
		DynRngCtl:    [17]uint8{24},
	}
	info.BandTop[0] = 255 // Would suggest 1024 samples

	// But we only have 4 samples
	spec := []float32{1.0, 1.0, 1.0, 1.0}

	// Should not panic, should only process available samples
	drc.Decode(info, spec)

	// All 4 samples should be boosted
	expected := float32(2.0)
	for i, v := range spec {
		if math.Abs(float64(v-expected)) > 1e-6 {
			t.Errorf("spec[%d]: got %v, want %v", i, v, expected)
		}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/output/... -run TestDRCDecode_ShortSpec -v`
Expected: PASS (implementation clamps top to spec length)

**Step 3: Commit**

```bash
git add internal/output/drc_test.go
git commit -m "test(output): add DRC short spec array test"
```

---

## Task 11: Run Full Test Suite and Verify

**Files:**
- None (verification only)

**Step 1: Run all DRC tests**

Run: `go test ./internal/output/... -v`
Expected: All tests PASS

**Step 2: Run linter**

Run: `make lint`
Expected: No errors

**Step 3: Run full check**

Run: `make check`
Expected: All checks pass

**Step 4: Commit final state**

```bash
git add -A
git commit -m "feat(output): complete DRC decoder implementation"
```

---

## Task 12: Delete doc.go Placeholder

**Files:**
- Delete: `internal/output/doc.go`

Since drc.go now contains the package documentation comment, the doc.go placeholder is redundant.

**Step 1: Verify drc.go has package doc**

Check that `internal/output/drc.go` starts with the package documentation comment.

**Step 2: Delete doc.go**

Run: `rm internal/output/doc.go`

**Step 3: Verify tests still pass**

Run: `go test ./internal/output/... -v`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add -A
git commit -m "chore(output): remove redundant doc.go"
```

---

## Summary

After completing all tasks, the DRC implementation will include:

1. **`internal/output/drc.go`**:
   - `DRCRefLevel` constant (80, representing -20 dB)
   - `DRC` struct with Cut and Boost fields
   - `NewDRC()` constructor
   - `Decode()` method for applying DRC to spectral coefficients

2. **`internal/output/drc_test.go`**:
   - Tests for constant values
   - Tests for constructor
   - Tests for single-band decode (no gain, compression, boost)
   - Tests for program reference level adjustment
   - Tests for multiple bands
   - Tests for edge cases (nil info, zero bands, short spec)

The implementation follows FAAD2's floating-point code path (not the fixed-point path) for simplicity and matches the existing float32 spectral processing in go-aac.
