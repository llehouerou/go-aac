# LTP (Long Term Prediction) Decoder Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement Long Term Prediction (LTP) decoding for AAC LTP profile support.

**Architecture:** LTP uses previously decoded samples to predict the current frame's spectral coefficients. The decoder looks back into a state buffer, applies a coefficient, transforms to frequency domain via forward MDCT, applies TNS encoding, and adds the prediction to the current spectrum. LTP requires forward MDCT (filterbank) and TNS MA filter (not yet implemented).

**Tech Stack:** Pure Go, no external dependencies. Floating-point arithmetic (float64).

**Dependencies Status:**
- ✅ LTPInfo struct already exists in `internal/syntax/ltp.go`
- ✅ TNS AR filter exists in `internal/spectrum/tns.go`
- ❌ TNS MA filter NOT implemented (needed for LTP)
- ❌ Forward MDCT NOT implemented (Phase 5 - filterbank)
- ❌ Filterbank NOT implemented (Phase 5)

**Implementation Strategy:** Implement all LTP components that don't require filterbank. Create an interface for the forward MDCT dependency that will be satisfied when Phase 5 (filterbank) is implemented.

---

## Task 1: Add TNS MA Filter Function

The TNS MA (Moving Average) filter is needed for `tns_encode_frame`, which LTP uses to apply TNS to the predicted spectrum. This is the all-zeros filter counterpart to the AR filter.

**Files:**
- Modify: `internal/spectrum/tns.go`
- Modify: `internal/spectrum/tns_test.go`

**Step 1: Write the failing test for tnsMAFilter**

Add to `internal/spectrum/tns_test.go`:

```go
func TestTNSMAFilter_Basic(t *testing.T) {
	// Simple test: apply MA filter with known coefficients
	// MA filter: y[n] = x[n] + lpc[1]*x[n-1] + lpc[2]*x[n-2] + ...
	spec := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	// Simple 2nd order filter: y[n] = x[n] + 0.5*x[n-1] + 0.25*x[n-2]
	lpc := []float64{1.0, 0.5, 0.25}

	tnsMAFilter(spec, 5, 1, lpc, 2)

	// Expected results:
	// y[0] = x[0] + 0.5*0 + 0.25*0 = 1.0
	// y[1] = x[1] + 0.5*x[0] + 0.25*0 = 2.0 + 0.5*1.0 = 2.5
	// y[2] = x[2] + 0.5*x[1] + 0.25*x[0] = 3.0 + 0.5*2.0 + 0.25*1.0 = 4.25
	// y[3] = x[3] + 0.5*x[2] + 0.25*x[1] = 4.0 + 0.5*3.0 + 0.25*2.0 = 6.0
	// y[4] = x[4] + 0.5*x[3] + 0.25*x[2] = 5.0 + 0.5*4.0 + 0.25*3.0 = 7.75
	expected := []float64{1.0, 2.5, 4.25, 6.0, 7.75}

	for i, exp := range expected {
		if math.Abs(spec[i]-exp) > 1e-10 {
			t.Errorf("sample %d: got %v, want %v", i, spec[i], exp)
		}
	}
}

func TestTNSMAFilter_BackwardDirection(t *testing.T) {
	// Test backward direction (inc = -1)
	spec := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	original := make([]float64, len(spec))
	copy(original, spec)

	lpc := []float64{1.0, 0.5, 0.25}

	// Start from index 4, go backward
	tnsMAFilterWithOffset(spec, 4, 5, -1, lpc, 2)

	// Processing order: spec[4], spec[3], spec[2], spec[1], spec[0]
	// y[4] = x[4] + 0.5*0 + 0.25*0 = 5.0
	// y[3] = x[3] + 0.5*x[4] + 0.25*0 = 4.0 + 0.5*5.0 = 6.5
	// y[2] = x[2] + 0.5*x[3] + 0.25*x[4] = 3.0 + 0.5*4.0 + 0.25*5.0 = 6.25
	// y[1] = x[1] + 0.5*x[2] + 0.25*x[3] = 2.0 + 0.5*3.0 + 0.25*4.0 = 4.5
	// y[0] = x[0] + 0.5*x[1] + 0.25*x[2] = 1.0 + 0.5*2.0 + 0.25*3.0 = 2.75
	expected := []float64{2.75, 4.5, 6.25, 6.5, 5.0}

	for i, exp := range expected {
		if math.Abs(spec[i]-exp) > 1e-10 {
			t.Errorf("sample %d: got %v, want %v", i, spec[i], exp)
		}
	}
}

func TestTNSMAFilter_ZeroOrder(t *testing.T) {
	spec := []float64{1.0, 2.0, 3.0}
	original := make([]float64, len(spec))
	copy(original, spec)

	lpc := []float64{1.0}
	tnsMAFilter(spec, 3, 1, lpc, 0)

	// Zero order filter should not modify spectrum
	for i := range spec {
		if spec[i] != original[i] {
			t.Errorf("sample %d modified with zero order: got %v, want %v", i, spec[i], original[i])
		}
	}
}

func TestTNSMAFilter_ZeroSize(t *testing.T) {
	spec := []float64{1.0, 2.0, 3.0}
	original := make([]float64, len(spec))
	copy(original, spec)

	lpc := []float64{1.0, 0.5}
	tnsMAFilter(spec, 0, 1, lpc, 1)

	// Zero size should not modify spectrum
	for i := range spec {
		if spec[i] != original[i] {
			t.Errorf("sample %d modified with zero size: got %v, want %v", i, spec[i], original[i])
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test -v ./internal/spectrum/... -run TestTNSMAFilter`
Expected: FAIL with "undefined: tnsMAFilter"

