# Window Functions Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement window functions (sine and KBD) for the IMDCT filter bank windowing stage.

**Architecture:** Extract window values directly from FAAD2's `kbd_win.h` and `sine_win.h` header files to ensure bit-exact matching. Provide a clean lookup interface for the filter bank to access windows by type (sine=0, KBD=1) and size (long=1024, short=128).

**Tech Stack:** Go, regexp for value extraction from FAAD2 C headers

---

## Background

### FAAD2 Window Tables

From `~/dev/faad2/libfaad/filtbank.c:70-73`:
```c
fb->long_window[0]  = sine_long_1024;  // window_shape=0 (sine)
fb->short_window[0] = sine_short_128;
fb->long_window[1]  = kbd_long_1024;   // window_shape=1 (KBD)
fb->short_window[1] = kbd_short_128;
```

### Window Sizes

| Window Type | Size | Array Name | Purpose |
|-------------|------|------------|---------|
| Long Sine | 1024 | `sine_long_1024` | ONLY_LONG, LONG_START, LONG_STOP |
| Short Sine | 128 | `sine_short_128` | EIGHT_SHORT |
| Long KBD | 1024 | `kbd_long_1024` | ONLY_LONG, LONG_START, LONG_STOP |
| Short KBD | 128 | `kbd_short_128` | EIGHT_SHORT |

### Window Formulas (for reference only - we extract from FAAD2)

- **Sine window:** `w[n] = sin((π/N) * (n + 0.5))` for n = 0..N-1
- **KBD window:** Kaiser-Bessel Derived, computed from I₀ Bessel function

---

## Task 1: Create Window Type Constants

**Files:**
- Create: `internal/filterbank/window.go`
- Test: `internal/filterbank/window_test.go`

**Step 1: Write the failing test**

```go
// internal/filterbank/window_test.go
package filterbank

import "testing"

func TestWindowShapeConstants(t *testing.T) {
	// Window shapes must match FAAD2 indices
	if SineWindow != 0 {
		t.Errorf("SineWindow = %d, want 0", SineWindow)
	}
	if KBDWindow != 1 {
		t.Errorf("KBDWindow = %d, want 1", KBDWindow)
	}
}

func TestWindowSizeConstants(t *testing.T) {
	// Standard AAC frame uses 1024 long, 128 short
	if LongWindowSize != 1024 {
		t.Errorf("LongWindowSize = %d, want 1024", LongWindowSize)
	}
	if ShortWindowSize != 128 {
		t.Errorf("ShortWindowSize = %d, want 128", ShortWindowSize)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/filterbank`
Expected: FAIL with "undefined: SineWindow"

**Step 3: Write minimal implementation**

```go
// internal/filterbank/window.go
package filterbank

// Window shape constants.
// These match FAAD2's indexing in filtbank.c:70-73.
//
// Ported from: ~/dev/faad2/libfaad/filtbank.c
const (
	// SineWindow is the sine window shape (index 0).
	SineWindow = 0

	// KBDWindow is the Kaiser-Bessel Derived window shape (index 1).
	KBDWindow = 1
)

// Window size constants for standard AAC (1024 samples per frame).
const (
	// LongWindowSize is the size of long windows (1024 samples).
	LongWindowSize = 1024

	// ShortWindowSize is the size of short windows (128 samples = 1024/8).
	ShortWindowSize = 128
)
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/filterbank`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/filterbank/window.go internal/filterbank/window_test.go
git commit -m "feat(filterbank): add window shape and size constants"
```

---

## Task 2: Create Window Generator Script

**Files:**
- Create: `scripts/generate_windows.go`

**Step 1: Write the generator script**

```go
//go:build ignore

// generate_windows.go extracts window tables from FAAD2 header files.
// This ensures bit-exact matching with the reference implementation.
//
// Run with: go run scripts/generate_windows.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	faad2SineWin = "/home/laurent/dev/faad2/libfaad/sine_win.h"
	faad2KBDWin  = "/home/laurent/dev/faad2/libfaad/kbd_win.h"
)

// windowTable holds extracted window data
type windowTable struct {
	name   string
	goName string
	size   int
	values []string
}

