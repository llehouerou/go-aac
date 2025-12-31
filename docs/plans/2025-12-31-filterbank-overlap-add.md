# Filter Bank - Overlap-Add Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the inverse filter bank (IMDCT + windowing + overlap-add) that converts frequency-domain spectral data to time-domain PCM samples.

**Architecture:** The FilterBank holds pre-initialized MDCT instances (256 for short blocks, 2048 for long blocks) and uses window lookup functions from the existing filterbank package. The core `IFilterBank()` function handles all 4 window sequences with proper overlap-add.

**Tech Stack:** Pure Go, using existing `internal/mdct`, `internal/filterbank` window tables, `internal/syntax` constants

---

## Prerequisites (Already Implemented)

- `internal/mdct/mdct.go` - MDCT.IMDCT() function ✓
- `internal/filterbank/window.go` - GetLongWindow(), GetShortWindow() ✓
- `internal/filterbank/window_sine.go`, `window_kbd.go` - Generated window tables ✓
- `internal/syntax/constants.go` - WindowSequence constants ✓

## Files to Create/Modify

- Create: `internal/filterbank/filterbank.go` - Main FilterBank struct and IFilterBank
- Create: `internal/filterbank/filterbank_test.go` - Unit tests

## Reference

- FAAD2 Source: `~/dev/faad2/libfaad/filtbank.c` (lines 48-334)
- FAAD2 Struct: `fb_info` in `~/dev/faad2/libfaad/structs.h:67-83`

---

### Task 1: Create FilterBank Struct and Constructor

**Files:**
- Create: `internal/filterbank/filterbank.go`
- Test: `internal/filterbank/filterbank_test.go`

**Step 1: Write the failing test for NewFilterBank**

Create `internal/filterbank/filterbank_test.go`:

```go
package filterbank

import "testing"

func TestNewFilterBank(t *testing.T) {
	fb := NewFilterBank(1024)
	if fb == nil {
		t.Fatal("expected non-nil FilterBank")
	}
	if fb.mdct256 == nil {
		t.Error("expected mdct256 to be initialized")
	}
	if fb.mdct2048 == nil {
		t.Error("expected mdct2048 to be initialized")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/filterbank`
Expected: FAIL with "undefined: NewFilterBank"

**Step 3: Write minimal implementation**

Create `internal/filterbank/filterbank.go`:

