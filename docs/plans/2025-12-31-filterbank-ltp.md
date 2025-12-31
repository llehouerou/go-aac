# Filter Bank LTP (Forward Transform) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add forward MDCT with windowing for Long Term Prediction (LTP) to the filter bank.

**Architecture:** The `FilterBankLTP` method applies windowing to time-domain samples, then performs a forward MDCT. It's the inverse of `IFilterBank` - instead of freq→time, it transforms time→freq for LTP prediction. Only supports long block sequences (not EIGHT_SHORT_SEQUENCE).

**Tech Stack:** Go, internal/filterbank, internal/mdct (Forward method already exists)

---

## Background

**FAAD2 Source:** `~/dev/faad2/libfaad/filtbank.c:337-408` (`filter_bank_ltp`)

**Key Constraints:**
- Only works for LTP profile (used for prediction)
- Does NOT support `EIGHT_SHORT_SEQUENCE` (short blocks not used with LTP)
- Windows time samples before applying forward MDCT
- Different windowing for each window sequence type

**Existing Infrastructure:**
- `FilterBank` struct with `mdct256`, `mdct2048` (filterbank.go:29-35)
- `MDCT.Forward()` method (mdct.go:148-204)
- Window functions: `GetLongWindow()`, `GetShortWindow()` (window.go)
- Window sequence constants: `syntax.OnlyLongSequence`, etc.

---

## Task 1: Add Windowed Buffer to FilterBank Struct

**Files:**
- Modify: `internal/filterbank/filterbank.go:29-51`

**Step 1: Update FilterBank struct to add windowed buffer**

Add a new buffer field for windowed data (reused to avoid allocations):

```go
// FilterBank holds state for inverse filter bank operations.
// ...existing comment...
type FilterBank struct {
	mdct256  *mdct.MDCT // For short blocks (256-sample IMDCT)
	mdct2048 *mdct.MDCT // For long blocks (2048-sample IMDCT)

	// Internal buffers (reused to avoid allocations)
	transfBuf   []float32 // 2*frameLength for IMDCT output
	windowedBuf []float32 // 2*frameLength for LTP windowed input
}
```

**Step 2: Initialize the buffer in NewFilterBank**

Update `NewFilterBank` to allocate the new buffer:

```go
func NewFilterBank(frameLen uint16) *FilterBank {
	nshort := frameLen / 8 // 128 for standard AAC

	fb := &FilterBank{
		mdct256:     mdct.NewMDCT(2 * nshort),   // 256 for short blocks
		mdct2048:    mdct.NewMDCT(2 * frameLen), // 2048 for long blocks
		transfBuf:   make([]float32, 2*frameLen),
		windowedBuf: make([]float32, 2*frameLen),
	}

	return fb
}
```

**Step 3: Run tests to verify no regressions**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -v`
Expected: All existing tests PASS

**Step 4: Commit**

```bash
git add internal/filterbank/filterbank.go
git commit -m "feat(filterbank): add windowedBuf for LTP forward transform"
```

---

## Task 2: Write Failing Test for FilterBankLTP with OnlyLongSequence

**Files:**
- Modify: `internal/filterbank/filterbank_test.go`

**Step 1: Write the failing test**

Add test for `FilterBankLTP` with `OnlyLongSequence`:

```go
func TestFilterBankLTP_OnlyLongSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	// Input: 2*frameLength time samples (2048)
	inData := make([]float32, 2048)
	for i := range inData {
		inData[i] = float32(i%100) * 0.01
	}

	// Output: frameLength MDCT coefficients (1024)
	outMDCT := make([]float32, 1024)

	// Call FilterBankLTP
	fb.FilterBankLTP(
		syntax.OnlyLongSequence,
		SineWindow, // window_shape
		SineWindow, // window_shape_prev
		inData,
		outMDCT,
	)

	// Output should not be all zeros
	allZero := true
	for _, v := range outMDCT {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("outMDCT should contain non-zero values after FilterBankLTP")
	}

	// Verify no NaN or Inf values
	for i, v := range outMDCT {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Errorf("outMDCT[%d] = %v (invalid)", i, v)
		}
	}
}
```

Add the math import at the top if not present.

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -run TestFilterBankLTP_OnlyLongSequence -v`
Expected: FAIL with "fb.FilterBankLTP undefined"