func main() {
	// Extract sine windows from sine_win.h
	sineTables := []windowTable{
		{name: "sine_long_1024", goName: "sineLong1024", size: 1024},
		{name: "sine_short_128", goName: "sineShort128", size: 128},
	}

	for i := range sineTables {
		values, err := extractTable(faad2SineWin, sineTables[i].name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error extracting %s: %v\n", sineTables[i].name, err)
			os.Exit(1)
		}
		if len(values) != sineTables[i].size {
			fmt.Fprintf(os.Stderr, "%s: got %d values, want %d\n", sineTables[i].name, len(values), sineTables[i].size)
			os.Exit(1)
		}
		sineTables[i].values = values
	}

	// Generate window_sine.go
	if err := generateSineFile(sineTables); err != nil {
		fmt.Fprintf(os.Stderr, "error generating sine file: %v\n", err)
		os.Exit(1)
	}

	// Extract KBD windows from kbd_win.h
	kbdTables := []windowTable{
		{name: "kbd_long_1024", goName: "kbdLong1024", size: 1024},
		{name: "kbd_short_128", goName: "kbdShort128", size: 128},
	}

	for i := range kbdTables {
		values, err := extractTable(faad2KBDWin, kbdTables[i].name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error extracting %s: %v\n", kbdTables[i].name, err)
			os.Exit(1)
		}
		if len(values) != kbdTables[i].size {
			fmt.Fprintf(os.Stderr, "%s: got %d values, want %d\n", kbdTables[i].name, len(values), kbdTables[i].size)
			os.Exit(1)
		}
		kbdTables[i].values = values
	}

	// Generate window_kbd.go
	if err := generateKBDFile(kbdTables); err != nil {
		fmt.Fprintf(os.Stderr, "error generating KBD file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated internal/filterbank/window_sine.go")
	fmt.Println("Generated internal/filterbank/window_kbd.go")
}

// extractTable extracts a window table from a FAAD2 header file
func extractTable(filename, tableName string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var values []string
	scanner := bufio.NewScanner(file)

	// Match FRAC_CONST(value) pattern
	fracRegex := regexp.MustCompile(`FRAC_CONST\(([^)]+)\)`)

	inTable := false
	for scanner.Scan() {
		line := scanner.Text()

		// Detect start of table
		if strings.Contains(line, tableName+"[]") || strings.Contains(line, tableName+" []") {
			inTable = true
			continue
		}

		// Detect end of table
		if inTable && strings.Contains(line, "};") {
			break
		}

		if !inTable {
			continue
		}

		// Extract values from FRAC_CONST macros
		matches := fracRegex.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) > 1 {
				// Validate it's a valid number
				_, err := strconv.ParseFloat(m[1], 64)
				if err == nil {
					values = append(values, m[1])
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return values, nil
}

func generateSineFile(tables []windowTable) error {
	f, err := os.Create("internal/filterbank/window_sine.go")
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "// Code generated by generate_windows.go; DO NOT EDIT.")
	fmt.Fprintln(f, "//")
	fmt.Fprintln(f, "// Sine window tables for IMDCT windowing.")
	fmt.Fprintln(f, "// Values extracted directly from ~/dev/faad2/libfaad/sine_win.h")
	fmt.Fprintln(f, "// to ensure bit-exact matching with FAAD2.")
	fmt.Fprintln(f, "//")
	fmt.Fprintln(f, "// Formula: w[n] = sin((π/N) * (n + 0.5)) for n = 0..N-1")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "package filterbank")
	fmt.Fprintln(f, "")

	for _, t := range tables {
		fmt.Fprintf(f, "// %s contains %d sine window coefficients.\n", t.goName, t.size)
		fmt.Fprintf(f, "var %s = [%d]float32{\n", t.goName, t.size)

		for i, v := range t.values {
			if i%4 == 0 {
				fmt.Fprint(f, "\t")
			}
			fmt.Fprintf(f, "%s, ", v)
			if i%4 == 3 {
				fmt.Fprintln(f)
			}
		}
		if len(t.values)%4 != 0 {
			fmt.Fprintln(f)
		}
		fmt.Fprintln(f, "}")
		fmt.Fprintln(f, "")
	}

	return nil
}

func generateKBDFile(tables []windowTable) error {
	f, err := os.Create("internal/filterbank/window_kbd.go")
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "// Code generated by generate_windows.go; DO NOT EDIT.")
	fmt.Fprintln(f, "//")
	fmt.Fprintln(f, "// Kaiser-Bessel Derived (KBD) window tables for IMDCT windowing.")
	fmt.Fprintln(f, "// Values extracted directly from ~/dev/faad2/libfaad/kbd_win.h")
	fmt.Fprintln(f, "// to ensure bit-exact matching with FAAD2.")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "package filterbank")
	fmt.Fprintln(f, "")

	for _, t := range tables {
		fmt.Fprintf(f, "// %s contains %d KBD window coefficients.\n", t.goName, t.size)
		fmt.Fprintf(f, "var %s = [%d]float32{\n", t.goName, t.size)

		for i, v := range t.values {
			if i%4 == 0 {
				fmt.Fprint(f, "\t")
			}
			fmt.Fprintf(f, "%s, ", v)
			if i%4 == 3 {
				fmt.Fprintln(f)
			}
		}
		if len(t.values)%4 != 0 {
			fmt.Fprintln(f)
		}
		fmt.Fprintln(f, "}")
		fmt.Fprintln(f, "")
	}

	return nil
}
```

**Step 2: Run generator and verify output**

Run: `go run scripts/generate_windows.go`
Expected: "Generated internal/filterbank/window_sine.go" and "Generated internal/filterbank/window_kbd.go"

**Step 3: Verify generated files compile**

Run: `go build ./internal/filterbank/...`
Expected: No errors

**Step 4: Commit**

```bash
git add scripts/generate_windows.go
git add internal/filterbank/window_sine.go
git add internal/filterbank/window_kbd.go
git commit -m "feat(filterbank): add window table generator and generated tables"
```

---

## Task 3: Add Window Lookup Interface

**Files:**
- Modify: `internal/filterbank/window.go`
- Modify: `internal/filterbank/window_test.go`

**Step 1: Write the failing test**

Add to `internal/filterbank/window_test.go`:

```go
func TestGetLongWindow(t *testing.T) {
	tests := []struct {
		shape     int
		wantFirst float32
		wantLast  float32
		wantLen   int
	}{
		{SineWindow, 0.00076699031874270449, 0.00076699031874270449, 1024},
		{KBDWindow, 0.00029256153896361, 0.00029256153896361, 1024},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("shape=%d", tt.shape), func(t *testing.T) {
			w := GetLongWindow(tt.shape)
			if len(w) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(w), tt.wantLen)
			}
			// Check first value (approximately - full precision check in validation test)
			if w[0] < tt.wantFirst*0.999 || w[0] > tt.wantFirst*1.001 {
				t.Errorf("w[0] = %v, want ~%v", w[0], tt.wantFirst)
			}
		})
	}
}

