# TNS Decoder Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement Temporal Noise Shaping (TNS) decoding to reduce pre-echo artifacts in AAC audio.

**Architecture:** TNS applies an all-pole IIR filter to spectral coefficients. Transmitted filter coefficients are converted to LPC coefficients via Levinson-Durbin recursion, then applied as an AR filter in either forward or backward direction. Uses a double ringbuffer for efficient state management.

**Tech Stack:** Pure Go, float64 for spectral processing, follows existing spectrum package patterns.

---

## Background

TNS (Temporal Noise Shaping) is an AAC coding tool that shapes quantization noise in the time domain by filtering the spectral coefficients. At the encoder, an MA (moving average) filter is applied. The decoder reverses this by applying the inverse AR (autoregressive) filter.

**Key FAAD2 source files:**
- `~/dev/faad2/libfaad/tns.c` (339 lines)
- `~/dev/faad2/libfaad/tns.h` (51 lines)

**Existing Go code to use:**
- `internal/syntax/tns.go` - TNSInfo struct and ParseTNSData (already implemented)
- `internal/tables/sample_rates.go` - MaxTNSSFB function (already implemented)
- `internal/syntax/ics.go` - ICStream with TNSDataPresent and TNS fields

---

## Task 1: Add TNS Coefficient Tables

**Files:**
- Create: `internal/spectrum/tns_tables.go`
- Test: `internal/spectrum/tns_tables_test.go`

**Step 1: Write the failing test**

```go
// internal/spectrum/tns_tables_test.go
package spectrum

import (
	"math"
	"testing"
)

func TestTNSCoefTables_Length(t *testing.T) {
	// Each table should have 16 entries
	if len(tnsCoef03) != 16 {
		t.Errorf("tnsCoef03: got %d entries, want 16", len(tnsCoef03))
	}
	if len(tnsCoef04) != 16 {
		t.Errorf("tnsCoef04: got %d entries, want 16", len(tnsCoef04))
	}
	if len(tnsCoef13) != 16 {
		t.Errorf("tnsCoef13: got %d entries, want 16", len(tnsCoef13))
	}
	if len(tnsCoef14) != 16 {
		t.Errorf("tnsCoef14: got %d entries, want 16", len(tnsCoef14))
	}
}

func TestTNSCoefTables_Values(t *testing.T) {
	// Verify first values from FAAD2
	const tolerance = 1e-9

	// tns_coef_0_3[0] = 0.0
	if math.Abs(tnsCoef03[0]-0.0) > tolerance {
		t.Errorf("tnsCoef03[0]: got %v, want 0.0", tnsCoef03[0])
	}
	// tns_coef_0_3[1] = 0.4338837391
	if math.Abs(tnsCoef03[1]-0.4338837391) > tolerance {
		t.Errorf("tnsCoef03[1]: got %v, want 0.4338837391", tnsCoef03[1])
	}
	// tns_coef_0_4[7] = 0.9945218954
	if math.Abs(tnsCoef04[7]-0.9945218954) > tolerance {
		t.Errorf("tnsCoef04[7]: got %v, want 0.9945218954", tnsCoef04[7])
	}
}

func TestGetTNSCoefTable(t *testing.T) {
	tests := []struct {
		coefCompress uint8
		coefRes      uint8
		wantTable    *[16]float64
	}{
		{0, 0, &tnsCoef03}, // coef_compress=0, coef_res_bits=3
		{0, 1, &tnsCoef04}, // coef_compress=0, coef_res_bits=4
		{1, 0, &tnsCoef13}, // coef_compress=1, coef_res_bits=3
		{1, 1, &tnsCoef14}, // coef_compress=1, coef_res_bits=4
	}

	for _, tc := range tests {
		got := getTNSCoefTable(tc.coefCompress, tc.coefRes)
		if got != tc.wantTable {
			t.Errorf("getTNSCoefTable(%d, %d): got wrong table", tc.coefCompress, tc.coefRes)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run TestTNSCoef -v`
Expected: FAIL with "undefined: tnsCoef03"

**Step 3: Write minimal implementation**