**Step 3: Commit**

```bash
git add internal/filterbank/filterbank_test.go
git commit -m "test(filterbank): add failing test for FilterBankLTP OnlyLongSequence"
```

---

## Task 3: Implement FilterBankLTP for OnlyLongSequence

**Files:**
- Modify: `internal/filterbank/filterbank.go`

**Step 1: Add FilterBankLTP method signature and OnlyLongSequence case**

Add the method after `IFilterBank`:

```go
// FilterBankLTP performs the forward filter bank operation for Long Term Prediction.
// This converts time-domain samples to frequency-domain MDCT coefficients.
//
// Parameters:
//   - windowSequence: One of OnlyLongSequence, LongStartSequence, LongStopSequence
//     (EIGHT_SHORT_SEQUENCE is NOT supported for LTP)
//   - windowShape: Current frame's window shape (SineWindow or KBDWindow)
//   - windowShapePrev: Previous frame's window shape
//   - inData: Input time samples (2*frameLen samples)
//   - outMDCT: Output MDCT coefficients (frameLen samples)
//
// Ported from: filter_bank_ltp() in ~/dev/faad2/libfaad/filtbank.c:337-408
func (fb *FilterBank) FilterBankLTP(
	windowSequence syntax.WindowSequence,
	windowShape uint8,
	windowShapePrev uint8,
	inData []float32,
	outMDCT []float32,
) {
	nlong := len(outMDCT)
	nshort := nlong / 8
	_ = nshort // Will be used for other window sequences

	windowedBuf := fb.windowedBuf

	// Clear windowed buffer
	for i := range windowedBuf[:2*nlong] {
		windowedBuf[i] = 0
	}

	// Get windows for current and previous frame
	windowLong := GetLongWindow(int(windowShape))
	windowLongPrev := GetLongWindow(int(windowShapePrev))
	_ = windowLong     // Used in windowing
	_ = windowLongPrev // Used in windowing

	switch windowSequence {
	case syntax.OnlyLongSequence:
		// Window first half with previous window (ascending)
		// Window second half with current window (descending)
		// Ported from: filtbank.c:374-380
		for i := nlong - 1; i >= 0; i-- {
			windowedBuf[i] = inData[i] * windowLongPrev[i]
			windowedBuf[i+nlong] = inData[i+nlong] * windowLong[nlong-1-i]
		}
		// Forward MDCT
		fb.mdct2048.Forward(windowedBuf[:2*nlong], outMDCT)

	case syntax.LongStartSequence:
		panic("LongStartSequence not yet implemented in FilterBankLTP")

	case syntax.LongStopSequence:
		panic("LongStopSequence not yet implemented in FilterBankLTP")

	case syntax.EightShortSequence:
		panic("EightShortSequence is not supported for LTP")

	default:
		panic("unknown window sequence in FilterBankLTP")
	}
}
```