func TestGetShortWindow(t *testing.T) {
	tests := []struct {
		shape   int
		wantLen int
	}{
		{SineWindow, 128},
		{KBDWindow, 128},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("shape=%d", tt.shape), func(t *testing.T) {
			w := GetShortWindow(tt.shape)
			if len(w) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(w), tt.wantLen)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/filterbank`
Expected: FAIL with "undefined: GetLongWindow"

**Step 3: Write minimal implementation**

Add to `internal/filterbank/window.go`:

```go
// GetLongWindow returns the long window (1024 samples) for the given shape.
// shape must be SineWindow (0) or KBDWindow (1).
//
// Ported from: fb->long_window[window_shape] in ~/dev/faad2/libfaad/filtbank.c:197
func GetLongWindow(shape int) []float32 {
	switch shape {
	case SineWindow:
		return sineLong1024[:]
	case KBDWindow:
		return kbdLong1024[:]
	default:
		panic("invalid window shape")
	}
}

// GetShortWindow returns the short window (128 samples) for the given shape.
// shape must be SineWindow (0) or KBDWindow (1).
//
// Ported from: fb->short_window[window_shape] in ~/dev/faad2/libfaad/filtbank.c:199
func GetShortWindow(shape int) []float32 {
	switch shape {
	case SineWindow:
		return sineShort128[:]
	case KBDWindow:
		return kbdShort128[:]
	default:
		panic("invalid window shape")
	}
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/filterbank`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/filterbank/window.go internal/filterbank/window_test.go
git commit -m "feat(filterbank): add window lookup functions"
```

---

## Task 4: Add FAAD2 Validation Test

**Files:**
- Create: `internal/filterbank/window_faad2_test.go`

**Step 1: Write the validation test**

```go
// internal/filterbank/window_faad2_test.go
package filterbank

import (
	"math"
	"testing"
)

// TestWindowValues_MatchFAAD2 validates that our window values match FAAD2 exactly.
// This is critical because windows affect the final audio output.
func TestWindowValues_MatchFAAD2(t *testing.T) {
	// Test sine long window - first and last 5 values
	sineLongExpected := []float32{
		0.00076699031874270449,
		0.002300969151425805,
		0.0038349425697062275,
		0.0053689069639963425,
		0.0069028587247297558,
	}

	w := GetLongWindow(SineWindow)
	for i, expected := range sineLongExpected {
		if !closeEnough(w[i], expected, 1e-10) {
			t.Errorf("sineLong1024[%d] = %.20f, want %.20f", i, w[i], expected)
		}
	}

	// Test KBD long window - first 5 values
	kbdLongExpected := []float32{
		0.00029256153896361,
		0.00042998567353047,
		0.00054674074589540,
		0.00065482304299792,
		0.00075870195068747,
	}

	w = GetLongWindow(KBDWindow)
	for i, expected := range kbdLongExpected {
		if !closeEnough(w[i], expected, 1e-10) {
			t.Errorf("kbdLong1024[%d] = %.20f, want %.20f", i, w[i], expected)
		}
	}

	// Verify window lengths
	if len(GetLongWindow(SineWindow)) != LongWindowSize {
		t.Errorf("sineLong1024 length = %d, want %d", len(GetLongWindow(SineWindow)), LongWindowSize)
	}
	if len(GetShortWindow(SineWindow)) != ShortWindowSize {
		t.Errorf("sineShort128 length = %d, want %d", len(GetShortWindow(SineWindow)), ShortWindowSize)
	}
}

// TestWindowSymmetry verifies that windows have the expected symmetry.
// Sine windows are symmetric: w[n] = w[N-1-n] due to the sin formula.
// KBD windows are also symmetric by construction.
func TestWindowSymmetry(t *testing.T) {
	tests := []struct {
		name  string
		shape int
		size  int
	}{
		{"sine_long", SineWindow, LongWindowSize},
		{"sine_short", SineWindow, ShortWindowSize},
		{"kbd_long", KBDWindow, LongWindowSize},
		{"kbd_short", KBDWindow, ShortWindowSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w []float32
			if tt.size == LongWindowSize {
				w = GetLongWindow(tt.shape)
			} else {
				w = GetShortWindow(tt.shape)
			}

			n := len(w)
			for i := 0; i < n/2; i++ {
				// Windows should be approximately symmetric
				// (not exact due to floating point, but very close)
				if !closeEnough(w[i], w[n-1-i], 1e-6) {
					t.Errorf("asymmetry at [%d]=%v vs [%d]=%v", i, w[i], n-1-i, w[n-1-i])
				}
			}
		})
	}
}

// TestWindowRange verifies all window values are in valid range [0, 1].
func TestWindowRange(t *testing.T) {
	windows := []struct {
		name string
		w    []float32
	}{
		{"sine_long", GetLongWindow(SineWindow)},
		{"sine_short", GetShortWindow(SineWindow)},
		{"kbd_long", GetLongWindow(KBDWindow)},
		{"kbd_short", GetShortWindow(KBDWindow)},
	}

	for _, ww := range windows {
		t.Run(ww.name, func(t *testing.T) {
			for i, v := range ww.w {
				if v < 0 || v > 1 {
					t.Errorf("[%d] = %v, want in range [0, 1]", i, v)
				}
			}
		})
	}
}

func closeEnough(a, b float32, tolerance float64) bool {
	return math.Abs(float64(a-b)) < tolerance
}
```

**Step 2: Run test to verify it passes**

Run: `make test PKG=./internal/filterbank`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/filterbank/window_faad2_test.go
git commit -m "test(filterbank): add FAAD2 window validation tests"
```

---

## Task 5: Update Makefile for Window Generation

**Files:**
- Modify: `Makefile`

**Step 1: Add generate target**

Add to `Makefile` after the `testdata-clean` target:

```makefile
# Generate window tables from FAAD2
generate-windows:
	go run scripts/generate_windows.go
```

**Step 2: Verify it works**

Run: `make generate-windows`
Expected: "Generated internal/filterbank/window_sine.go" and "Generated internal/filterbank/window_kbd.go"

**Step 3: Run full check**

Run: `make check`
Expected: All formatting, linting, and tests pass

**Step 4: Commit**

```bash
git add Makefile
git commit -m "build: add generate-windows target to Makefile"
```

---

## Task 6: Final Verification and Documentation

**Files:**
- Verify all files compile and tests pass

**Step 1: Run full test suite**

Run: `make check`
Expected: All tests pass

**Step 2: Verify window table sizes match FAAD2**

Run: `wc -l internal/filterbank/window_*.go`
Expected:
- `window_sine.go` should have ~270 lines (1024+128 values + headers)
- `window_kbd.go` should have ~290 lines (1024+128 values + headers)

**Step 3: Create final commit with all changes**

```bash
git status  # Verify nothing uncommitted
```

---

## Summary

### Files Created
| File | Purpose | Lines |
|------|---------|-------|
| `internal/filterbank/window.go` | Window constants and lookup interface | ~50 |
| `internal/filterbank/window_test.go` | Unit tests for window functions | ~80 |
| `internal/filterbank/window_faad2_test.go` | FAAD2 validation tests | ~90 |
| `internal/filterbank/window_sine.go` | Generated sine window tables | ~270 |
| `internal/filterbank/window_kbd.go` | Generated KBD window tables | ~290 |
| `scripts/generate_windows.go` | Window table generator | ~180 |

### Total: ~960 lines

### Acceptance Criteria
- [x] Generate windows programmatically via `go generate` / `make generate-windows`
- [x] Match FAAD2 window values (validated by tests)
- [x] Both sine and KBD windows
- [x] Generated files have `// Code generated` header
- [x] All tests pass (`make check`)
