# Pulse Data Decoder Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement `PulseDecode()` function that applies pulse amplitudes to spectral coefficients.

**Architecture:** The pulse decoder adds amplitude values to specific spectral coefficients based on parsed pulse data. It operates on quantized spectral data before inverse quantization, modifying coefficients at positions determined by pulse offsets from a starting SFB.

**Tech Stack:** Pure Go, no external dependencies.

---

## Background

Pulse coding is used in AAC to efficiently encode transients/attacks. Up to 4 pulses can modify spectral coefficients in long blocks only. The pulse data contains:
- `pulse_start_sfb`: Starting scale factor band
- `number_pulse`: Number of pulses - 1 (0-3)
- `pulse_offset[i]`: Offset from previous position
- `pulse_amp[i]`: Amplitude to add/subtract

**FAAD2 Source:** `~/dev/faad2/libfaad/pulse.c:36-58`

---

### Task 1: Add Pulse Position Error

**Files:**
- Modify: `internal/syntax/errors.go`

**Step 1: Add error definition**

Add to the "Pulse data errors" section in `errors.go`:

```go
// ErrPulsePosition indicates pulse position exceeds frame length.
// FAAD2 error code: 15
ErrPulsePosition = errors.New("syntax: pulse position exceeds frame length")
```

**Step 2: Verify compilation**

Run: `go build ./internal/syntax/...`
Expected: Success

**Step 3: Commit**

```bash
git add internal/syntax/errors.go
git commit -m "feat(syntax): add ErrPulsePosition error

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Write Failing Test for PulseDecode

**Files:**
- Create: `internal/spectrum/pulse_test.go`

**Step 1: Write the failing test**

```go
package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestPulseDecode_SinglePulse(t *testing.T) {
	// Set up ICS with SFB offsets (simulating 44.1kHz long window)
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	// SFB 2 starts at position 8
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffset[3] = 12

	// Set up pulse info: 1 pulse at SFB 2, offset 2, amplitude 5
	ics.Pul = syntax.PulseInfo{
		NumberPulse:   0, // 0 means 1 pulse
		PulseStartSFB: 2,
		PulseOffset:   [4]uint8{2, 0, 0, 0}, // Position = 8 + 2 = 10
		PulseAmp:      [4]uint8{5, 0, 0, 0},
	}

	// Spectral data: positive value at position 10
	specData := make([]int16, 1024)
	specData[10] = 100

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Value should be increased by pulse amplitude
	if specData[10] != 105 {
		t.Errorf("specData[10]: got %d, want 105", specData[10])
	}
}

func TestPulseDecode_NegativeValue(t *testing.T) {
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	ics.SWBOffset[2] = 8

	ics.Pul = syntax.PulseInfo{
		NumberPulse:   0,
		PulseStartSFB: 2,
		PulseOffset:   [4]uint8{0, 0, 0, 0}, // Position = 8
		PulseAmp:      [4]uint8{3, 0, 0, 0},
	}

	specData := make([]int16, 1024)
	specData[8] = -50 // Negative value

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Negative value should be decreased (more negative)
	if specData[8] != -53 {
		t.Errorf("specData[8]: got %d, want -53", specData[8])
	}
}

func TestPulseDecode_MultiplePulses(t *testing.T) {
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	ics.SWBOffset[0] = 0

	// 4 pulses (number_pulse = 3)
	ics.Pul = syntax.PulseInfo{
		NumberPulse:   3,
		PulseStartSFB: 0,
		PulseOffset:   [4]uint8{5, 3, 2, 10}, // Positions: 5, 8, 10, 20
		PulseAmp:      [4]uint8{1, 2, 3, 4},
	}

	specData := make([]int16, 1024)
	specData[5] = 10
	specData[8] = 20
	specData[10] = 30
	specData[20] = -40

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if specData[5] != 11 {
		t.Errorf("specData[5]: got %d, want 11", specData[5])
	}
	if specData[8] != 22 {
		t.Errorf("specData[8]: got %d, want 22", specData[8])
	}
	if specData[10] != 33 {
		t.Errorf("specData[10]: got %d, want 33", specData[10])
	}
	if specData[20] != -44 {
		t.Errorf("specData[20]: got %d, want -44", specData[20])
	}
}