```go
// internal/spectrum/tns_tables.go

// Package spectrum implements spectral processing for AAC decoding.
package spectrum

// TNSMaxOrder is the maximum TNS filter order.
// Ported from: TNS_MAX_ORDER in ~/dev/faad2/libfaad/tns.h:39
const TNSMaxOrder = 20

// TNS coefficient lookup tables.
// These tables convert transmitted coefficient indices to actual filter coefficient values.
// The table selection depends on coef_compress and coef_res_bits (coefficient resolution).
//
// Ported from: tns_coef_0_3, tns_coef_0_4, tns_coef_1_3, tns_coef_1_4
// in ~/dev/faad2/libfaad/tns.c:52-79

// tnsCoef03 is used when coef_compress=0 and coef_res_bits=3
var tnsCoef03 = [16]float64{
	0.0, 0.4338837391, 0.7818314825, 0.9749279122,
	-0.9848077530, -0.8660254038, -0.6427876097, -0.3420201433,
	-0.4338837391, -0.7818314825, -0.9749279122, -0.9749279122,
	-0.9848077530, -0.8660254038, -0.6427876097, -0.3420201433,
}

// tnsCoef04 is used when coef_compress=0 and coef_res_bits=4
var tnsCoef04 = [16]float64{
	0.0, 0.2079116908, 0.4067366431, 0.5877852523,
	0.7431448255, 0.8660254038, 0.9510565163, 0.9945218954,
	-0.9957341763, -0.9618256432, -0.8951632914, -0.7980172273,
	-0.6736956436, -0.5264321629, -0.3612416662, -0.1837495178,
}

// tnsCoef13 is used when coef_compress=1 and coef_res_bits=3
var tnsCoef13 = [16]float64{
	0.0, 0.4338837391, -0.6427876097, -0.3420201433,
	0.9749279122, 0.7818314825, -0.6427876097, -0.3420201433,
	-0.4338837391, -0.7818314825, -0.6427876097, -0.3420201433,
	-0.7818314825, -0.4338837391, -0.6427876097, -0.3420201433,
}

// tnsCoef14 is used when coef_compress=1 and coef_res_bits=4
var tnsCoef14 = [16]float64{
	0.0, 0.2079116908, 0.4067366431, 0.5877852523,
	-0.6736956436, -0.5264321629, -0.3612416662, -0.1837495178,
	0.9945218954, 0.9510565163, 0.8660254038, 0.7431448255,
	-0.6736956436, -0.5264321629, -0.3612416662, -0.1837495178,
}

// allTNSCoefs provides indexed access to coefficient tables.
// Index = 2*coef_compress + (coef_res_bits != 3 ? 1 : 0)
// Ported from: all_tns_coefs in ~/dev/faad2/libfaad/tns.c:81
var allTNSCoefs = [4]*[16]float64{
	&tnsCoef03, // index 0: compress=0, res=3
	&tnsCoef04, // index 1: compress=0, res=4
	&tnsCoef13, // index 2: compress=1, res=3
	&tnsCoef14, // index 3: compress=1, res=4
}

// getTNSCoefTable returns the appropriate coefficient table.
// coefCompress: 0 or 1 (from bitstream)
// coefRes: coef_res field from bitstream (0 means 3-bit, 1 means 4-bit)
//
// Ported from: table_index calculation in ~/dev/faad2/libfaad/tns.c:199
func getTNSCoefTable(coefCompress uint8, coefRes uint8) *[16]float64 {
	// In FAAD2: table_index = 2 * (coef_compress != 0) + (coef_res_bits != 3)
	// coef_res_bits = coef_res + 3, so (coef_res_bits != 3) == (coef_res != 0)
	index := 0
	if coefCompress != 0 {
		index = 2
	}
	if coefRes != 0 {
		index++
	}
	return allTNSCoefs[index]
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/spectrum -run TestTNSCoef -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/tns_tables.go internal/spectrum/tns_tables_test.go
git commit -m "$(cat <<'EOF'
feat(spectrum): add TNS coefficient lookup tables

Add the four TNS coefficient tables (tnsCoef03, tnsCoef04, tnsCoef13,
tnsCoef14) and getTNSCoefTable() selector function. Tables are copied
exactly from FAAD2 tns.c:52-81.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Implement tns_decode_coef (LPC Coefficient Conversion)

**Files:**
- Create: `internal/spectrum/tns.go`
- Test: `internal/spectrum/tns_test.go`

**Step 1: Write the failing test**

```go
// internal/spectrum/tns_test.go
package spectrum