**Step 3: Write minimal implementation**

Add to `internal/spectrum/tns.go` after the AR filter functions:

```go
// tnsMAFilter applies an all-zero (MA) FIR filter to spectral coefficients.
// This is the TNS encoding filter used by LTP (opposite of AR decoding filter).
//
// The filter is defined by:
//
//	y[n] = x[n] + lpc[1]*x[n-1] + lpc[2]*x[n-2] + ... + lpc[order]*x[n-order]
//
// Note: This uses the INPUT values (x) in the state, not output values (y).
// This is the key difference from the AR filter.
//
// Parameters:
//   - spectrum: spectral data to filter (modified in-place)
//   - size: number of samples to filter
//   - inc: direction (+1 for forward, -1 for backward)
//   - lpc: LPC filter coefficients (lpc[0] is always 1.0)
//   - order: filter order
//
// Ported from: tns_ma_filter() in ~/dev/faad2/libfaad/tns.c:295-339
func tnsMAFilter(spectrum []float64, size int16, inc int8, lpc []float64, order uint8) {
	tnsMAFilterWithOffset(spectrum, 0, size, inc, lpc, order)
}

// tnsMAFilterWithOffset applies an all-zero (MA) FIR filter starting at a specific offset.
//
// Ported from: tns_ma_filter() in ~/dev/faad2/libfaad/tns.c:295-339
func tnsMAFilterWithOffset(spectrum []float64, startOffset int, size int16, inc int8, lpc []float64, order uint8) {
	if size <= 0 || order == 0 {
		return
	}

	// State is stored as a double ringbuffer for efficient wraparound
	// State stores INPUT values (x), not output values
	state := make([]float64, 2*TNSMaxOrder)
	stateIndex := int8(0)

	// Process each sample
	idx := startOffset
	for i := int16(0); i < size; i++ {
		// Store current input in state BEFORE computing output
		x := spectrum[idx]

		// Compute filter output: y = x + sum(lpc[j+1] * state[j])
		y := x
		for j := uint8(0); j < order; j++ {
			y += state[int(stateIndex)+int(j)] * lpc[j+1]
		}

		// Update double ringbuffer state with INPUT value
		stateIndex--
		if stateIndex < 0 {
			stateIndex = int8(order - 1)
		}
		state[stateIndex] = x
		state[int(stateIndex)+int(order)] = x

		// Write output
		spectrum[idx] = y
		idx += int(inc)
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test -v ./internal/spectrum/... -run TestTNSMAFilter`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/tns.go internal/spectrum/tns_test.go
git commit -m "$(cat <<'EOF'
feat(spectrum): add tnsMAFilter for TNS encoding used by LTP