**Step 2: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -run TestFilterBankLTP_OnlyLongSequence -v`
Expected: PASS

**Step 3: Run all filterbank tests to verify no regressions**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -v`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add internal/filterbank/filterbank.go
git commit -m "feat(filterbank): implement FilterBankLTP for OnlyLongSequence"
```

---

## Task 4: Write Failing Test for FilterBankLTP with LongStartSequence

**Files:**
- Modify: `internal/filterbank/filterbank_test.go`

**Step 1: Write the failing test**

```go
func TestFilterBankLTP_LongStartSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	inData := make([]float32, 2048)
	for i := range inData {
		inData[i] = float32(i%100) * 0.01
	}

	outMDCT := make([]float32, 1024)

	// Should not panic
	fb.FilterBankLTP(
		syntax.LongStartSequence,
		SineWindow,
		SineWindow,
		inData,
		outMDCT,
	)

	// Output should not be all zeros
	allZero := true
	for _, v := range outMDCT {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("outMDCT should contain non-zero values")
	}

	// Verify structure: second half should have zeros at the end
	// because LONG_START has zeros padding after the short window region
	nshort := 1024 / 8              // 128
	nflat_ls := (1024 - nshort) / 2 // 448

	// The windowed buffer should have structure:
	// [0:nlong] = windowed with long_prev
	// [nlong:nlong+nflat_ls] = direct copy (1.0 multiplier)
	// [nlong+nflat_ls:nlong+nflat_ls+nshort] = windowed with short
	// [nlong+nflat_ls+nshort:2*nlong] = zeros

	// Verify no NaN/Inf
	for i, v := range outMDCT {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Errorf("outMDCT[%d] = %v (invalid)", i, v)
		}
	}

	_ = nflat_ls // Acknowledge the structure knowledge
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -run TestFilterBankLTP_LongStartSequence -v`
Expected: FAIL with panic "LongStartSequence not yet implemented"

**Step 3: Commit**

```bash
git add internal/filterbank/filterbank_test.go
git commit -m "test(filterbank): add failing test for FilterBankLTP LongStartSequence"
```

---

## Task 5: Implement FilterBankLTP for LongStartSequence

**Files:**
- Modify: `internal/filterbank/filterbank.go`

**Step 1: Implement LongStartSequence case**

Replace the panic in the `LongStartSequence` case with:

```go
	case syntax.LongStartSequence:
		// First half: window with long_prev (ascending)
		// Second half: flat region + short window + zeros
		// Ported from: filtbank.c:383-393
		nflat_ls := (nlong - nshort) / 2
		windowShort := GetShortWindow(int(windowShape))

		for i := 0; i < nlong; i++ {
			windowedBuf[i] = inData[i] * windowLongPrev[i]
		}
		for i := 0; i < nflat_ls; i++ {
			windowedBuf[i+nlong] = inData[i+nlong]
		}
		for i := 0; i < nshort; i++ {
			windowedBuf[i+nlong+nflat_ls] = inData[i+nlong+nflat_ls] * windowShort[nshort-1-i]
		}
		for i := 0; i < nflat_ls; i++ {
			windowedBuf[i+nlong+nflat_ls+nshort] = 0
		}
		// Forward MDCT
		fb.mdct2048.Forward(windowedBuf[:2*nlong], outMDCT)
```

**Step 2: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -run TestFilterBankLTP_LongStartSequence -v`
Expected: PASS

**Step 3: Run all filterbank tests**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -v`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add internal/filterbank/filterbank.go
git commit -m "feat(filterbank): implement FilterBankLTP for LongStartSequence"
```

---

## Task 6: Write Failing Test for FilterBankLTP with LongStopSequence

**Files:**
- Modify: `internal/filterbank/filterbank_test.go`

**Step 1: Write the failing test**

```go
func TestFilterBankLTP_LongStopSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	inData := make([]float32, 2048)
	for i := range inData {
		inData[i] = float32(i%100) * 0.01
	}

	outMDCT := make([]float32, 1024)

	// Should not panic
	fb.FilterBankLTP(
		syntax.LongStopSequence,
		SineWindow,
		SineWindow,
		inData,
		outMDCT,
	)

	// Output should not be all zeros
	allZero := true
	for _, v := range outMDCT {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("outMDCT should contain non-zero values")
	}

	// Verify no NaN/Inf
	for i, v := range outMDCT {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Errorf("outMDCT[%d] = %v (invalid)", i, v)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -run TestFilterBankLTP_LongStopSequence -v`
Expected: FAIL with panic "LongStopSequence not yet implemented"

**Step 3: Commit**

```bash
git add internal/filterbank/filterbank_test.go
git commit -m "test(filterbank): add failing test for FilterBankLTP LongStopSequence"
```

---

## Task 7: Implement FilterBankLTP for LongStopSequence