import (
	"math"
	"testing"
)

func TestTNSDecodeCoef_Order1(t *testing.T) {
	// Simple case: order=1, coefRes=0 (3-bit), compress=0, coef[0]=1
	coef := [32]uint8{1}
	lpc := make([]float64, TNSMaxOrder+1)

	tnsDecodeCoef(1, 0, 0, coef[:], lpc)

	// lpc[0] should always be 1.0
	if lpc[0] != 1.0 {
		t.Errorf("lpc[0]: got %v, want 1.0", lpc[0])
	}

	// lpc[1] should be tnsCoef03[1] = 0.4338837391
	const tolerance = 1e-9
	if math.Abs(lpc[1]-0.4338837391) > tolerance {
		t.Errorf("lpc[1]: got %v, want 0.4338837391", lpc[1])
	}
}

func TestTNSDecodeCoef_Order2(t *testing.T) {
	// Order=2 to verify Levinson-Durbin recursion
	coef := [32]uint8{1, 2} // indices into table
	lpc := make([]float64, TNSMaxOrder+1)

	tnsDecodeCoef(2, 0, 0, coef[:], lpc)

	// lpc[0] should always be 1.0
	if lpc[0] != 1.0 {
		t.Errorf("lpc[0]: got %v, want 1.0", lpc[0])
	}

	// Verify non-zero coefficients were computed
	if lpc[1] == 0.0 && lpc[2] == 0.0 {
		t.Error("lpc coefficients should be non-zero for order 2")
	}
}

func TestTNSDecodeCoef_Order0(t *testing.T) {
	// Edge case: order=0 should just set lpc[0]=1.0
	coef := [32]uint8{}
	lpc := make([]float64, TNSMaxOrder+1)

	tnsDecodeCoef(0, 0, 0, coef[:], lpc)

	if lpc[0] != 1.0 {
		t.Errorf("lpc[0]: got %v, want 1.0", lpc[0])
	}
}