The MA (Moving Average) filter is the encoding counterpart to the AR
(Auto-Regressive) decoding filter. While AR filter uses output values
in feedback (y[n-k]), MA filter uses input values (x[n-k]).

This is needed by lt_prediction() which applies TNS encoding to the
predicted spectrum before adding it to the decoded spectrum.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Add TNS Encode Frame Function

The `TNSEncodeFrame` function applies TNS to a spectrum using the MA filter. This is used by LTP to match the TNS processing that was applied to the original spectrum.

**Files:**
- Modify: `internal/spectrum/tns.go`
- Modify: `internal/spectrum/tns_test.go`

**Step 1: Write the failing test**

Add to `internal/spectrum/tns_test.go`:

```go
func TestTNSEncodeFrame_NoTNSData(t *testing.T) {
	spec := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	original := make([]float64, len(spec))
	copy(original, spec)

	ics := &syntax.ICStream{
		TNSDataPresent: false,
	}

	cfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     4, // 44100 Hz
		ObjectType:  aac.ObjectTypeLC,
		FrameLength: 1024,
	}

	TNSEncodeFrame(spec, cfg)

	// No TNS data - spectrum should be unchanged
	for i := range spec {
		if spec[i] != original[i] {
			t.Errorf("sample %d modified without TNS data: got %v, want %v", i, spec[i], original[i])
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test -v ./internal/spectrum/... -run TestTNSEncodeFrame`
Expected: FAIL with "undefined: TNSEncodeFrame"

**Step 3: Write minimal implementation**

Add to `internal/spectrum/tns.go`:

```go
// TNSEncodeFrame applies TNS encoding to one channel.
// This applies the MA (all-zero) filter to the spectrum, which is the
// inverse operation of TNS decoding. Used by LTP to match TNS processing.
//
// Ported from: tns_encode_frame() in ~/dev/faad2/libfaad/tns.c:139-191
func TNSEncodeFrame(spec []float64, cfg *TNSDecodeConfig) {
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

			// Apply the MA filter (encoding, not decoding)
			windowOffset := uint16(w) * nshort
			tnsMAFilterWithOffset(spec, int(windowOffset+filterStart), size, inc, lpc, tnsOrder)
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test -v ./internal/spectrum/... -run TestTNSEncodeFrame`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/tns.go internal/spectrum/tns_test.go
git commit -m "$(cat <<'EOF'
feat(spectrum): add TNSEncodeFrame for LTP TNS processing

TNSEncodeFrame applies MA filtering to spectrum, the inverse of
TNSDecodeFrame which applies AR filtering. This is used by LTP
to apply the same TNS processing to the predicted spectrum.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Add LTP Codebook and Helper Functions

Create the LTP module with the codebook table and helper functions that don't require filterbank.

**Files:**
- Create: `internal/spectrum/ltp.go`
- Create: `internal/spectrum/ltp_test.go`

**Step 1: Write the failing test**

Create `internal/spectrum/ltp_test.go`:

```go
package spectrum

import (
	"math"
	"testing"

	"github.com/llehouerou/go-aac"
)

func TestIsLTPObjectType(t *testing.T) {
	tests := []struct {
		name       string
		objectType aac.ObjectType
		want       bool
	}{
		{"LC is not LTP", aac.ObjectTypeLC, false},
		{"Main is not LTP", aac.ObjectTypeMain, false},
		{"LTP is LTP", aac.ObjectTypeLTP, true},
		{"ER_LTP is LTP", aac.ObjectTypeERLTP, true},
		{"LD is LTP", aac.ObjectTypeLD, true},
		{"SSR is not LTP", aac.ObjectTypeSSR, false},
		{"HE_AAC is not LTP", aac.ObjectTypeHE, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLTPObjectType(tt.objectType)
			if got != tt.want {
				t.Errorf("IsLTPObjectType(%v) = %v, want %v", tt.objectType, got, tt.want)
			}
		})
	}
}

func TestLTPCodebook(t *testing.T) {
	// Verify codebook has correct values from FAAD2
	expected := []float64{
		0.570829,
		0.696616,
		0.813004,
		0.911304,
		0.984900,
		1.067894,
		1.194601,
		1.369533,
	}

	if len(ltpCodebook) != 8 {
		t.Fatalf("ltpCodebook length = %d, want 8", len(ltpCodebook))
	}

	for i, exp := range expected {
		if math.Abs(ltpCodebook[i]-exp) > 1e-6 {
			t.Errorf("ltpCodebook[%d] = %v, want %v", i, ltpCodebook[i], exp)
		}
	}
}

func TestRealToInt16(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  int16
	}{
		{"zero", 0.0, 0},
		{"positive small", 100.5, 101},   // rounds to nearest
		{"negative small", -100.5, -101}, // rounds to nearest (away from zero for .5)
		{"positive large", 32767.0, 32767},
		{"negative large", -32768.0, -32768},
		{"positive overflow", 40000.0, 32767},   // clamp
		{"negative overflow", -40000.0, -32768}, // clamp
		{"positive round down", 100.3, 100},
		{"negative round down", -100.3, -100},
		{"positive round up", 100.7, 101},
		{"negative round up", -100.7, -101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := realToInt16(tt.input)
			if got != tt.want {
				t.Errorf("realToInt16(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test -v ./internal/spectrum/... -run "TestIsLTPObjectType|TestLTPCodebook|TestRealToInt16"`
Expected: FAIL with "undefined: IsLTPObjectType"

**Step 3: Write minimal implementation**

Create `internal/spectrum/ltp.go`:

```go
// internal/spectrum/ltp.go
package spectrum

import (
	"math"

	"github.com/llehouerou/go-aac"
)

// ltpCodebook contains the 8 LTP coefficient values.
// The transmitted coef index (0-7) indexes into this table.
//
// Ported from: codebook[] in ~/dev/faad2/libfaad/lt_predict.c:68-78
var ltpCodebook = [8]float64{
	0.570829,
	0.696616,
	0.813004,
	0.911304,
	0.984900,
	1.067894,
	1.194601,
	1.369533,
}

// IsLTPObjectType returns true if the given object type supports LTP.
//
// Ported from: is_ltp_ot() in ~/dev/faad2/libfaad/lt_predict.c:49-66
func IsLTPObjectType(objectType aac.ObjectType) bool {
	switch objectType {
	case aac.ObjectTypeLTP, aac.ObjectTypeERLTP, aac.ObjectTypeLD:
		return true
	default:
		return false
	}
}

// realToInt16 converts a floating-point sample to int16 with rounding and clamping.
// Uses round-half-away-from-zero for .5 values.
//
// Ported from: real_to_int16() in ~/dev/faad2/libfaad/lt_predict.c:152-170
func realToInt16(sigIn float64) int16 {
	// Round to nearest integer (away from zero for .5)
	var rounded float64
	if sigIn >= 0 {
		rounded = math.Floor(sigIn + 0.5)
		if rounded >= 32768.0 {
			return 32767
		}
	} else {
		rounded = math.Ceil(sigIn - 0.5)
		if rounded <= -32768.0 {
			return -32768
		}
	}
	return int16(rounded)
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test -v ./internal/spectrum/... -run "TestIsLTPObjectType|TestLTPCodebook|TestRealToInt16"`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/ltp.go internal/spectrum/ltp_test.go
git commit -m "$(cat <<'EOF'
feat(spectrum): add LTP codebook and helper functions

Adds the foundation for Long Term Prediction support:
- ltpCodebook: 8-value coefficient lookup table
- IsLTPObjectType: checks if object type supports LTP
- realToInt16: converts float samples to int16 for LTP state buffer

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Add LTP State Update Function

The `LTPUpdateState` function updates the LTP state buffer with the latest decoded samples. This function is independent of filterbank.