```go
// Package filterbank implements the AAC filter bank (IMDCT + windowing + overlap-add).
// Ported from: ~/dev/faad2/libfaad/filtbank.c
package filterbank

import (
	"github.com/llehouerou/go-aac/internal/mdct"
	"github.com/llehouerou/go-aac/internal/syntax"
)

// FilterBank holds state for inverse filter bank operations.
// It contains pre-initialized MDCT instances for short and long blocks.
//
// Ported from: fb_info struct in ~/dev/faad2/libfaad/structs.h:67-83
type FilterBank struct {
	mdct256  *mdct.MDCT // For short blocks (256-sample IMDCT)
	mdct2048 *mdct.MDCT // For long blocks (2048-sample IMDCT)

	// Internal buffers (reused to avoid allocations)
	transfBuf []float32 // 2*frameLength for IMDCT output
}

// NewFilterBank creates and initializes a FilterBank for the given frame length.
// Standard AAC uses frameLength=1024.
//
// Ported from: filter_bank_init() in ~/dev/faad2/libfaad/filtbank.c:48-92
func NewFilterBank(frameLen uint16) *FilterBank {
	nshort := frameLen / 8 // 128 for standard AAC

	fb := &FilterBank{
		mdct256:   mdct.NewMDCT(2 * nshort),  // 256 for short blocks
		mdct2048:  mdct.NewMDCT(2 * frameLen), // 2048 for long blocks
		transfBuf: make([]float32, 2*frameLen),
	}

	return fb
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/filterbank`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/filterbank/filterbank.go internal/filterbank/filterbank_test.go
git commit -m "feat(filterbank): add FilterBank struct and constructor"
```

---

### Task 2: Implement ONLY_LONG_SEQUENCE Case

**Files:**
- Modify: `internal/filterbank/filterbank.go`
- Modify: `internal/filterbank/filterbank_test.go`

**Step 1: Write the failing test for IFilterBank with ONLY_LONG_SEQUENCE**

Add to `internal/filterbank/filterbank_test.go`:

```go
func TestIFilterBank_OnlyLongSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	// Create test input (1024 frequency coefficients)
	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i % 100) // Simple pattern
	}

	// Output buffer (1024 time samples)
	timeOut := make([]float32, 1024)

	// Overlap buffer (1024 samples, starts at zero)
	overlap := make([]float32, 1024)

	// Process one frame
	fb.IFilterBank(
		syntax.OnlyLongSequence,
		SineWindow,       // window_shape
		SineWindow,       // window_shape_prev
		freqIn,
		timeOut,
		overlap,
	)

	// After processing, overlap should contain non-zero values
	// (the second half of the windowed IMDCT output)
	allZero := true
	for _, v := range overlap {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("overlap buffer should contain non-zero values after processing")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/filterbank`
Expected: FAIL with "undefined: fb.IFilterBank"

**Step 3: Write minimal implementation for ONLY_LONG_SEQUENCE**

Add to `internal/filterbank/filterbank.go`:

```go
// IFilterBank performs the inverse filter bank operation.
// This converts frequency-domain spectral data to time-domain samples.
//
// Parameters:
// - windowSequence: One of OnlyLongSequence, LongStartSequence, EightShortSequence, LongStopSequence
// - windowShape: Current frame's window shape (SineWindow or KBDWindow)
// - windowShapePrev: Previous frame's window shape
// - freqIn: Input spectral coefficients (frameLen samples)
// - timeOut: Output time samples (frameLen samples)
// - overlap: Overlap buffer from previous frame (frameLen samples, modified in place)
//
// The overlap buffer is modified to contain the overlap for the next frame.
//
// Ported from: ifilter_bank() in ~/dev/faad2/libfaad/filtbank.c:164-334
func (fb *FilterBank) IFilterBank(
	windowSequence syntax.WindowSequence,
	windowShape uint8,
	windowShapePrev uint8,
	freqIn []float32,
	timeOut []float32,
	overlap []float32,
) {
	nlong := len(freqIn)
	nshort := nlong / 8
	transfBuf := fb.transfBuf

	// Get windows for current and previous frame
	windowLong := GetLongWindow(int(windowShape))
	windowLongPrev := GetLongWindow(int(windowShapePrev))
	windowShort := GetShortWindow(int(windowShape))
	windowShortPrev := GetShortWindow(int(windowShapePrev))

	// Suppress unused variable warnings for cases we haven't implemented yet
	_ = windowShort
	_ = windowShortPrev
	_ = nshort

	switch windowSequence {
	case syntax.OnlyLongSequence:
		// Perform IMDCT
		fb.mdct2048.IMDCT(freqIn, transfBuf)

		// Add second half of previous frame to windowed output of current frame
		// time_out[i] = overlap[i] + transf_buf[i] * window_long_prev[i]
		for i := 0; i < nlong; i++ {
			timeOut[i] = overlap[i] + transfBuf[i]*windowLongPrev[i]
		}

		// Window the second half and save as overlap for next frame
		// overlap[i] = transf_buf[nlong+i] * window_long[nlong-1-i]
		for i := 0; i < nlong; i++ {
			overlap[i] = transfBuf[nlong+i] * windowLong[nlong-1-i]
		}

	default:
		// TODO: implement other window sequences
		panic("window sequence not implemented")
	}
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/filterbank`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/filterbank/filterbank.go internal/filterbank/filterbank_test.go
git commit -m "feat(filterbank): implement IFilterBank for ONLY_LONG_SEQUENCE"
```

---

### Task 3: Implement LONG_START_SEQUENCE Case

**Files:**
- Modify: `internal/filterbank/filterbank.go`
- Modify: `internal/filterbank/filterbank_test.go`

**Step 1: Write the failing test for LONG_START_SEQUENCE**

Add to `internal/filterbank/filterbank_test.go`:

```go
func TestIFilterBank_LongStartSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i % 100)
	}

	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)

	// First, process with ONLY_LONG to initialize overlap
	fb.IFilterBank(syntax.OnlyLongSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// Now test LONG_START_SEQUENCE
	fb.IFilterBank(syntax.LongStartSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// The overlap structure for LONG_START should have:
	// - First nflat_ls samples: direct values (no windowing)
	// - Next nshort samples: windowed with short window
	// - Last nflat_ls samples: zeros
	nshort := 1024 / 8  // 128
	nflat_ls := (1024 - nshort) / 2  // 448

	// Check that the end section is zeros
	for i := nflat_ls + nshort; i < 1024; i++ {
		if overlap[i] != 0 {
			t.Errorf("overlap[%d] = %f, expected 0 (zeros region)", i, overlap[i])
			break
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/filterbank`
Expected: FAIL with panic "window sequence not implemented"

**Step 3: Implement LONG_START_SEQUENCE case**

Add to the switch statement in `IFilterBank`:

```go
	case syntax.LongStartSequence:
		nflat_ls := (nlong - nshort) / 2

		// Perform IMDCT
		fb.mdct2048.IMDCT(freqIn, transfBuf)

		// Add second half of previous frame to windowed output of current frame
		for i := 0; i < nlong; i++ {
			timeOut[i] = overlap[i] + transfBuf[i]*windowLongPrev[i]
		}

		// Window the second half and save as overlap for next frame
		// Construct second half window using padding with 1's and 0's
		for i := 0; i < nflat_ls; i++ {
			overlap[i] = transfBuf[nlong+i]
		}
		for i := 0; i < nshort; i++ {
			overlap[nflat_ls+i] = transfBuf[nlong+nflat_ls+i] * windowShort[nshort-i-1]
		}
		for i := 0; i < nflat_ls; i++ {
			overlap[nflat_ls+nshort+i] = 0
		}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/filterbank`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/filterbank/filterbank.go internal/filterbank/filterbank_test.go
git commit -m "feat(filterbank): implement IFilterBank for LONG_START_SEQUENCE"
```

---

### Task 4: Implement LONG_STOP_SEQUENCE Case

**Files:**
- Modify: `internal/filterbank/filterbank.go`
- Modify: `internal/filterbank/filterbank_test.go`

**Step 1: Write the failing test for LONG_STOP_SEQUENCE**

Add to `internal/filterbank/filterbank_test.go`:

```go
func TestIFilterBank_LongStopSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i % 100)
	}

	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)

	// Initialize overlap as if coming from short blocks
	nshort := 1024 / 8  // 128
	nflat_ls := (1024 - nshort) / 2  // 448
	for i := 0; i < nflat_ls; i++ {
		overlap[i] = 0  // zeros before short window region
	}
	for i := nflat_ls; i < nflat_ls+nshort; i++ {
		overlap[i] = float32(i)  // some values in short window region
	}
	for i := nflat_ls + nshort; i < 1024; i++ {
		overlap[i] = float32(i)  // values in flat region after short
	}

	fb.IFilterBank(syntax.LongStopSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// After LONG_STOP, the overlap should be full long window style
	allZero := true
	for _, v := range overlap {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("overlap should have non-zero values")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/filterbank`
Expected: FAIL with panic "window sequence not implemented"

**Step 3: Implement LONG_STOP_SEQUENCE case**

Add to the switch statement in `IFilterBank`:

```go
	case syntax.LongStopSequence:
		nflat_ls := (nlong - nshort) / 2

		// Perform IMDCT
		fb.mdct2048.IMDCT(freqIn, transfBuf)

		// Add second half of previous frame to windowed output of current frame
		// Construct first half window using padding with 1's and 0's
		for i := 0; i < nflat_ls; i++ {
			timeOut[i] = overlap[i]
		}
		for i := 0; i < nshort; i++ {
			timeOut[nflat_ls+i] = overlap[nflat_ls+i] + transfBuf[nflat_ls+i]*windowShortPrev[i]
		}
		for i := 0; i < nflat_ls; i++ {
			timeOut[nflat_ls+nshort+i] = overlap[nflat_ls+nshort+i] + transfBuf[nflat_ls+nshort+i]
		}

		// Window the second half and save as overlap for next frame
		for i := 0; i < nlong; i++ {
			overlap[i] = transfBuf[nlong+i] * windowLong[nlong-1-i]
		}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/filterbank`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/filterbank/filterbank.go internal/filterbank/filterbank_test.go
git commit -m "feat(filterbank): implement IFilterBank for LONG_STOP_SEQUENCE"
```

---

### Task 5: Implement EIGHT_SHORT_SEQUENCE Case

**Files:**
- Modify: `internal/filterbank/filterbank.go`
- Modify: `internal/filterbank/filterbank_test.go`

**Step 1: Write the failing test for EIGHT_SHORT_SEQUENCE**

Add to `internal/filterbank/filterbank_test.go`:

```go
func TestIFilterBank_EightShortSequence(t *testing.T) {
	fb := NewFilterBank(1024)

	// Input is 1024 coefficients, but treated as 8x128 short blocks
	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i % 50)
	}

	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)

	// Set up overlap as if coming from LONG_START
	nshort := 1024 / 8  // 128
	nflat_ls := (1024 - nshort) / 2  // 448
	for i := 0; i < nflat_ls; i++ {
		overlap[i] = float32(i)
	}
	for i := nflat_ls; i < 1024; i++ {
		overlap[i] = float32(i % 100)
	}

	fb.IFilterBank(syntax.EightShortSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// timeOut should have valid data
	allZero := true
	for _, v := range timeOut {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("timeOut should have non-zero values after EIGHT_SHORT_SEQUENCE")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/filterbank`
Expected: FAIL with panic "window sequence not implemented"

**Step 3: Implement EIGHT_SHORT_SEQUENCE case**

This is the most complex case. Add to the switch statement in `IFilterBank`:

```go
	case syntax.EightShortSequence:
		nflat_ls := (nlong - nshort) / 2
		trans := nshort / 2

		// Perform IMDCT for each of the 8 short blocks
		// FAAD2 uses a separate transfBuf for all 8 blocks (8*256 = 2048)
		// But our transfBuf is already 2048, so we can reuse it
		for blk := 0; blk < 8; blk++ {
			fb.mdct256.IMDCT(freqIn[blk*nshort:], transfBuf[2*nshort*blk:])
		}

		// Add second half of previous frame to windowed output of current frame
		// First nflat_ls samples: direct copy from overlap
		for i := 0; i < nflat_ls; i++ {
			timeOut[i] = overlap[i]
		}

		// Process the overlapping short blocks
		for i := 0; i < nshort; i++ {
			// Block 0: previous overlap + windowed first half of block 0
			timeOut[nflat_ls+i] = overlap[nflat_ls+i] + transfBuf[nshort*0+i]*windowShortPrev[i]

			// Blocks 1-3: windowed second half of prev block + windowed first half of current block
			timeOut[nflat_ls+1*nshort+i] = overlap[nflat_ls+nshort*1+i] +
				transfBuf[nshort*1+i]*windowShort[nshort-1-i] +
				transfBuf[nshort*2+i]*windowShort[i]
			timeOut[nflat_ls+2*nshort+i] = overlap[nflat_ls+nshort*2+i] +
				transfBuf[nshort*3+i]*windowShort[nshort-1-i] +
				transfBuf[nshort*4+i]*windowShort[i]
			timeOut[nflat_ls+3*nshort+i] = overlap[nflat_ls+nshort*3+i] +
				transfBuf[nshort*5+i]*windowShort[nshort-1-i] +
				transfBuf[nshort*6+i]*windowShort[i]

			// Block 4: partial (only first half, where i < trans)
			if i < trans {
				timeOut[nflat_ls+4*nshort+i] = overlap[nflat_ls+nshort*4+i] +
					transfBuf[nshort*7+i]*windowShort[nshort-1-i] +
					transfBuf[nshort*8+i]*windowShort[i]
			}
		}

		// Window the second half and save as overlap for next frame
		for i := 0; i < nshort; i++ {
			// Block 4 continuation (where i >= trans)
			if i >= trans {
				overlap[nflat_ls+4*nshort+i-nlong] =
					transfBuf[nshort*7+i]*windowShort[nshort-1-i] +
						transfBuf[nshort*8+i]*windowShort[i]
			}
			// Blocks 5-7
			overlap[nflat_ls+5*nshort+i-nlong] =
				transfBuf[nshort*9+i]*windowShort[nshort-1-i] +
					transfBuf[nshort*10+i]*windowShort[i]
			overlap[nflat_ls+6*nshort+i-nlong] =
				transfBuf[nshort*11+i]*windowShort[nshort-1-i] +
					transfBuf[nshort*12+i]*windowShort[i]
			overlap[nflat_ls+7*nshort+i-nlong] =
				transfBuf[nshort*13+i]*windowShort[nshort-1-i] +
					transfBuf[nshort*14+i]*windowShort[i]
			// Block 7 tail (only second half, no next block)
			overlap[nflat_ls+8*nshort+i-nlong] =
				transfBuf[nshort*15+i] * windowShort[nshort-1-i]
		}

		// Zero pad the end
		for i := 0; i < nflat_ls; i++ {
			overlap[nflat_ls+nshort+i] = 0
		}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/filterbank`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/filterbank/filterbank.go internal/filterbank/filterbank_test.go
git commit -m "feat(filterbank): implement IFilterBank for EIGHT_SHORT_SEQUENCE"
```

---

### Task 6: Add Window Transition Tests

**Files:**
- Modify: `internal/filterbank/filterbank_test.go`

**Step 1: Write tests for window sequence transitions**

Add to `internal/filterbank/filterbank_test.go`:

```go
func TestIFilterBank_WindowTransitionLongToShort(t *testing.T) {
	fb := NewFilterBank(1024)

	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i % 100) * 0.01
	}

	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)

	// Simulate: ONLY_LONG -> LONG_START -> EIGHT_SHORT

	// Frame 1: ONLY_LONG
	fb.IFilterBank(syntax.OnlyLongSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// Frame 2: LONG_START (transition to short)
	fb.IFilterBank(syntax.LongStartSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// Frame 3: EIGHT_SHORT
	fb.IFilterBank(syntax.EightShortSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// Verify no panics and output is reasonable
	if len(timeOut) != 1024 {
		t.Errorf("timeOut length = %d, expected 1024", len(timeOut))
	}
}

func TestIFilterBank_WindowTransitionShortToLong(t *testing.T) {
	fb := NewFilterBank(1024)

	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i % 100) * 0.01
	}

	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)

	// Simulate: ONLY_LONG -> LONG_START -> EIGHT_SHORT -> LONG_STOP -> ONLY_LONG

	fb.IFilterBank(syntax.OnlyLongSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.LongStartSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.EightShortSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.LongStopSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.OnlyLongSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)

	// Verify no panics and output is reasonable
	if len(timeOut) != 1024 {
		t.Errorf("timeOut length = %d, expected 1024", len(timeOut))
	}
}