func TestPulseDecode_ZeroValue(t *testing.T) {
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	ics.SWBOffset[0] = 0

	ics.Pul = syntax.PulseInfo{
		NumberPulse:   0,
		PulseStartSFB: 0,
		PulseOffset:   [4]uint8{0, 0, 0, 0},
		PulseAmp:      [4]uint8{7, 0, 0, 0},
	}

	specData := make([]int16, 1024)
	specData[0] = 0 // Zero value - should be treated as positive

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Zero is not > 0, so it goes to else branch (subtract)
	// But in FAAD2, the condition is spec_data[k] > 0
	// So zero values subtract the amplitude
	if specData[0] != -7 {
		t.Errorf("specData[0]: got %d, want -7", specData[0])
	}
}

func TestPulseDecode_PositionExceedsFrame(t *testing.T) {
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	ics.SWBOffset[0] = 1020 // Near end of frame

	ics.Pul = syntax.PulseInfo{
		NumberPulse:   0,
		PulseStartSFB: 0,
		PulseOffset:   [4]uint8{10, 0, 0, 0}, // Position = 1020 + 10 = 1030 > 1024
		PulseAmp:      [4]uint8{1, 0, 0, 0},
	}

	specData := make([]int16, 1024)

	err := PulseDecode(ics, specData, 1024)
	if err != syntax.ErrPulsePosition {
		t.Errorf("expected ErrPulsePosition, got %v", err)
	}
}