**Files:**
- Modify: `internal/spectrum/ltp.go`
- Modify: `internal/spectrum/ltp_test.go`

**Step 1: Write the failing test**

Add to `internal/spectrum/ltp_test.go`:

```go
func TestLTPUpdateState_NonLD(t *testing.T) {
	// Test state update for non-LD object types (LC, LTP, etc.)
	frameLen := uint16(8) // Small for testing

	// State buffer: 4*frameLen = 32 samples
	// Layout: [old_half | time_samples | overlap_samples | zeros]
	state := make([]int16, 4*frameLen)

	// Initialize with some values
	for i := range state {
		state[i] = int16(i + 100)
	}

	// Time domain samples (current frame output)
	time := make([]float64, frameLen)
	for i := range time {
		time[i] = float64(i * 10)
	}

	// Overlap samples from filter bank
	overlap := make([]float64, frameLen)
	for i := range overlap {
		overlap[i] = float64(i * 20)
	}

	LTPUpdateState(state, time, overlap, frameLen, aac.ObjectTypeLTP)

	// Expected layout after update:
	// [0..7] = old state[8..15]
	// [8..15] = realToInt16(time[0..7])
	// [16..23] = realToInt16(overlap[0..7])
	// [24..31] = unchanged (zeros initialized at start)

	// Check shifted values
	for i := uint16(0); i < frameLen; i++ {
		expected := int16(i + 100 + 8) // Original state[i+8]
		if state[i] != expected {
			t.Errorf("state[%d] = %d, want %d (shifted)", i, state[i], expected)
		}
	}

	// Check time values
	for i := uint16(0); i < frameLen; i++ {
		expected := realToInt16(float64(i * 10))
		if state[frameLen+i] != expected {
			t.Errorf("state[%d] = %d, want %d (time)", frameLen+i, state[frameLen+i], expected)
		}
	}

	// Check overlap values
	for i := uint16(0); i < frameLen; i++ {
		expected := realToInt16(float64(i * 20))
		if state[2*frameLen+i] != expected {
			t.Errorf("state[%d] = %d, want %d (overlap)", 2*frameLen+i, state[2*frameLen+i], expected)
		}
	}
}

func TestLTPUpdateState_LD(t *testing.T) {
	// Test state update for LD object type (extra 512 lookback)
	frameLen := uint16(8) // Small for testing

	// State buffer: 4*frameLen = 32 samples
	state := make([]int16, 4*frameLen)

	// Initialize with some values
	for i := range state {
		state[i] = int16(i + 100)
	}

	// Time domain samples
	time := make([]float64, frameLen)
	for i := range time {
		time[i] = float64(i * 10)
	}

	// Overlap samples
	overlap := make([]float64, frameLen)
	for i := range overlap {
		overlap[i] = float64(i * 20)
	}

	LTPUpdateState(state, time, overlap, frameLen, aac.ObjectTypeLD)

	// Expected layout after update (LD mode):
	// [0..7] = old state[8..15]
	// [8..15] = old state[16..23]
	// [16..23] = realToInt16(time[0..7])
	// [24..31] = realToInt16(overlap[0..7])

	// Check first shift
	for i := uint16(0); i < frameLen; i++ {
		expected := int16(i + 100 + 8) // Original state[i+8]
		if state[i] != expected {
			t.Errorf("state[%d] = %d, want %d (first shift)", i, state[i], expected)
		}
	}

	// Check second shift
	for i := uint16(0); i < frameLen; i++ {
		expected := int16(i + 100 + 16) // Original state[i+16]
		if state[frameLen+i] != expected {
			t.Errorf("state[%d] = %d, want %d (second shift)", frameLen+i, state[frameLen+i], expected)
		}
	}

	// Check time values
	for i := uint16(0); i < frameLen; i++ {
		expected := realToInt16(float64(i * 10))
		if state[2*frameLen+i] != expected {
			t.Errorf("state[%d] = %d, want %d (time)", 2*frameLen+i, state[2*frameLen+i], expected)
		}
	}

	// Check overlap values
	for i := uint16(0); i < frameLen; i++ {
		expected := realToInt16(float64(i * 20))
		if state[3*frameLen+i] != expected {
			t.Errorf("state[%d] = %d, want %d (overlap)", 3*frameLen+i, state[3*frameLen+i], expected)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test -v ./internal/spectrum/... -run TestLTPUpdateState`