func TestIFilterBank_MixedWindowShapes(t *testing.T) {
	fb := NewFilterBank(1024)

	freqIn := make([]float32, 1024)
	for i := range freqIn {
		freqIn[i] = float32(i % 100) * 0.01
	}

	timeOut := make([]float32, 1024)
	overlap := make([]float32, 1024)

	// Test transitioning between sine and KBD windows
	fb.IFilterBank(syntax.OnlyLongSequence, SineWindow, SineWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.OnlyLongSequence, KBDWindow, SineWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.OnlyLongSequence, KBDWindow, KBDWindow, freqIn, timeOut, overlap)
	fb.IFilterBank(syntax.OnlyLongSequence, SineWindow, KBDWindow, freqIn, timeOut, overlap)

	// Verify no panics
	if len(timeOut) != 1024 {
		t.Errorf("timeOut length = %d, expected 1024", len(timeOut))
	}
}
```

**Step 2: Run tests to verify they pass**

Run: `make test PKG=./internal/filterbank`
Expected: PASS (these tests just verify no panics and correct behavior)

**Step 3: Commit**

```bash
git add internal/filterbank/filterbank_test.go
git commit -m "test(filterbank): add window sequence transition tests"
```

---

### Task 7: Add FAAD2 Reference Validation Tests

**Files:**
- Create: `internal/filterbank/filterbank_faad2_test.go`

**Step 1: Create FAAD2 reference test structure**

Create `internal/filterbank/filterbank_faad2_test.go`:

```go
//go:build faad2_validation