func TestPulseDecode_SWBOffsetMaxClamp(t *testing.T) {
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 100, // Clamped to 100
	}
	ics.SWBOffset[5] = 200 // Larger than SWBOffsetMax

	ics.Pul = syntax.PulseInfo{
		NumberPulse:   0,
		PulseStartSFB: 5,
		PulseOffset:   [4]uint8{10, 0, 0, 0}, // k = min(200, 100) + 10 = 110
		PulseAmp:      [4]uint8{1, 0, 0, 0},
	}

	specData := make([]int16, 1024)
	specData[110] = 50

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if specData[110] != 51 {
		t.Errorf("specData[110]: got %d, want 51", specData[110])
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/spectrum/... -v`
Expected: FAIL with "undefined: PulseDecode"

---

### Task 3: Implement PulseDecode

**Files:**
- Create: `internal/spectrum/pulse.go`

**Step 1: Write minimal implementation**

```go
// Package spectrum implements spectral processing for AAC decoding.
//
// This includes inverse quantization, scale factor application,
// M/S stereo, intensity stereo, PNS, and TNS.
package spectrum

import "github.com/llehouerou/go-aac/internal/syntax"

// PulseDecode applies pulse data to spectral coefficients.
// Pulses add or subtract amplitude values at specific positions,
// used to efficiently encode transients and attacks.
//
// The function modifies specData in place. It should only be called
// for long blocks (pulse coding is not allowed in short blocks).
//
// Ported from: pulse_decode() in ~/dev/faad2/libfaad/pulse.c:36-58
func PulseDecode(ics *syntax.ICStream, specData []int16, frameLen uint16) error {
	pul := &ics.Pul

	// Start position is clamped to swb_offset_max
	k := ics.SWBOffset[pul.PulseStartSFB]
	if k > ics.SWBOffsetMax {
		k = ics.SWBOffsetMax
	}

	// Apply each pulse
	numPulses := pul.NumberPulse + 1
	for i := uint8(0); i < numPulses; i++ {
		k += uint16(pul.PulseOffset[i])

		if k >= frameLen {
			return syntax.ErrPulsePosition
		}

		if specData[k] > 0 {
			specData[k] += int16(pul.PulseAmp[i])
		} else {
			specData[k] -= int16(pul.PulseAmp[i])
		}
	}

	return nil
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/spectrum/... -v`
Expected: PASS

**Step 3: Run linting**

Run: `make lint`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/spectrum/pulse.go internal/spectrum/pulse_test.go
git commit -m "feat(spectrum): implement PulseDecode function

Applies pulse amplitudes to spectral coefficients at positions
determined by pulse offsets from a starting SFB. Pulses add to
positive coefficients and subtract from non-positive ones.

Ported from: pulse_decode() in FAAD2 pulse.c:36-58

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Add Edge Case Tests

**Files:**
- Modify: `internal/spectrum/pulse_test.go`

**Step 1: Add additional edge case tests**

Append to `pulse_test.go`:

```go
func TestPulseDecode_ExactFrameLength(t *testing.T) {
	// Test position exactly at frame length - 1 (valid)
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	ics.SWBOffset[0] = 1020

	ics.Pul = syntax.PulseInfo{
		NumberPulse:   0,
		PulseStartSFB: 0,
		PulseOffset:   [4]uint8{3, 0, 0, 0}, // Position = 1023
		PulseAmp:      [4]uint8{1, 0, 0, 0},
	}

	specData := make([]int16, 1024)
	specData[1023] = 10

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if specData[1023] != 11 {
		t.Errorf("specData[1023]: got %d, want 11", specData[1023])
	}
}

func TestPulseDecode_AccumulatingOffsets(t *testing.T) {
	// Verify offsets accumulate correctly
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	ics.SWBOffset[0] = 0

	// Each offset adds to previous position
	ics.Pul = syntax.PulseInfo{
		NumberPulse:   2, // 3 pulses
		PulseStartSFB: 0,
		PulseOffset:   [4]uint8{10, 10, 10, 0}, // Positions: 10, 20, 30
		PulseAmp:      [4]uint8{1, 1, 1, 0},
	}

	specData := make([]int16, 1024)
	specData[10] = 1
	specData[20] = 2
	specData[30] = 3

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if specData[10] != 2 {
		t.Errorf("specData[10]: got %d, want 2", specData[10])
	}
	if specData[20] != 3 {
		t.Errorf("specData[20]: got %d, want 3", specData[20])
	}
	if specData[30] != 4 {
		t.Errorf("specData[30]: got %d, want 4", specData[30])
	}
}

func TestPulseDecode_LargeAmplitude(t *testing.T) {
	// Test with maximum amplitude (15)
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	ics.SWBOffset[0] = 0

	ics.Pul = syntax.PulseInfo{
		NumberPulse:   0,
		PulseStartSFB: 0,
		PulseOffset:   [4]uint8{0, 0, 0, 0},
		PulseAmp:      [4]uint8{15, 0, 0, 0}, // Max amplitude (4 bits)
	}

	specData := make([]int16, 1024)
	specData[0] = 100

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if specData[0] != 115 {
		t.Errorf("specData[0]: got %d, want 115", specData[0])
	}
}
```

**Step 2: Run tests**

Run: `go test ./internal/spectrum/... -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/pulse_test.go
git commit -m "test(spectrum): add edge case tests for PulseDecode

- Exact frame length boundary
- Accumulating offsets verification
- Maximum amplitude handling

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Run Full Test Suite

**Step 1: Run all tests**

Run: `make check`
Expected: PASS

**Step 2: Verify no regressions**

All existing tests should still pass.

---

## Summary

After completing these tasks:
- `internal/spectrum/pulse.go` contains `PulseDecode()` function
- `internal/spectrum/pulse_test.go` contains comprehensive tests
- `internal/syntax/errors.go` has `ErrPulsePosition` error
- All tests pass and code is properly formatted/linted

The implementation matches FAAD2's `pulse_decode()` function exactly:
1. Start position k = min(swb_offset[pulse_start_sfb], swb_offset_max)
2. For each pulse: k += offset, then add/subtract amplitude based on sign
3. Return error if position exceeds frame length