Expected: FAIL with "undefined: LTPUpdateState"

**Step 3: Write minimal implementation**

Add to `internal/spectrum/ltp.go`:

```go
// LTPUpdateState updates the LTP state buffer with the latest decoded samples.
// This must be called after each frame is decoded to maintain the prediction state.
//
// The state buffer layout is:
//   - Non-LD: [old_half | time_samples | overlap_samples | zeros]
//   - LD: [extra_512 | old_half | time_samples | overlap_samples]
//
// Parameters:
//   - ltPredStat: LTP state buffer (4*frameLen samples for LTP, or 4*512 for LD)
//   - time: decoded time-domain samples for current frame
//   - overlap: overlap samples from filter bank
//   - frameLen: frame length (1024 or 960)
//   - objectType: AAC object type
//
// Ported from: lt_update_state() in ~/dev/faad2/libfaad/lt_predict.c:173-213
func LTPUpdateState(ltPredStat []int16, time, overlap []float64, frameLen uint16, objectType aac.ObjectType) {
	if objectType == aac.ObjectTypeLD {
		// LD mode: extra 512 samples lookback
		for i := uint16(0); i < frameLen; i++ {
			ltPredStat[i] = ltPredStat[i+frameLen]                 // Shift down
			ltPredStat[frameLen+i] = ltPredStat[i+2*frameLen]      // Shift down
			ltPredStat[2*frameLen+i] = realToInt16(time[i])        // New time samples
			ltPredStat[3*frameLen+i] = realToInt16(overlap[i])     // New overlap samples
		}
	} else {
		// Non-LD mode (LTP, etc.)
		for i := uint16(0); i < frameLen; i++ {
			ltPredStat[i] = ltPredStat[i+frameLen]                 // Shift down
			ltPredStat[frameLen+i] = realToInt16(time[i])          // New time samples
			ltPredStat[2*frameLen+i] = realToInt16(overlap[i])     // New overlap samples
			// ltPredStat[3*frameLen+i] stays zero (initialized once)
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test -v ./internal/spectrum/... -run TestLTPUpdateState`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/ltp.go internal/spectrum/ltp_test.go
git commit -m "$(cat <<'EOF'
feat(spectrum): add LTPUpdateState for LTP state buffer management

Updates the LTP prediction state buffer after each decoded frame.
Handles both regular LTP mode and LD mode (which has extra lookback).

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Add LTP Prediction Interface and Stub

Create the `LTPPrediction` function with an interface for the forward MDCT dependency. This allows the function structure to be tested even though filterbank isn't implemented yet.

**Files:**
- Modify: `internal/spectrum/ltp.go`
- Modify: `internal/spectrum/ltp_test.go`

**Step 1: Write the failing test**

Add to `internal/spectrum/ltp_test.go`:

```go
func TestLTPPrediction_NoDataPresent(t *testing.T) {
	frameLen := uint16(1024)
	spec := make([]float64, frameLen)
	for i := range spec {
		spec[i] = float64(i)
	}
	original := make([]float64, len(spec))
	copy(original, spec)

	ics := &syntax.ICStream{
		WindowSequence: syntax.OnlyLongSequence,
	}

	ltp := &syntax.LTPInfo{
		DataPresent: false,
	}

	cfg := &LTPConfig{
		ICS:         ics,
		LTP:         ltp,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLTP,
		FrameLength: frameLen,
		// FilterBank is nil - won't be called since DataPresent is false
	}

	LTPPrediction(spec, nil, cfg)

	// No LTP data - spectrum should be unchanged
	for i := range spec {
		if spec[i] != original[i] {
			t.Errorf("sample %d modified without LTP data: got %v, want %v", i, spec[i], original[i])
		}
	}
}

func TestLTPPrediction_ShortBlocks(t *testing.T) {
	frameLen := uint16(1024)
	spec := make([]float64, frameLen)
	for i := range spec {
		spec[i] = float64(i)
	}
	original := make([]float64, len(spec))
	copy(original, spec)

	ics := &syntax.ICStream{
		WindowSequence: syntax.EightShortSequence, // LTP not applied to short blocks
	}

	ltp := &syntax.LTPInfo{
		DataPresent: true,
		Lag:         100,
		Coef:        3,
	}

	cfg := &LTPConfig{
		ICS:         ics,
		LTP:         ltp,
		SRIndex:     4,
		ObjectType:  aac.ObjectTypeLTP,
		FrameLength: frameLen,
	}

	LTPPrediction(spec, nil, cfg)

	// Short blocks - spectrum should be unchanged
	for i := range spec {
		if spec[i] != original[i] {
			t.Errorf("sample %d modified with short blocks: got %v, want %v", i, spec[i], original[i])
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test -v ./internal/spectrum/... -run TestLTPPrediction`
Expected: FAIL with "undefined: LTPConfig"

**Step 3: Write minimal implementation**

Add to `internal/spectrum/ltp.go`:

```go
import (
	// ... existing imports ...
	"github.com/llehouerou/go-aac/internal/syntax"
)

// ForwardMDCT is the interface for forward MDCT transformation.
// This is provided by the filterbank package (Phase 5).
type ForwardMDCT interface {
	// FilterBankLTP applies forward MDCT for LTP.
	// Transforms time-domain samples to frequency-domain coefficients.
	FilterBankLTP(windowSequence uint8, windowShape, windowShapePrev uint8,
		inData []float64, outMDCT []float64, objectType aac.ObjectType, frameLen uint16)
}

// LTPConfig holds configuration for LTP prediction.
type LTPConfig struct {
	// ICS is the individual channel stream
	ICS *syntax.ICStream

	// LTP is the LTP info from parsing
	LTP *syntax.LTPInfo

	// SRIndex is the sample rate index
	SRIndex uint8

	// ObjectType is the AAC object type
	ObjectType aac.ObjectType

	// FrameLength is the frame length (1024 or 960)
	FrameLength uint16

	// WindowShape is the current window shape
	WindowShape uint8

	// WindowShapePrev is the previous window shape
	WindowShapePrev uint8
}

// LTPPrediction applies Long Term Prediction to spectral coefficients.
// LTP uses previously decoded samples to predict and enhance the current frame.
//
// The prediction process:
// 1. Looks back into the state buffer by 'lag' samples
// 2. Multiplies by the LTP coefficient
// 3. Applies forward MDCT to get frequency-domain prediction
// 4. Applies TNS encoding to match original processing
// 5. Adds prediction to spectrum for bands where LTP is active
//
// Parameters:
//   - spec: spectral coefficients to modify (input/output)
//   - ltPredStat: LTP state buffer (previous decoded samples as int16)
//   - fb: forward MDCT interface (from filterbank package)
//   - cfg: LTP configuration
//
// Note: LTP is only applied to long windows, not short blocks.
//
// Ported from: lt_prediction() in ~/dev/faad2/libfaad/lt_predict.c:80-133
func LTPPrediction(spec []float64, ltPredStat []int16, cfg *LTPConfig) {
	LTPPredictionWithMDCT(spec, ltPredStat, nil, cfg)
}

// LTPPredictionWithMDCT applies Long Term Prediction with an explicit forward MDCT.
// Use this when the filterbank is available.
func LTPPredictionWithMDCT(spec []float64, ltPredStat []int16, fb ForwardMDCT, cfg *LTPConfig) {
	ics := cfg.ICS
	ltp := cfg.LTP

	// LTP is not applied to short blocks
	if ics.WindowSequence == syntax.EightShortSequence {
		return
	}

	// Check if LTP data is present
	if !ltp.DataPresent {
		return
	}

	// Forward MDCT is required for actual prediction
	if fb == nil {
		// TODO: Remove this check once filterbank is implemented
		// For now, return early if no filterbank is provided
		return
	}

	numSamples := cfg.FrameLength * 2

	// Create time-domain estimate from state buffer
	xEst := make([]float64, numSamples)
	coef := ltpCodebook[ltp.Coef]

	for i := uint16(0); i < numSamples; i++ {
		// Look back by 'lag' samples and multiply by coefficient
		stateIdx := numSamples + i - ltp.Lag
		xEst[i] = float64(ltPredStat[stateIdx]) * coef
	}

	// Apply forward MDCT to get frequency-domain prediction
	XEst := make([]float64, numSamples)
	fb.FilterBankLTP(ics.WindowSequence, cfg.WindowShape, cfg.WindowShapePrev,
		xEst, XEst, cfg.ObjectType, cfg.FrameLength)

	// Apply TNS encoding to match the processing applied to the original spectrum
	tnsCfg := &TNSDecodeConfig{
		ICS:         ics,
		SRIndex:     cfg.SRIndex,
		ObjectType:  cfg.ObjectType,
		FrameLength: cfg.FrameLength,
	}
	TNSEncodeFrame(XEst, tnsCfg)

	// Add prediction to spectrum for SFBs where LTP is used
	for sfb := uint8(0); sfb < ltp.LastBand; sfb++ {
		if ltp.LongUsed[sfb] {
			low := ics.SWBOffset[sfb]
			high := ics.SWBOffset[sfb+1]
			if high > ics.SWBOffsetMax {
				high = ics.SWBOffsetMax
			}

			for bin := low; bin < high; bin++ {
				spec[bin] += XEst[bin]
			}
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test -v ./internal/spectrum/... -run TestLTPPrediction`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/ltp.go internal/spectrum/ltp_test.go
git commit -m "$(cat <<'EOF'
feat(spectrum): add LTPPrediction with ForwardMDCT interface