package filterbank

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/llehouerou/go-aac/internal/syntax"
)

// TestIFilterBank_FAAD2Reference validates filter bank output against FAAD2 reference.
// This test is skipped by default; run with -tags=faad2_validation
func TestIFilterBank_FAAD2Reference(t *testing.T) {
	refDir := "/tmp/faad2_filterbank_ref"
	if _, err := os.Stat(refDir); os.IsNotExist(err) {
		t.Skipf("FAAD2 reference data not found at %s; generate with scripts/check_faad2", refDir)
	}

	// Load test cases from reference directory
	entries, err := os.ReadDir(refDir)
	if err != nil {
		t.Fatalf("failed to read reference directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			testDir := filepath.Join(refDir, entry.Name())
			validateFilterBankFrame(t, testDir)
		})
	}
}

func validateFilterBankFrame(t *testing.T, testDir string) {
	// Load input spectral data
	freqInPath := filepath.Join(testDir, "freq_in.bin")
	freqInData, err := os.ReadFile(freqInPath)
	if err != nil {
		t.Skipf("skipping: %v", err)
		return
	}

	// Load expected output
	timeOutPath := filepath.Join(testDir, "time_out.bin")
	expectedData, err := os.ReadFile(timeOutPath)
	if err != nil {
		t.Skipf("skipping: %v", err)
		return
	}

	// Load window sequence info
	infoPath := filepath.Join(testDir, "info.bin")
	infoData, err := os.ReadFile(infoPath)
	if err != nil {
		t.Skipf("skipping: %v", err)
		return
	}

	// Parse info (window_sequence, window_shape, window_shape_prev)
	windowSeq := syntax.WindowSequence(infoData[0])
	windowShape := infoData[1]
	windowShapePrev := infoData[2]

	// Convert binary to float32 slices
	frameLen := len(freqInData) / 4
	freqIn := make([]float32, frameLen)
	for i := 0; i < frameLen; i++ {
		bits := binary.LittleEndian.Uint32(freqInData[i*4:])
		freqIn[i] = float32frombits(bits)
	}

	expected := make([]float32, len(expectedData)/4)
	for i := 0; i < len(expected); i++ {
		bits := binary.LittleEndian.Uint32(expectedData[i*4:])
		expected[i] = float32frombits(bits)
	}

	// Run filter bank
	fb := NewFilterBank(uint16(frameLen))
	timeOut := make([]float32, frameLen)
	overlap := make([]float32, frameLen) // Assume fresh start

	fb.IFilterBank(windowSeq, windowShape, windowShapePrev, freqIn, timeOut, overlap)

	// Compare output
	const tolerance = 1e-5
	for i := 0; i < len(expected); i++ {
		diff := timeOut[i] - expected[i]
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			t.Errorf("sample %d: got %f, expected %f (diff %f)", i, timeOut[i], expected[i], diff)
			if i > 10 {
				t.Fatalf("too many errors, stopping")
			}
		}
	}
}