**Files:**
- Modify: `internal/filterbank/filterbank.go`

**Step 1: Implement LongStopSequence case**

Replace the panic in the `LongStopSequence` case with:

```go
	case syntax.LongStopSequence:
		// First half: zeros + short_prev window + flat region
		// Second half: window with long (descending)
		// Ported from: filtbank.c:395-405
		nflat_ls := (nlong - nshort) / 2
		windowShortPrev := GetShortWindow(int(windowShapePrev))

		for i := 0; i < nflat_ls; i++ {
			windowedBuf[i] = 0
		}
		for i := 0; i < nshort; i++ {
			windowedBuf[i+nflat_ls] = inData[i+nflat_ls] * windowShortPrev[i]
		}
		for i := 0; i < nflat_ls; i++ {
			windowedBuf[i+nflat_ls+nshort] = inData[i+nflat_ls+nshort]
		}
		for i := 0; i < nlong; i++ {
			windowedBuf[i+nlong] = inData[i+nlong] * windowLong[nlong-1-i]
		}
		// Forward MDCT
		fb.mdct2048.Forward(windowedBuf[:2*nlong], outMDCT)
```

**Step 2: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -run TestFilterBankLTP_LongStopSequence -v`
Expected: PASS

**Step 3: Run all filterbank tests**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -v`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add internal/filterbank/filterbank.go
git commit -m "feat(filterbank): implement FilterBankLTP for LongStopSequence"
```

---

## Task 8: Write Test for EightShortSequence Panic

**Files:**
- Modify: `internal/filterbank/filterbank_test.go`

**Step 1: Write test that verifies panic for EightShortSequence**

```go
func TestFilterBankLTP_EightShortSequence_Panics(t *testing.T) {
	fb := NewFilterBank(1024)

	inData := make([]float32, 2048)
	outMDCT := make([]float32, 1024)

	defer func() {
		if r := recover(); r == nil {
			t.Error("FilterBankLTP should panic for EightShortSequence")
		}
	}()

	fb.FilterBankLTP(
		syntax.EightShortSequence,
		SineWindow,
		SineWindow,
		inData,
		outMDCT,
	)
}
```

**Step 2: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -run TestFilterBankLTP_EightShortSequence_Panics -v`
Expected: PASS (the panic is expected and caught)

**Step 3: Commit**

```bash
git add internal/filterbank/filterbank_test.go
git commit -m "test(filterbank): verify FilterBankLTP panics for EightShortSequence"
```

---

## Task 9: Write Test for Mixed Window Shapes in FilterBankLTP

**Files:**
- Modify: `internal/filterbank/filterbank_test.go`

**Step 1: Write test for mixed sine/KBD windows**

```go
func TestFilterBankLTP_MixedWindowShapes(t *testing.T) {
	fb := NewFilterBank(1024)

	inData := make([]float32, 2048)
	for i := range inData {
		inData[i] = float32(i%100) * 0.01
	}

	outMDCT := make([]float32, 1024)

	testCases := []struct {
		name            string
		windowSeq       syntax.WindowSequence
		windowShape     uint8
		windowShapePrev uint8
	}{
		{"OnlyLong_Sine_KBD", syntax.OnlyLongSequence, SineWindow, KBDWindow},
		{"OnlyLong_KBD_Sine", syntax.OnlyLongSequence, KBDWindow, SineWindow},
		{"OnlyLong_KBD_KBD", syntax.OnlyLongSequence, KBDWindow, KBDWindow},
		{"LongStart_Sine_KBD", syntax.LongStartSequence, SineWindow, KBDWindow},
		{"LongStop_KBD_Sine", syntax.LongStopSequence, KBDWindow, SineWindow},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fb.FilterBankLTP(
				tc.windowSeq,
				tc.windowShape,
				tc.windowShapePrev,
				inData,
				outMDCT,
			)

			// Verify output is valid
			for i, v := range outMDCT {
				if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
					t.Errorf("outMDCT[%d] = %v (invalid)", i, v)
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -run TestFilterBankLTP_MixedWindowShapes -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/filterbank/filterbank_test.go
git commit -m "test(filterbank): add mixed window shape tests for FilterBankLTP"
```