Adds the main LTP prediction function that:
- Checks for valid window sequence (not short blocks)
- Retrieves samples from state buffer with lag offset
- Applies LTP coefficient multiplication
- Applies forward MDCT (via interface, requires filterbank)
- Applies TNS encoding to match original processing
- Adds prediction to spectrum for active SFBs

The ForwardMDCT interface abstracts the filterbank dependency,
allowing this code to be tested before Phase 5 is implemented.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Run Full Test Suite and Lint

Verify all tests pass and code is properly formatted.

**Files:**
- None (verification only)

**Step 1: Run make check**

Run: `cd /home/laurent/dev/go-aac && make check`
Expected: All tests pass, no lint errors, code formatted

**Step 2: Fix any issues**

If there are any issues, fix them and re-run.

**Step 3: Final commit if needed**

If any fixes were made, commit them.

---

## Summary

This plan implements LTP support with the following components:

1. **TNS MA Filter** (`tnsMAFilter`): The all-zero filter used for TNS encoding
2. **TNS Encode Frame** (`TNSEncodeFrame`): Applies TNS to spectrum using MA filter
3. **LTP Codebook**: The 8-value coefficient lookup table
4. **Helper Functions**: `IsLTPObjectType`, `realToInt16`
5. **LTP Update State** (`LTPUpdateState`): Updates state buffer after each frame
6. **LTP Prediction** (`LTPPrediction`): Main prediction function with ForwardMDCT interface

**Dependency Notes:**
- The `ForwardMDCT` interface abstracts the filterbank dependency
- Full LTP functionality requires Phase 5 (filterbank) implementation
- Once filterbank is implemented, pass it to `LTPPredictionWithMDCT`
- The `LTPPrediction` function currently returns early if no filterbank is provided

**Files Created/Modified:**
- `internal/spectrum/tns.go` - Added MA filter and TNS encode
- `internal/spectrum/tns_test.go` - Added tests for MA filter and TNS encode
- `internal/spectrum/ltp.go` - New file with all LTP functions
- `internal/spectrum/ltp_test.go` - New file with LTP tests