func TestTNSDecodeCoef_DifferentTables(t *testing.T) {
	// Test that different coefRes/compress values select different tables
	coef := [32]uint8{1}
	lpc03 := make([]float64, TNSMaxOrder+1)
	lpc04 := make([]float64, TNSMaxOrder+1)

	tnsDecodeCoef(1, 0, 0, coef[:], lpc03) // 3-bit, no compress
	tnsDecodeCoef(1, 1, 0, coef[:], lpc04) // 4-bit, no compress

	// Different tables should produce different results
	if lpc03[1] == lpc04[1] {
		t.Errorf("Different tables should produce different lpc[1]: both got %v", lpc03[1])
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run TestTNSDecodeCoef -v`
Expected: FAIL with "undefined: tnsDecodeCoef"

**Step 3: Write minimal implementation**

```go
// internal/spectrum/tns.go

package spectrum

// tnsDecodeCoef converts transmitted TNS coefficients to LPC filter coefficients.
// Uses Levinson-Durbin recursion to convert reflection coefficients to direct form.
//
// Parameters:
//   - order: filter order (0-20)
//   - coefRes: coefficient resolution (0=3-bit, 1=4-bit)
//   - coefCompress: compression flag (0 or 1)
//   - coef: transmitted coefficient indices
//   - lpc: output LPC coefficients (must be len >= order+1)
//
// Ported from: tns_decode_coef() in ~/dev/faad2/libfaad/tns.c:193-242
func tnsDecodeCoef(order uint8, coefRes uint8, coefCompress uint8, coef []uint8, lpc []float64) {
	// Get the appropriate coefficient table
	tnsCoef := getTNSCoefTable(coefCompress, coefRes)

	// Convert transmitted indices to coefficient values
	tmp2 := make([]float64, TNSMaxOrder+1)
	for i := uint8(0); i < order; i++ {
		tmp2[i] = tnsCoef[coef[i]]
	}

	// Levinson-Durbin recursion to convert reflection coefficients to LPC
	// a[0] is always 1.0
	lpc[0] = 1.0

	b := make([]float64, TNSMaxOrder+1)
	for m := uint8(1); m <= order; m++ {
		// Set a[m] = reflection coefficient
		lpc[m] = tmp2[m-1]

		// Update previous coefficients
		for i := uint8(1); i < m; i++ {
			b[i] = lpc[i] + lpc[m]*lpc[m-i]
		}
		for i := uint8(1); i < m; i++ {
			lpc[i] = b[i]
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/spectrum -run TestTNSDecodeCoef -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/tns.go internal/spectrum/tns_test.go
git commit -m "$(cat <<'EOF'
feat(spectrum): add tnsDecodeCoef for LPC coefficient conversion

Implements Levinson-Durbin recursion to convert transmitted TNS
reflection coefficients to direct-form LPC coefficients.

Ported from: tns_decode_coef() in ~/dev/faad2/libfaad/tns.c:193-242

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Implement tns_ar_filter (All-Pole IIR Filter)

**Files:**
- Modify: `internal/spectrum/tns.go`
- Modify: `internal/spectrum/tns_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/spectrum/tns_test.go

func TestTNSARFilter_Identity(t *testing.T) {
	// When lpc = [1, 0, 0, ...], filter should be identity (y = x)
	lpc := make([]float64, TNSMaxOrder+1)
	lpc[0] = 1.0

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	spec := make([]float64, len(input))
	copy(spec, input)

	tnsARFilter(spec, int16(len(spec)), 1, lpc, 0)

	for i, want := range input {
		if spec[i] != want {
			t.Errorf("spec[%d]: got %v, want %v", i, spec[i], want)
		}
	}
}

func TestTNSARFilter_Order1(t *testing.T) {
	// Simple first-order filter: y[n] = x[n] - 0.5*y[n-1]
	lpc := make([]float64, TNSMaxOrder+1)
	lpc[0] = 1.0
	lpc[1] = 0.5

	spec := []float64{1.0, 0.0, 0.0, 0.0, 0.0}

	tnsARFilter(spec, int16(len(spec)), 1, lpc, 1)

	// y[0] = 1.0 - 0.5*0 = 1.0
	// y[1] = 0.0 - 0.5*1.0 = -0.5
	// y[2] = 0.0 - 0.5*(-0.5) = 0.25
	// y[3] = 0.0 - 0.5*0.25 = -0.125
	// y[4] = 0.0 - 0.5*(-0.125) = 0.0625

	expected := []float64{1.0, -0.5, 0.25, -0.125, 0.0625}
	const tolerance = 1e-9

	for i, want := range expected {
		if math.Abs(spec[i]-want) > tolerance {
			t.Errorf("spec[%d]: got %v, want %v", i, spec[i], want)
		}
	}
}

func TestTNSARFilter_Backward(t *testing.T) {
	// Test backward filtering (inc = -1)
	lpc := make([]float64, TNSMaxOrder+1)
	lpc[0] = 1.0
	lpc[1] = 0.5

	spec := []float64{0.0, 0.0, 0.0, 0.0, 1.0}

	// Start at spec[4], go backward
	tnsARFilter(spec[4:], 5, -1, lpc, 1)

	// Should filter from end to start
	// y[4] = 1.0 - 0.5*0 = 1.0
	// y[3] = 0.0 - 0.5*1.0 = -0.5
	// y[2] = 0.0 - 0.5*(-0.5) = 0.25
	// etc.

	expected := []float64{0.0625, -0.125, 0.25, -0.5, 1.0}
	const tolerance = 1e-9

	for i, want := range expected {
		if math.Abs(spec[i]-want) > tolerance {
			t.Errorf("spec[%d]: got %v, want %v", i, spec[i], want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run TestTNSARFilter -v`
Expected: FAIL with "undefined: tnsARFilter"

**Step 3: Write minimal implementation**

```go
// Add to internal/spectrum/tns.go

// tnsARFilter applies an all-pole (AR) IIR filter to spectral coefficients.
// This is the core TNS decoding filter operation.
//
// The filter is defined by:
//   y[n] = x[n] - lpc[1]*y[n-1] - lpc[2]*y[n-2] - ... - lpc[order]*y[n-order]
//
// Parameters:
//   - spectrum: spectral data to filter (modified in-place)
//   - size: number of samples to filter
//   - inc: direction (+1 for forward, -1 for backward)
//   - lpc: LPC filter coefficients (lpc[0] is always 1.0)
//   - order: filter order
//
// Uses a double ringbuffer for efficient state management.
//
// Ported from: tns_ar_filter() in ~/dev/faad2/libfaad/tns.c:244-293
func tnsARFilter(spectrum []float64, size int16, inc int8, lpc []float64, order uint8) {
	if size <= 0 || order == 0 {
		return
	}

	// State is stored as a double ringbuffer for efficient wraparound
	state := make([]float64, 2*TNSMaxOrder)
	stateIndex := int8(0)

	// Process each sample
	idx := 0
	for i := int16(0); i < size; i++ {
		// Compute filter output: y = x - sum(lpc[j+1] * state[j])
		y := 0.0
		for j := uint8(0); j < order; j++ {
			y += state[int(stateIndex)+int(j)] * lpc[j+1]
		}
		y = spectrum[idx] - y

		// Update double ringbuffer state
		stateIndex--
		if stateIndex < 0 {
			stateIndex = int8(order - 1)
		}
		state[stateIndex] = y
		state[int(stateIndex)+int(order)] = y

		// Write output and advance
		spectrum[idx] = y
		idx += int(inc)
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/spectrum -run TestTNSARFilter -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/tns.go internal/spectrum/tns_test.go
git commit -m "$(cat <<'EOF'
feat(spectrum): add tnsARFilter for all-pole IIR filtering

Implements the TNS decoding filter using a double ringbuffer for
efficient state management. Supports both forward and backward
filtering directions.

Ported from: tns_ar_filter() in ~/dev/faad2/libfaad/tns.c:244-293

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Implement TNSDecodeFrame (Main Entry Point)

**Files:**
- Modify: `internal/spectrum/tns.go`
- Modify: `internal/spectrum/tns_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/spectrum/tns_test.go

func TestTNSDecodeFrame_NoTNSData(t *testing.T) {
	// When tns_data_present is false, spectrum should be unchanged
	ics := &syntax.ICStream{
		TNSDataPresent: false,
	}

	original := []float64{1.0, 2.0, 3.0, 4.0}
	spec := make([]float64, len(original))
	copy(spec, original)

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4, // 44100 Hz
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	for i, want := range original {
		if spec[i] != want {
			t.Errorf("spec[%d]: got %v, want %v (should be unchanged)", i, spec[i], want)
		}
	}
}

func TestTNSDecodeFrame_SingleFilter(t *testing.T) {
	// Test with a single TNS filter on long block
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        1,
		NumWindowGroups:   1,
		WindowSequence:    syntax.OnlyLongSequence,
		NumSWB:            49,
		MaxSFB:            49,
		SWBOffsetMax:      1024,
		WindowGroupLength: [8]uint8{1},
	}

	// Set up SWB offsets (simplified - just use linear)
	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
		if ics.SWBOffset[i] > 1024 {
			ics.SWBOffset[i] = 1024
		}
	}

	// Set up TNS filter
	ics.TNS.NFilt[0] = 1
	ics.TNS.CoefRes[0] = 0      // 3-bit coefficients
	ics.TNS.Length[0][0] = 20   // Filter spans 20 SFBs
	ics.TNS.Order[0][0] = 1     // First-order filter
	ics.TNS.Direction[0][0] = 0 // Forward
	ics.TNS.CoefCompress[0][0] = 0
	ics.TNS.Coef[0][0][0] = 0 // Index 0 = coefficient 0.0

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = 1.0
	}

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4, // 44100 Hz
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	// With coefficient 0.0, filter should be identity (no change)
	// Just verify it doesn't crash and runs
	for i := range spec {
		if math.IsNaN(spec[i]) || math.IsInf(spec[i], 0) {
			t.Errorf("spec[%d] is invalid: %v", i, spec[i])
		}
	}
}

func TestTNSDecodeFrame_ShortBlock(t *testing.T) {
	// Test with 8 short windows
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        8,
		NumWindowGroups:   8,
		WindowSequence:    syntax.EightShortSequence,
		NumSWB:            14,
		MaxSFB:            14,
		SWBOffsetMax:      128,
		WindowGroupLength: [8]uint8{1, 1, 1, 1, 1, 1, 1, 1},
	}

	// Set up SWB offsets for short blocks
	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 8)
		if ics.SWBOffset[i] > 128 {
			ics.SWBOffset[i] = 128
		}
	}

	// Set up TNS filter for first window only
	ics.TNS.NFilt[0] = 1
	ics.TNS.CoefRes[0] = 0
	ics.TNS.Length[0][0] = 10
	ics.TNS.Order[0][0] = 1
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefCompress[0][0] = 0
	ics.TNS.Coef[0][0][0] = 0

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = 1.0
	}

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	// Verify no crashes and valid output
	for i := range spec {
		if math.IsNaN(spec[i]) || math.IsInf(spec[i], 0) {
			t.Errorf("spec[%d] is invalid: %v", i, spec[i])
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run TestTNSDecodeFrame -v`
Expected: FAIL with "undefined: TNSDecodeConfig" or "undefined: TNSDecodeFrame"

**Step 3: Write minimal implementation**

```go
// Add to internal/spectrum/tns.go

import (
	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/syntax"
	"github.com/llehouerou/go-aac/internal/tables"
)

// TNSDecodeConfig holds configuration for TNS decoding.
type TNSDecodeConfig struct {
	// ICS is the individual channel stream containing TNS data
	ICS *syntax.ICStream

	// SRIndex is the sample rate index (0-15)
	SRIndex uint8

	// ObjectType is the AAC object type
	ObjectType aac.ObjectType

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16
}

// TNSDecodeFrame applies TNS (Temporal Noise Shaping) decoding to one channel.
// TNS applies all-pole IIR filters to spectral coefficients to shape
// the temporal envelope of quantization noise.
//
// Ported from: tns_decode_frame() in ~/dev/faad2/libfaad/tns.c:84-136
func TNSDecodeFrame(spec []float64, cfg *TNSDecodeConfig) {
	ics := cfg.ICS

	if !ics.TNSDataPresent {
		return
	}

	tns := &ics.TNS
	nshort := cfg.FrameLength / 8
	isShort := ics.WindowSequence == syntax.EightShortSequence

	lpc := make([]float64, TNSMaxOrder+1)

	for w := uint8(0); w < ics.NumWindows; w++ {
		bottom := ics.NumSWB

		for f := uint8(0); f < tns.NFilt[w]; f++ {
			top := bottom
			// Compute bottom, ensuring non-negative
			if tns.Length[w][f] > top {
				bottom = 0
			} else {
				bottom = top - tns.Length[w][f]
			}

			// Clamp order to TNSMaxOrder
			tnsOrder := tns.Order[w][f]
			if tnsOrder > TNSMaxOrder {
				tnsOrder = TNSMaxOrder
			}
			if tnsOrder == 0 {
				continue
			}

			// Decode LPC coefficients
			tnsDecodeCoef(tnsOrder, tns.CoefRes[w], tns.CoefCompress[w][f], tns.Coef[w][f][:], lpc)

			// Calculate filter region bounds
			maxTNS := tables.MaxTNSSFB(cfg.SRIndex, cfg.ObjectType, isShort)

			// Start position
			start := bottom
			if start > maxTNS {
				start = maxTNS
			}
			if start > ics.MaxSFB {
				start = ics.MaxSFB
			}
			startSample := ics.SWBOffset[start]
			if startSample > ics.SWBOffsetMax {
				startSample = ics.SWBOffsetMax
			}

			// End position
			end := top
			if end > maxTNS {
				end = maxTNS
			}
			if end > ics.MaxSFB {
				end = ics.MaxSFB
			}
			endSample := ics.SWBOffset[end]
			if endSample > ics.SWBOffsetMax {
				endSample = ics.SWBOffsetMax
			}

			size := int16(endSample) - int16(startSample)
			if size <= 0 {
				continue
			}

			// Determine filter direction and starting position
			var inc int8
			var filterStart uint16
			if tns.Direction[w][f] != 0 {
				// Backward filtering
				inc = -1
				filterStart = endSample - 1
			} else {
				// Forward filtering
				inc = 1
				filterStart = startSample
			}

			// Apply the filter
			windowOffset := uint16(w) * nshort
			tnsARFilter(spec[windowOffset+filterStart:], size, inc, lpc, tnsOrder)
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/spectrum -run TestTNSDecodeFrame -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/tns.go internal/spectrum/tns_test.go
git commit -m "$(cat <<'EOF'
feat(spectrum): add TNSDecodeFrame for main TNS decoding

Implements the main TNS decoding function that processes all filters
for all windows. Handles both long and short block configurations.

Ported from: tns_decode_frame() in ~/dev/faad2/libfaad/tns.c:84-136

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Add Comprehensive Edge Case Tests

**Files:**
- Modify: `internal/spectrum/tns_test.go`

**Step 1: Write the failing tests**

```go
// Add to internal/spectrum/tns_test.go

func TestTNSDecodeFrame_MultipleFilters(t *testing.T) {
	// Test with multiple TNS filters per window
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        1,
		NumWindowGroups:   1,
		WindowSequence:    syntax.OnlyLongSequence,
		NumSWB:            49,
		MaxSFB:            49,
		SWBOffsetMax:      1024,
		WindowGroupLength: [8]uint8{1},
	}

	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
		if ics.SWBOffset[i] > 1024 {
			ics.SWBOffset[i] = 1024
		}
	}

	// Two filters
	ics.TNS.NFilt[0] = 2
	ics.TNS.CoefRes[0] = 1 // 4-bit coefficients

	// First filter: SFB 30-40
	ics.TNS.Length[0][0] = 10
	ics.TNS.Order[0][0] = 2
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefCompress[0][0] = 0
	ics.TNS.Coef[0][0][0] = 1
	ics.TNS.Coef[0][0][1] = 2

	// Second filter: SFB 40-49
	ics.TNS.Length[0][1] = 9
	ics.TNS.Order[0][1] = 1
	ics.TNS.Direction[0][1] = 1 // Backward
	ics.TNS.CoefCompress[0][1] = 0
	ics.TNS.Coef[0][1][0] = 3

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = float64(i % 10)
	}

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	// Verify no invalid values
	for i := range spec {
		if math.IsNaN(spec[i]) || math.IsInf(spec[i], 0) {
			t.Errorf("spec[%d] is invalid: %v", i, spec[i])
		}
	}
}

func TestTNSDecodeFrame_MaxOrder(t *testing.T) {
	// Test with maximum filter order (20)
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        1,
		NumWindowGroups:   1,
		WindowSequence:    syntax.OnlyLongSequence,
		NumSWB:            49,
		MaxSFB:            49,
		SWBOffsetMax:      1024,
		WindowGroupLength: [8]uint8{1},
	}

	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
		if ics.SWBOffset[i] > 1024 {
			ics.SWBOffset[i] = 1024
		}
	}

	ics.TNS.NFilt[0] = 1
	ics.TNS.CoefRes[0] = 1
	ics.TNS.Length[0][0] = 40
	ics.TNS.Order[0][0] = 20 // Max order
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefCompress[0][0] = 0

	// Fill all 20 coefficients
	for i := 0; i < 20; i++ {
		ics.TNS.Coef[0][0][i] = uint8(i % 16)
	}

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = 1.0
	}

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	// Should not crash and produce valid output
	for i := range spec {
		if math.IsNaN(spec[i]) || math.IsInf(spec[i], 0) {
			t.Errorf("spec[%d] is invalid: %v", i, spec[i])
		}
	}
}

func TestTNSDecodeFrame_OrderExceedsMax(t *testing.T) {
	// Test that order > TNSMaxOrder is clamped
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        1,
		NumWindowGroups:   1,
		WindowSequence:    syntax.OnlyLongSequence,
		NumSWB:            49,
		MaxSFB:            49,
		SWBOffsetMax:      1024,
		WindowGroupLength: [8]uint8{1},
	}

	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
		if ics.SWBOffset[i] > 1024 {
			ics.SWBOffset[i] = 1024
		}
	}

	ics.TNS.NFilt[0] = 1
	ics.TNS.CoefRes[0] = 0
	ics.TNS.Length[0][0] = 30
	ics.TNS.Order[0][0] = 25 // Exceeds TNSMaxOrder (20)
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefCompress[0][0] = 0

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = 1.0
	}

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	// Should not panic
	TNSDecodeFrame(spec, cfg)

	for i := range spec {
		if math.IsNaN(spec[i]) || math.IsInf(spec[i], 0) {
			t.Errorf("spec[%d] is invalid: %v", i, spec[i])
		}
	}
}

func TestTNSDecodeFrame_ZeroRegion(t *testing.T) {
	// Test when filter region computes to zero size
	ics := &syntax.ICStream{
		TNSDataPresent:    true,
		NumWindows:        1,
		NumWindowGroups:   1,
		WindowSequence:    syntax.OnlyLongSequence,
		NumSWB:            49,
		MaxSFB:            5, // Very low max_sfb
		SWBOffsetMax:      100,
		WindowGroupLength: [8]uint8{1},
	}

	for i := 0; i < 52; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
	}

	ics.TNS.NFilt[0] = 1
	ics.TNS.CoefRes[0] = 0
	ics.TNS.Length[0][0] = 10
	ics.TNS.Order[0][0] = 5
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefCompress[0][0] = 0

	spec := make([]float64, 1024)
	for i := range spec {
		spec[i] = 1.0
	}
	original := make([]float64, len(spec))
	copy(original, spec)

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSDecodeFrame(spec, cfg)

	// When region is zero or negative, spectrum should be unchanged
	// (filter is skipped)
}
```

**Step 2: Run tests to verify they pass or identify issues**

Run: `go test ./internal/spectrum -run TestTNSDecodeFrame -v`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/spectrum/tns_test.go
git commit -m "$(cat <<'EOF'
test(spectrum): add comprehensive TNS edge case tests

Tests for multiple filters, max order, order clamping, and zero
region handling.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Run Full Test Suite and Lint

**Step 1: Run make check**

Run: `make check`
Expected: All tests pass, no lint errors

**Step 2: Fix any issues found**

If there are lint errors or test failures, fix them before proceeding.

**Step 3: Final commit (if any fixes were needed)**

```bash
git add -A
git commit -m "$(cat <<'EOF'
fix(spectrum): address lint and test issues in TNS implementation

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Summary

**Files created/modified:**
- `internal/spectrum/tns_tables.go` - TNS coefficient tables
- `internal/spectrum/tns_tables_test.go` - Table tests
- `internal/spectrum/tns.go` - TNS decoding functions
- `internal/spectrum/tns_test.go` - Comprehensive tests

**Functions implemented:**
1. `getTNSCoefTable()` - Coefficient table selector
2. `tnsDecodeCoef()` - LPC coefficient conversion (Levinson-Durbin)
3. `tnsARFilter()` - All-pole IIR filter with double ringbuffer
4. `TNSDecodeFrame()` - Main entry point for TNS decoding

**Constants added:**
- `TNSMaxOrder = 20`

**Dependencies on existing code:**
- `internal/syntax.ICStream` and `syntax.TNSInfo`
- `internal/tables.MaxTNSSFB()`
- `aac.ObjectType`