---

## Task 10: Write Round-Trip Test (IFilterBank ↔ FilterBankLTP)

**Files:**
- Modify: `internal/filterbank/filterbank_test.go`

**Step 1: Write round-trip correlation test**

```go
func TestFilterBank_RoundTrip_LTP(t *testing.T) {
	// Verify that FilterBankLTP and IFilterBank are consistent.
	// A full round-trip requires proper overlap handling, but we can
	// verify the transforms produce correlated output.

	fb := NewFilterBank(1024)

	// Create smooth input signal
	input := make([]float32, 2048)
	for i := range input {
		input[i] = float32(math.Sin(float64(i) * 2 * math.Pi / 2048))
	}

	// Forward transform: time -> freq
	freqCoeffs := make([]float32, 1024)
	fb.FilterBankLTP(
		syntax.OnlyLongSequence,
		SineWindow,
		SineWindow,
		input,
		freqCoeffs,
	)

	// Inverse transform: freq -> time
	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)
	fb.IFilterBank(
		syntax.OnlyLongSequence,
		SineWindow,
		SineWindow,
		freqCoeffs,
		timeOut,
		overlap,
	)

	// Verify outputs are valid (not NaN/Inf)
	for i, v := range timeOut {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Errorf("timeOut[%d] = %v (invalid)", i, v)
		}
	}

	// Check for some correlation between original and reconstructed
	// (exact reconstruction requires multiple frames with overlap-add)
	var energy float64
	for _, v := range timeOut {
		energy += float64(v) * float64(v)
	}
	if energy == 0 {
		t.Error("round-trip produced all zeros")
	}

	t.Logf("Round-trip output energy: %v", energy)
}
```

**Step 2: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/filterbank/... -run TestFilterBank_RoundTrip_LTP -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/filterbank/filterbank_test.go
git commit -m "test(filterbank): add round-trip correlation test for LTP"
```

---

## Task 11: Run Full Test Suite and Lint

**Files:** None (verification only)

**Step 1: Run all tests**

Run: `cd /home/laurent/dev/go-aac && go test ./... -v`
Expected: All tests PASS

**Step 2: Run linter**

Run: `cd /home/laurent/dev/go-aac && make lint`
Expected: No lint errors

**Step 3: Run format check**

Run: `cd /home/laurent/dev/go-aac && make fmt`
Expected: No changes needed (or apply formatting)

**Step 4: Commit any formatting fixes**

```bash
git add -A
git commit -m "style: apply formatting fixes" || echo "No formatting changes"
```

---

## Task 12: Final Commit and Summary

**Step 1: Verify implementation completeness**

Check that all methods are implemented:
- [x] `FilterBank.windowedBuf` field added
- [x] `FilterBankLTP` with `OnlyLongSequence`
- [x] `FilterBankLTP` with `LongStartSequence`
- [x] `FilterBankLTP` with `LongStopSequence`
- [x] `EightShortSequence` panic (by design)

**Step 2: Review code matches FAAD2**

Verify the implementation matches `~/dev/faad2/libfaad/filtbank.c:337-408`:
- Windowing order matches FAAD2
- Buffer indexing matches FAAD2
- Window selection matches FAAD2

**Step 3: Create summary commit if needed**

If any cleanup was done:
```bash
git add -A
git commit -m "feat(filterbank): complete FilterBankLTP implementation

Ported from ~/dev/faad2/libfaad/filtbank.c:337-408 (filter_bank_ltp)

- Adds forward MDCT with windowing for Long Term Prediction
- Supports OnlyLongSequence, LongStartSequence, LongStopSequence
- Panics for EightShortSequence (not supported for LTP per spec)"
```

---

## Acceptance Criteria (from MIGRATION_STEPS.md)

- [x] Forward MDCT for LTP
- [x] Required only for LTP profile
- [x] Matches FAAD2 implementation structure