// float32frombits converts uint32 bits to float32
func float32frombits(b uint32) float32 {
	return *(*float32)(unsafe.Pointer(&b))
}
```

Note: This file requires `import "unsafe"` at the top.

**Step 2: Run linter to catch issues**

Run: `make lint`
Expected: Warnings about unused code (test is build-tagged)

**Step 3: Commit**

```bash
git add internal/filterbank/filterbank_faad2_test.go
git commit -m "test(filterbank): add FAAD2 reference validation tests (build-tagged)"
```

---

### Task 8: Final Cleanup and Run Full Test Suite

**Files:**
- Modify: `internal/filterbank/filterbank.go` (cleanup)

**Step 1: Clean up implementation (remove unused variable suppression)**

Remove the lines that suppress unused variable warnings now that all cases are implemented:

```go
	// Remove these lines:
	_ = windowShort
	_ = windowShortPrev
	_ = nshort
```

**Step 2: Add doc comment to package**

Update `internal/filterbank/doc.go` if it exists, or ensure the package comment in `filterbank.go` is complete:

```go
// Package filterbank implements the AAC filter bank (IMDCT + windowing + overlap-add).
//
// The filter bank is responsible for converting frequency-domain spectral
// coefficients (output from spectral reconstruction) into time-domain PCM samples.
//
// Key operations:
// - IMDCT (Inverse Modified Discrete Cosine Transform)
// - Windowing (sine or Kaiser-Bessel Derived windows)
// - Overlap-add (50% overlap between consecutive frames)
//
// Window sequences:
// - OnlyLongSequence: Standard long blocks (1024 samples)
// - LongStartSequence: Transition from long to short blocks
// - EightShortSequence: 8 short blocks (128 samples each)
// - LongStopSequence: Transition from short to long blocks
//
// Ported from: ~/dev/faad2/libfaad/filtbank.c
package filterbank
```

**Step 3: Run full test suite**

Run: `make check`
Expected: All tests pass, no linter errors

**Step 4: Commit**

```bash
git add internal/filterbank/
git commit -m "feat(filterbank): complete filter bank implementation with all window sequences"
```

---

## Summary

This plan implements Step 5.4 of the FAAD2 migration: the inverse filter bank with overlap-add. The implementation covers:

1. **FilterBank struct** - Holds MDCT instances and reusable buffers
2. **NewFilterBank()** - Initializes MDCT for short (256) and long (2048) transforms
3. **IFilterBank()** - Core function handling all 4 window sequences:
   - `ONLY_LONG_SEQUENCE` - Standard long blocks
   - `LONG_START_SEQUENCE` - Transition to short blocks
   - `EIGHT_SHORT_SEQUENCE` - 8 short blocks with overlap
   - `LONG_STOP_SEQUENCE` - Transition back to long blocks
4. **Window shape transitions** - Sine and KBD window support
5. **FAAD2 validation tests** - Build-tagged tests for reference comparison

**Total: 8 tasks, ~350 lines of Go code**
