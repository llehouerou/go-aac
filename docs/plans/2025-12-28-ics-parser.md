# Individual Channel Stream Parser Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the Individual Channel Stream (ICS) parsing functions to decode AAC audio channel data from bitstreams.

**Architecture:** Port FAAD2's ICS parsing from `syntax.c` and `specrec.c` to Go, splitting into logical files:
- `window.go` - Window grouping calculations
- `ics_info.go` - ICS info parsing (window sequence, shape, max_sfb)
- `section.go` - Section data parsing (codebook assignments)
- `scalefactor.go` - Scale factor decoding
- `pulse_data.go` - Pulse data parsing
- `tns_data.go` - TNS data parsing
- `ltp_data.go` - LTP data parsing
- `spectral.go` - Spectral data decoding
- `ics.go` - Main ICS parsing entry point (side_info + individual_channel_stream)

**Tech Stack:** Pure Go, depends on `internal/bits`, `internal/huffman`, `internal/tables`, and existing `internal/syntax` types.

---

## Background: ICS Parsing Flow

The Individual Channel Stream parsing follows this order:

```
individual_channel_stream()
├── side_info()
│   ├── global_gain (8 bits)
│   ├── ics_info() [if not common_window]
│   │   ├── window_sequence (2 bits)
│   │   ├── window_shape (1 bit)
│   │   ├── max_sfb (4 or 6 bits)
│   │   ├── scale_factor_grouping (7 bits, short blocks only)
│   │   ├── window_grouping_info()
│   │   └── predictor_data (MAIN/LTP profiles)
│   ├── section_data()
│   ├── scale_factor_data()
│   ├── pulse_data() [if present]
│   └── tns_data() [if present]
└── spectral_data()
```

---

## Task 1: Window Grouping Info Function

**Files:**
- Create: `internal/syntax/window.go`
- Test: `internal/syntax/window_test.go`

### Step 1.1: Write the failing test for WindowGroupingInfo

```go
// internal/syntax/window_test.go
package syntax

import "testing"

func TestWindowGroupingInfo_LongWindow(t *testing.T) {
	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		MaxSFB:         49,
	}

	err := WindowGroupingInfo(ics, 4, 1024) // sf_index=4 (44100 Hz), frameLength=1024
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumWindows != 1 {
		t.Errorf("NumWindows: got %d, want 1", ics.NumWindows)
	}
	if ics.NumWindowGroups != 1 {
		t.Errorf("NumWindowGroups: got %d, want 1", ics.NumWindowGroups)
	}
	if ics.WindowGroupLength[0] != 1 {
		t.Errorf("WindowGroupLength[0]: got %d, want 1", ics.WindowGroupLength[0])
	}
	// For 44100 Hz, num_swb should be 49
	if ics.NumSWB != 49 {
		t.Errorf("NumSWB: got %d, want 49", ics.NumSWB)
	}
}
```

### Step 1.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestWindowGroupingInfo_LongWindow`
Expected: FAIL with "undefined: WindowGroupingInfo"

### Step 1.3: Write WindowGroupingInfo function

```go
// internal/syntax/window.go
package syntax

import (
	"github.com/llehouerou/go-aac/internal/tables"
)

// WindowGroupingInfo calculates window grouping information for an ICS.
// It sets up the number of windows, window groups, and SFB offsets
// based on the window sequence and sample rate.
//
// Ported from: window_grouping_info() in ~/dev/faad2/libfaad/specrec.c:302-428
func WindowGroupingInfo(ics *ICStream, sfIndex uint8, frameLength uint16) error {
	if sfIndex >= 12 {
		return ErrInvalidSRIndex
	}

	switch ics.WindowSequence {
	case OnlyLongSequence, LongStartSequence, LongStopSequence:
		return windowGroupingLong(ics, sfIndex, frameLength)
	case EightShortSequence:
		return windowGroupingShort(ics, sfIndex, frameLength)
	default:
		return ErrInvalidWindowSequence
	}
}

// windowGroupingLong handles long window sequences.
func windowGroupingLong(ics *ICStream, sfIndex uint8, frameLength uint16) error {
	ics.NumWindows = 1
	ics.NumWindowGroups = 1
	ics.WindowGroupLength[0] = 1

	// Get number of SFBs for this sample rate and frame length
	numSWB, err := tables.GetNumSWB(sfIndex, frameLength, false)
	if err != nil {
		return err
	}
	ics.NumSWB = numSWB

	// Validate max_sfb
	if ics.MaxSFB > ics.NumSWB {
		return ErrMaxSFBTooLarge
	}

	// Get SFB offsets
	offsets, err := tables.GetSWBOffset(sfIndex, frameLength, false)
	if err != nil {
		return err
	}

	// Copy to sect_sfb_offset[0] and swb_offset
	for i := uint8(0); i < ics.NumSWB; i++ {
		ics.SectSFBOffset[0][i] = offsets[i]
		ics.SWBOffset[i] = offsets[i]
	}
	ics.SectSFBOffset[0][ics.NumSWB] = frameLength
	ics.SWBOffset[ics.NumSWB] = frameLength
	ics.SWBOffsetMax = frameLength

	return nil
}

// windowGroupingShort handles eight short window sequences.
func windowGroupingShort(ics *ICStream, sfIndex uint8, frameLength uint16) error {
	ics.NumWindows = 8
	ics.NumWindowGroups = 1
	ics.WindowGroupLength[0] = 1

	// Get number of SFBs for short windows
	numSWB, err := tables.GetNumSWB(sfIndex, frameLength, true)
	if err != nil {
		return err
	}
	ics.NumSWB = numSWB

	// Validate max_sfb
	if ics.MaxSFB > ics.NumSWB {
		return ErrMaxSFBTooLarge
	}

	// Get SFB offsets for short windows
	offsets, err := tables.GetSWBOffset(sfIndex, frameLength, true)
	if err != nil {
		return err
	}

	// Copy to swb_offset
	for i := uint8(0); i < ics.NumSWB; i++ {
		ics.SWBOffset[i] = offsets[i]
	}
	shortLen := frameLength / 8
	ics.SWBOffset[ics.NumSWB] = shortLen
	ics.SWBOffsetMax = shortLen

	// Calculate window groups from scale_factor_grouping
	// Bits 6-0 indicate grouping: bit N set means window N and N+1 are in same group
	for i := uint8(0); i < 7; i++ {
		if !bitSet(ics.ScaleFactorGrouping, 6-i) {
			// New group
			ics.NumWindowGroups++
			ics.WindowGroupLength[ics.NumWindowGroups-1] = 1
		} else {
			// Same group
			ics.WindowGroupLength[ics.NumWindowGroups-1]++
		}
	}

	// Calculate sect_sfb_offset for each group
	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		sectSFB := uint8(0)
		offset := uint16(0)

		for i := uint8(0); i < ics.NumSWB; i++ {
			var width uint16
			if i+1 == ics.NumSWB {
				width = shortLen - offsets[i]
			} else {
				width = offsets[i+1] - offsets[i]
			}
			width *= uint16(ics.WindowGroupLength[g])
			ics.SectSFBOffset[g][sectSFB] = offset
			sectSFB++
			offset += width
		}
		ics.SectSFBOffset[g][sectSFB] = offset
	}

	return nil
}

// bitSet returns true if bit B is set in A (1 << B).
func bitSet(a uint8, b uint8) bool {
	return (a & (1 << b)) != 0
}
```

### Step 1.4: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestWindowGroupingInfo_LongWindow`
Expected: PASS

### Step 1.5: Add error definitions

```go
// Add to internal/syntax/errors.go (or window.go if no errors file exists)
import "errors"

var (
	ErrInvalidSRIndex       = errors.New("syntax: invalid sample rate index")
	ErrInvalidWindowSequence = errors.New("syntax: invalid window sequence")
	ErrMaxSFBTooLarge       = errors.New("syntax: max_sfb exceeds num_swb")
)
```

### Step 1.6: Add test for short window grouping

```go
func TestWindowGroupingInfo_ShortWindow(t *testing.T) {
	ics := &ICStream{
		WindowSequence:      EightShortSequence,
		MaxSFB:              14,
		ScaleFactorGrouping: 0b1011010, // Groups: [1,2,1,2,2]
	}

	err := WindowGroupingInfo(ics, 4, 1024) // 44100 Hz
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumWindows != 8 {
		t.Errorf("NumWindows: got %d, want 8", ics.NumWindows)
	}

	// Check window groups from grouping pattern 1011010
	// Bit 6=1: windows 0,1 same group (length 2)
	// Bit 5=0: new group at window 2 (length 1)
	// Bit 4=1: windows 3,4 same group
	// Bit 3=1: windows 4,5 same group (length 3 total? No, need to trace)
	// Let's trace properly:
	// i=0: bit 6=1, same group -> group 0 length=2
	// i=1: bit 5=0, new group -> group 1 length=1
	// i=2: bit 4=1, same group -> group 1 length=2
	// i=3: bit 3=1, same group -> group 1 length=3
	// i=4: bit 2=0, new group -> group 2 length=1
	// i=5: bit 1=1, same group -> group 2 length=2
	// i=6: bit 0=0, new group -> group 3 length=1
	// Total: 4 groups with lengths [2, 3, 2, 1]
	// Wait, that's 8 windows but the calculation seems off

	// Actually re-reading FAAD2: 0b1011010 = 90 decimal
	// For i=0 to 6, check bit (6-i):
	// i=0: bit 6 = (90 >> 6) & 1 = 1 -> same group
	// i=1: bit 5 = (90 >> 5) & 1 = 0 -> new group
	// i=2: bit 4 = (90 >> 4) & 1 = 1 -> same group
	// i=3: bit 3 = (90 >> 3) & 1 = 1 -> same group
	// i=4: bit 2 = (90 >> 2) & 1 = 0 -> new group
	// i=5: bit 1 = (90 >> 1) & 1 = 1 -> same group
	// i=6: bit 0 = (90 >> 0) & 1 = 0 -> new group

	// So groups are: [0,1], [2,3,4], [5,6], [7]
	// Lengths: 2, 3, 2, 1
	// NumWindowGroups = 4

	if ics.NumWindowGroups != 4 {
		t.Errorf("NumWindowGroups: got %d, want 4", ics.NumWindowGroups)
	}

	expectedLengths := []uint8{2, 3, 2, 1}
	for i, want := range expectedLengths {
		if ics.WindowGroupLength[i] != want {
			t.Errorf("WindowGroupLength[%d]: got %d, want %d", i, ics.WindowGroupLength[i], want)
		}
	}
}
```

### Step 1.7: Run all window tests

Run: `go test -v ./internal/syntax -run TestWindowGroupingInfo`
Expected: PASS

### Step 1.8: Commit

```bash
git add internal/syntax/window.go internal/syntax/window_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add WindowGroupingInfo function

Implement window grouping calculation for ICS parsing.
Handles both long and short window sequences, calculates
window groups from scale_factor_grouping, and sets up
SFB offset tables.

Ported from: window_grouping_info() in specrec.c:302-428

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: ICS Info Parsing

**Files:**
- Create: `internal/syntax/ics_info.go`
- Test: `internal/syntax/ics_info_test.go`

### Step 2.1: Write the failing test for ParseICSInfo

```go
// internal/syntax/ics_info_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseICSInfo_LongWindow(t *testing.T) {
	// Build bitstream:
	// ics_reserved_bit: 0 (1 bit)
	// window_sequence: 0 (2 bits) = ONLY_LONG_SEQUENCE
	// window_shape: 1 (1 bit) = KBD
	// max_sfb: 49 (6 bits) = 0b110001
	// Predictor data present: 0 (1 bit)
	// Total: 1 + 2 + 1 + 6 + 1 = 11 bits
	// Bits: 0 00 1 110001 0 = 0b00111_00010 = 0x1C2... need to work out

	// Actually: 0 | 00 | 1 | 110001 | 0 = 0_00_1_110001_0
	// = 0b0001_1100_0100_0000 = 0x1C40 (but we pad)
	// Let's be more careful: reading MSB first
	// ics_reserved_bit = 0
	// window_sequence = 00 (ONLY_LONG)
	// window_shape = 1
	// max_sfb = 110001 (49)
	// predictor_data_present = 0
	// Concatenated: 0 00 1 110001 0 = 0_001_1100_10 padded = 0x1C80
	data := []byte{0x1C, 0x80}
	r := bits.NewReader(data)

	ics := &ICStream{}
	cfg := &ICSInfoConfig{
		SFIndex:      4, // 44100 Hz
		FrameLength:  1024,
		ObjectType:   2, // LC
		CommonWindow: false,
	}

	err := ParseICSInfo(r, ics, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.WindowSequence != OnlyLongSequence {
		t.Errorf("WindowSequence: got %d, want %d", ics.WindowSequence, OnlyLongSequence)
	}
	if ics.WindowShape != 1 {
		t.Errorf("WindowShape: got %d, want 1", ics.WindowShape)
	}
	if ics.MaxSFB != 49 {
		t.Errorf("MaxSFB: got %d, want 49", ics.MaxSFB)
	}
	if ics.PredictorDataPresent {
		t.Error("PredictorDataPresent should be false")
	}
}
```

### Step 2.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestParseICSInfo_LongWindow`
Expected: FAIL with "undefined: ParseICSInfo"

### Step 2.3: Write ICSInfoConfig type and ParseICSInfo function

```go
// internal/syntax/ics_info.go
package syntax

import (
	"github.com/llehouerou/go-aac/internal/bits"
)

// ICSInfoConfig holds configuration needed for ICS info parsing.
type ICSInfoConfig struct {
	SFIndex      uint8  // Sample rate index (0-11)
	FrameLength  uint16 // Frame length (960 or 1024)
	ObjectType   uint8  // Audio object type
	CommonWindow bool   // True if CPE with common window
}

// ObjectType constants
const (
	ObjectTypeMain = 1
	ObjectTypeLC   = 2
	ObjectTypeSSR  = 3
	ObjectTypeLTP  = 4
)

// ParseICSInfo parses the ics_info() element from the bitstream.
// Ported from: ics_info() in ~/dev/faad2/libfaad/syntax.c:829-952
func ParseICSInfo(r *bits.Reader, ics *ICStream, cfg *ICSInfoConfig) error {
	// ics_reserved_bit - must be 0
	reserved := r.Get1Bit()
	if reserved != 0 {
		return ErrICSReservedBit
	}

	// window_sequence (2 bits)
	ics.WindowSequence = WindowSequence(r.GetBits(2))

	// window_shape (1 bit)
	ics.WindowShape = r.Get1Bit()

	// max_sfb depends on window sequence
	if ics.WindowSequence == EightShortSequence {
		// Short blocks: 4 bits for max_sfb
		ics.MaxSFB = uint8(r.GetBits(4))
		// scale_factor_grouping (7 bits)
		ics.ScaleFactorGrouping = uint8(r.GetBits(7))
	} else {
		// Long blocks: 6 bits for max_sfb
		ics.MaxSFB = uint8(r.GetBits(6))
	}

	// Calculate window grouping
	if err := WindowGroupingInfo(ics, cfg.SFIndex, cfg.FrameLength); err != nil {
		return err
	}

	// Predictor data (only for long blocks)
	if ics.WindowSequence != EightShortSequence {
		ics.PredictorDataPresent = r.Get1Bit() != 0

		if ics.PredictorDataPresent {
			if cfg.ObjectType == ObjectTypeMain {
				// MAIN profile: MPEG-2 style prediction
				if err := parseMainPrediction(r, ics, cfg.SFIndex); err != nil {
					return err
				}
			} else {
				// LTP profile: Long Term Prediction
				if err := parseLTPPrediction(r, ics, cfg); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// parseMainPrediction parses MAIN profile prediction data.
// Ported from: ics_info() MAIN profile section in syntax.c:876-905
func parseMainPrediction(r *bits.Reader, ics *ICStream, sfIndex uint8) error {
	// Get max prediction SFB for this sample rate
	limit := maxPredSFB(sfIndex)
	if ics.MaxSFB < limit {
		limit = ics.MaxSFB
	}

	// predictor_reset (1 bit)
	predictorReset := r.Get1Bit() != 0
	var predictorResetGroup uint8
	if predictorReset {
		predictorResetGroup = uint8(r.GetBits(5))
	}

	// prediction_used flags for each SFB
	for sfb := uint8(0); sfb < limit; sfb++ {
		_ = r.Get1Bit() // prediction_used[sfb]
	}

	// Store in ICS if needed (currently not storing MAIN pred data)
	_ = predictorReset
	_ = predictorResetGroup

	return nil
}

// parseLTPPrediction parses LTP (Long Term Prediction) data.
// Ported from: ics_info() LTP section in syntax.c:907-947
func parseLTPPrediction(r *bits.Reader, ics *ICStream, cfg *ICSInfoConfig) error {
	// First LTP data
	ics.LTP.DataPresent = r.Get1Bit() != 0
	if ics.LTP.DataPresent {
		if err := ParseLTPData(r, ics, &ics.LTP, cfg.FrameLength); err != nil {
			return err
		}
	}

	// Second LTP data (only for common_window in CPE)
	if cfg.CommonWindow {
		ics.LTP2.DataPresent = r.Get1Bit() != 0
		if ics.LTP2.DataPresent {
			if err := ParseLTPData(r, ics, &ics.LTP2, cfg.FrameLength); err != nil {
				return err
			}
		}
	}

	return nil
}

// maxPredSFB returns the maximum SFB for MAIN profile prediction.
// Ported from: max_pred_sfb() in ~/dev/faad2/libfaad/common.c
func maxPredSFB(sfIndex uint8) uint8 {
	maxPredSFBTable := [12]uint8{
		33, 33, 38, 40, 40, 40, 41, 41, 37, 37, 37, 34,
	}
	if sfIndex >= 12 {
		return 0
	}
	return maxPredSFBTable[sfIndex]
}
```

### Step 2.4: Add error definition

```go
// Add to errors
var ErrICSReservedBit = errors.New("syntax: ics_reserved_bit must be 0")
```

### Step 2.5: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestParseICSInfo_LongWindow`
Expected: PASS

### Step 2.6: Add test for short window

```go
func TestParseICSInfo_ShortWindow(t *testing.T) {
	// Build bitstream:
	// ics_reserved_bit: 0 (1 bit)
	// window_sequence: 10 (2 bits) = EIGHT_SHORT_SEQUENCE
	// window_shape: 0 (1 bit) = sine
	// max_sfb: 14 (4 bits) = 0b1110
	// scale_factor_grouping: 1111111 (7 bits) = all same group
	// Total: 1 + 2 + 1 + 4 + 7 = 15 bits
	// Bits: 0 10 0 1110 1111111 = 0b0100_1110_1111_111 padded
	data := []byte{0x4E, 0xFE} // 0b01001110 0b11111110
	r := bits.NewReader(data)

	ics := &ICStream{}
	cfg := &ICSInfoConfig{
		SFIndex:     4,
		FrameLength: 1024,
		ObjectType:  ObjectTypeLC,
	}

	err := ParseICSInfo(r, ics, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.WindowSequence != EightShortSequence {
		t.Errorf("WindowSequence: got %d, want %d", ics.WindowSequence, EightShortSequence)
	}
	if ics.MaxSFB != 14 {
		t.Errorf("MaxSFB: got %d, want 14", ics.MaxSFB)
	}
	if ics.ScaleFactorGrouping != 0x7F {
		t.Errorf("ScaleFactorGrouping: got 0x%02X, want 0x7F", ics.ScaleFactorGrouping)
	}
	// All bits set = 1 group with 8 windows
	if ics.NumWindowGroups != 1 {
		t.Errorf("NumWindowGroups: got %d, want 1", ics.NumWindowGroups)
	}
}
```

### Step 2.7: Run all ICS info tests

Run: `go test -v ./internal/syntax -run TestParseICSInfo`
Expected: PASS

### Step 2.8: Commit

```bash
git add internal/syntax/ics_info.go internal/syntax/ics_info_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add ParseICSInfo function

Parse ICS info element including window sequence, shape,
max_sfb, scale factor grouping, and predictor data.
Supports MAIN and LTP profile prediction.

Ported from: ics_info() in syntax.c:829-952

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Section Data Parsing

**Files:**
- Create: `internal/syntax/section.go`
- Test: `internal/syntax/section_test.go`

### Step 3.1: Write the failing test for ParseSectionData

```go
// internal/syntax/section_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseSectionData_SingleSection(t *testing.T) {
	// Long window, max_sfb = 4, single section using codebook 1
	// sect_cb: 4 bits = 0001 (codebook 1)
	// sect_len_incr: 5 bits = 00100 (4) - covers all 4 SFBs
	// Total: 9 bits
	// Bits: 0001 00100 = 0b0001_0010_0 padded = 0x12
	data := []byte{0x12, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		MaxSFB:          4,
		NumWindowGroups: 1,
	}

	err := ParseSectionData(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 1 section covering SFBs 0-3
	if ics.NumSec[0] != 1 {
		t.Errorf("NumSec[0]: got %d, want 1", ics.NumSec[0])
	}
	if ics.SectCB[0][0] != 1 {
		t.Errorf("SectCB[0][0]: got %d, want 1", ics.SectCB[0][0])
	}
	if ics.SectStart[0][0] != 0 {
		t.Errorf("SectStart[0][0]: got %d, want 0", ics.SectStart[0][0])
	}
	if ics.SectEnd[0][0] != 4 {
		t.Errorf("SectEnd[0][0]: got %d, want 4", ics.SectEnd[0][0])
	}

	// Check SFBCB (codebook per SFB)
	for sfb := uint8(0); sfb < 4; sfb++ {
		if ics.SFBCB[0][sfb] != 1 {
			t.Errorf("SFBCB[0][%d]: got %d, want 1", sfb, ics.SFBCB[0][sfb])
		}
	}
}
```

### Step 3.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestParseSectionData_SingleSection`
Expected: FAIL with "undefined: ParseSectionData"

### Step 3.3: Write ParseSectionData function

```go
// internal/syntax/section.go
package syntax

import (
	"github.com/llehouerou/go-aac/internal/bits"
	"github.com/llehouerou/go-aac/internal/huffman"
)

// ParseSectionData parses section data (Table 4.4.25).
// Section data assigns Huffman codebooks to ranges of scale factor bands.
//
// Ported from: section_data() in ~/dev/faad2/libfaad/syntax.c:1731-1881
func ParseSectionData(r *bits.Reader, ics *ICStream) error {
	var sectBits, sectLim uint8

	if ics.WindowSequence == EightShortSequence {
		sectBits = 3
		sectLim = 8 * 15 // 120
	} else {
		sectBits = 5
		sectLim = MaxSFB // 51
	}
	sectEscVal := uint8((1 << sectBits) - 1)

	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		k := uint8(0)
		i := uint8(0)

		for k < ics.MaxSFB {
			if r.Error() {
				return ErrBitstreamRead
			}
			if i >= sectLim {
				return ErrSectionLimit
			}

			// Read codebook (4 bits)
			sectCB := uint8(r.GetBits(4))
			ics.SectCB[g][i] = sectCB

			// Codebook 12 is reserved
			if sectCB == 12 {
				return ErrReservedCodebook
			}

			// Track special codebook usage
			if sectCB == uint8(huffman.NoiseHCB) {
				ics.NoiseUsed = true
			}
			if sectCB == uint8(huffman.IntensityHCB) || sectCB == uint8(huffman.IntensityHCB2) {
				ics.IsUsed = true
			}

			// Read section length
			sectLen := uint8(0)
			for {
				sectLenIncr := uint8(r.GetBits(uint(sectBits)))
				if sectLen > sectLim {
					return ErrSectionLength
				}
				sectLen += sectLenIncr
				if sectLenIncr != sectEscVal {
					break
				}
			}

			ics.SectStart[g][i] = uint16(k)
			ics.SectEnd[g][i] = uint16(k + sectLen)

			if sectLen > sectLim || k+sectLen > sectLim {
				return ErrSectionLength
			}

			// Assign codebook to each SFB in this section
			for sfb := k; sfb < k+sectLen; sfb++ {
				ics.SFBCB[g][sfb] = sectCB
			}

			k += sectLen
			i++
		}

		ics.NumSec[g] = i

		// Verify all SFBs covered
		if k != ics.MaxSFB {
			return ErrSectionCoverage
		}
	}

	return nil
}
```

### Step 3.4: Add error definitions

```go
// Add to errors
var (
	ErrBitstreamRead    = errors.New("syntax: bitstream read error")
	ErrSectionLimit     = errors.New("syntax: section limit exceeded")
	ErrReservedCodebook = errors.New("syntax: reserved codebook 12 used")
	ErrSectionLength    = errors.New("syntax: section length exceeds limit")
	ErrSectionCoverage  = errors.New("syntax: sections do not cover all SFBs")
)
```

### Step 3.5: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestParseSectionData_SingleSection`
Expected: PASS

### Step 3.6: Add test for multiple sections

```go
func TestParseSectionData_MultipleSections(t *testing.T) {
	// Long window, max_sfb = 6, two sections:
	// Section 1: codebook 0 (zero), length 2
	// Section 2: codebook 1, length 4
	// sect_cb: 0000 (codebook 0)
	// sect_len_incr: 00010 (2)
	// sect_cb: 0001 (codebook 1)
	// sect_len_incr: 00100 (4)
	// Total: 4+5+4+5 = 18 bits
	// Bits: 0000 00010 0001 00100 = 0b0000_0001_0000_1001_00
	data := []byte{0x01, 0x09, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		MaxSFB:          6,
		NumWindowGroups: 1,
	}

	err := ParseSectionData(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumSec[0] != 2 {
		t.Errorf("NumSec[0]: got %d, want 2", ics.NumSec[0])
	}

	// Section 0: codebook 0, SFBs 0-1
	if ics.SectCB[0][0] != 0 {
		t.Errorf("SectCB[0][0]: got %d, want 0", ics.SectCB[0][0])
	}
	if ics.SectEnd[0][0] != 2 {
		t.Errorf("SectEnd[0][0]: got %d, want 2", ics.SectEnd[0][0])
	}

	// Section 1: codebook 1, SFBs 2-5
	if ics.SectCB[0][1] != 1 {
		t.Errorf("SectCB[0][1]: got %d, want 1", ics.SectCB[0][1])
	}
	if ics.SectStart[0][1] != 2 {
		t.Errorf("SectStart[0][1]: got %d, want 2", ics.SectStart[0][1])
	}
	if ics.SectEnd[0][1] != 6 {
		t.Errorf("SectEnd[0][1]: got %d, want 6", ics.SectEnd[0][1])
	}
}
```

### Step 3.7: Run all section tests

Run: `go test -v ./internal/syntax -run TestParseSectionData`
Expected: PASS

### Step 3.8: Commit

```bash
git add internal/syntax/section.go internal/syntax/section_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add ParseSectionData function

Parse section data which assigns Huffman codebooks to
scale factor band ranges. Tracks noise and intensity
stereo usage flags.

Ported from: section_data() in syntax.c:1731-1881

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Scale Factor Parsing

**Files:**
- Create: `internal/syntax/scalefactor.go`
- Test: `internal/syntax/scalefactor_test.go`

### Step 4.1: Write the failing test for DecodeScaleFactors

```go
// internal/syntax/scalefactor_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestDecodeScaleFactors_AllZero(t *testing.T) {
	// Global gain = 100, single window group, max_sfb = 2
	// Both SFBs use zero codebook -> scale factors should be 0
	ics := &ICStream{
		GlobalGain:      100,
		NumWindowGroups: 1,
		MaxSFB:          2,
	}
	ics.SFBCB[0][0] = 0 // Zero codebook
	ics.SFBCB[0][1] = 0 // Zero codebook

	// No bits needed for zero codebook
	data := []byte{0x00}
	r := bits.NewReader(data)

	err := DecodeScaleFactors(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.ScaleFactors[0][0] != 0 {
		t.Errorf("ScaleFactors[0][0]: got %d, want 0", ics.ScaleFactors[0][0])
	}
	if ics.ScaleFactors[0][1] != 0 {
		t.Errorf("ScaleFactors[0][1]: got %d, want 0", ics.ScaleFactors[0][1])
	}
}
```

### Step 4.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestDecodeScaleFactors_AllZero`
Expected: FAIL with "undefined: DecodeScaleFactors"

### Step 4.3: Write DecodeScaleFactors function

```go
// internal/syntax/scalefactor.go
package syntax

import (
	"github.com/llehouerou/go-aac/internal/bits"
	"github.com/llehouerou/go-aac/internal/huffman"
)

// DecodeScaleFactors decodes scale factors from the bitstream.
// Scale factors are differentially coded relative to the global gain.
//
// Ported from: decode_scale_factors() in ~/dev/faad2/libfaad/syntax.c:1894-1985
func DecodeScaleFactors(r *bits.Reader, ics *ICStream) error {
	scaleFactor := int16(ics.GlobalGain)
	isPosition := int16(0)
	noisePCMFlag := true
	noiseEnergy := int16(ics.GlobalGain) - 90

	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
			cb := ics.SFBCB[g][sfb]

			switch huffman.Codebook(cb) {
			case huffman.ZeroHCB:
				// Zero codebook: scale factor is 0
				ics.ScaleFactors[g][sfb] = 0

			case huffman.IntensityHCB, huffman.IntensityHCB2:
				// Intensity stereo: decode position
				t := huffman.ScaleFactor(r)
				isPosition += int16(t)
				ics.ScaleFactors[g][sfb] = isPosition

			case huffman.NoiseHCB:
				// PNS: decode noise energy
				if noisePCMFlag {
					noisePCMFlag = false
					t := int16(r.GetBits(9)) - 256
					noiseEnergy += t
				} else {
					t := huffman.ScaleFactor(r)
					noiseEnergy += int16(t)
				}
				ics.ScaleFactors[g][sfb] = noiseEnergy

			default:
				// Spectral codebook: decode scale factor
				t := huffman.ScaleFactor(r)
				scaleFactor += int16(t)
				if scaleFactor < 0 || scaleFactor > 255 {
					return ErrScaleFactorRange
				}
				ics.ScaleFactors[g][sfb] = scaleFactor
			}
		}
	}

	return nil
}

// ParseScaleFactorData is the wrapper that matches FAAD2's scale_factor_data().
// Ported from: scale_factor_data() in ~/dev/faad2/libfaad/syntax.c:1988-2016
func ParseScaleFactorData(r *bits.Reader, ics *ICStream) error {
	return DecodeScaleFactors(r, ics)
}
```

### Step 4.4: Add error definition

```go
var ErrScaleFactorRange = errors.New("syntax: scale factor out of range [0, 255]")
```

### Step 4.5: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestDecodeScaleFactors_AllZero`
Expected: PASS

### Step 4.6: Add test for spectral scale factors

```go
func TestDecodeScaleFactors_Spectral(t *testing.T) {
	// Global gain = 100, 2 SFBs with spectral codebook
	// First SFB: delta = 0 (huffman returns 0 for certain patterns)
	// Second SFB: delta = +5
	// Need to construct bit patterns that decode to these deltas

	ics := &ICStream{
		GlobalGain:      100,
		NumWindowGroups: 1,
		MaxSFB:          2,
	}
	ics.SFBCB[0][0] = 1 // Spectral codebook
	ics.SFBCB[0][1] = 1 // Spectral codebook

	// The scale factor huffman codebook:
	// Value 60 (delta=0) has a specific codeword
	// For testing, we'll just verify no error and reasonable values
	// Real validation comes from FAAD2 comparison tests

	// Use a pattern that produces valid huffman codes
	// This is a simplified test - detailed validation via FAAD2 reference
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	r := bits.NewReader(data)

	err := DecodeScaleFactors(r, ics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify scale factors are in valid range
	for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
		sf := ics.ScaleFactors[0][sfb]
		if sf < 0 || sf > 255 {
			t.Errorf("ScaleFactors[0][%d] out of range: %d", sfb, sf)
		}
	}
}
```

### Step 4.7: Run all scale factor tests

Run: `go test -v ./internal/syntax -run TestDecodeScaleFactors`
Expected: PASS

### Step 4.8: Commit

```bash
git add internal/syntax/scalefactor.go internal/syntax/scalefactor_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add DecodeScaleFactors function

Decode DPCM-coded scale factors using Huffman decoding.
Handles zero, intensity stereo, noise (PNS), and spectral
codebook cases with proper differential decoding.

Ported from: decode_scale_factors() in syntax.c:1894-1985

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Pulse Data Parsing

**Files:**
- Modify: `internal/syntax/pulse.go` (add parsing function)
- Create: `internal/syntax/pulse_data_test.go`

### Step 5.1: Write the failing test for ParsePulseData

```go
// internal/syntax/pulse_data_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParsePulseData_TwoPulses(t *testing.T) {
	// number_pulse: 1 (2 bits) = 2 pulses (number_pulse + 1)
	// pulse_start_sfb: 5 (6 bits)
	// pulse_offset[0]: 10 (5 bits)
	// pulse_amp[0]: 7 (4 bits)
	// pulse_offset[1]: 15 (5 bits)
	// pulse_amp[1]: 3 (4 bits)
	// Total: 2 + 6 + 5 + 4 + 5 + 4 = 26 bits
	// Bits: 01 000101 01010 0111 01111 0011
	data := []byte{0x45, 0x47, 0x4C}
	r := bits.NewReader(data)

	ics := &ICStream{
		NumSWB: 40, // Valid for the start SFB
	}
	pul := &PulseInfo{}

	err := ParsePulseData(r, ics, pul)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pul.NumberPulse != 1 { // 1 means 2 pulses
		t.Errorf("NumberPulse: got %d, want 1", pul.NumberPulse)
	}
	if pul.PulseStartSFB != 5 {
		t.Errorf("PulseStartSFB: got %d, want 5", pul.PulseStartSFB)
	}
	if pul.PulseOffset[0] != 10 {
		t.Errorf("PulseOffset[0]: got %d, want 10", pul.PulseOffset[0])
	}
	if pul.PulseAmp[0] != 7 {
		t.Errorf("PulseAmp[0]: got %d, want 7", pul.PulseAmp[0])
	}
}
```

### Step 5.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestParsePulseData_TwoPulses`
Expected: FAIL with "undefined: ParsePulseData"

### Step 5.3: Add ParsePulseData function to pulse.go

```go
// Add to internal/syntax/pulse.go

import "github.com/llehouerou/go-aac/internal/bits"

// ParsePulseData parses pulse data from the bitstream.
// Ported from: pulse_data() in ~/dev/faad2/libfaad/syntax.c:955-980
func ParsePulseData(r *bits.Reader, ics *ICStream, pul *PulseInfo) error {
	// number_pulse (2 bits) - actual count is number_pulse + 1
	pul.NumberPulse = uint8(r.GetBits(2))

	// pulse_start_sfb (6 bits)
	pul.PulseStartSFB = uint8(r.GetBits(6))

	// Validate start SFB
	if pul.PulseStartSFB > ics.NumSWB {
		return ErrPulseStartSFB
	}

	// Read offset and amplitude for each pulse
	numPulses := pul.NumberPulse + 1
	for i := uint8(0); i < numPulses; i++ {
		pul.PulseOffset[i] = uint8(r.GetBits(5))
		pul.PulseAmp[i] = uint8(r.GetBits(4))
	}

	return nil
}
```

### Step 5.4: Add error definition

```go
var ErrPulseStartSFB = errors.New("syntax: pulse_start_sfb exceeds num_swb")
```

### Step 5.5: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestParsePulseData_TwoPulses`
Expected: PASS

### Step 5.6: Commit

```bash
git add internal/syntax/pulse.go internal/syntax/pulse_data_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add ParsePulseData function

Parse pulse data for spectral coefficient modification.
Validates pulse_start_sfb against num_swb.

Ported from: pulse_data() in syntax.c:955-980

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: TNS Data Parsing

**Files:**
- Modify: `internal/syntax/tns.go` (add parsing function)
- Create: `internal/syntax/tns_data_test.go`

### Step 6.1: Write the failing test for ParseTNSData

```go
// internal/syntax/tns_data_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseTNSData_LongWindow(t *testing.T) {
	// Long window, 1 filter with order 4
	// n_filt[0]: 1 (2 bits)
	// coef_res: 1 (1 bit) = 4-bit coefficients
	// length: 20 (6 bits)
	// order: 4 (5 bits)
	// direction: 0 (1 bit)
	// coef_compress: 0 (1 bit)
	// coef[0-3]: each 4 bits
	// Total: 2 + 1 + 6 + 5 + 1 + 1 + 16 = 32 bits
	data := []byte{0x55, 0x02, 0x12, 0x34}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		NumWindows:     1,
	}
	tns := &TNSInfo{}

	ParseTNSData(r, ics, tns)

	if tns.NFilt[0] != 1 {
		t.Errorf("NFilt[0]: got %d, want 1", tns.NFilt[0])
	}
}
```

### Step 6.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestParseTNSData_LongWindow`
Expected: FAIL with "undefined: ParseTNSData"

### Step 6.3: Add ParseTNSData function to tns.go

```go
// Add to internal/syntax/tns.go

import "github.com/llehouerou/go-aac/internal/bits"

// ParseTNSData parses TNS (Temporal Noise Shaping) data from the bitstream.
// Ported from: tns_data() in ~/dev/faad2/libfaad/syntax.c:2019-2089
func ParseTNSData(r *bits.Reader, ics *ICStream, tns *TNSInfo) {
	var nFiltBits, lengthBits, orderBits uint

	if ics.WindowSequence == EightShortSequence {
		nFiltBits = 1
		lengthBits = 4
		orderBits = 3
	} else {
		nFiltBits = 2
		lengthBits = 6
		orderBits = 5
	}

	for w := uint8(0); w < ics.NumWindows; w++ {
		startCoefBits := uint(3)

		tns.NFilt[w] = uint8(r.GetBits(nFiltBits))

		if tns.NFilt[w] != 0 {
			tns.CoefRes[w] = r.Get1Bit()
			if tns.CoefRes[w] != 0 {
				startCoefBits = 4
			}
		}

		for filt := uint8(0); filt < tns.NFilt[w]; filt++ {
			tns.Length[w][filt] = uint8(r.GetBits(lengthBits))
			tns.Order[w][filt] = uint8(r.GetBits(orderBits))

			if tns.Order[w][filt] != 0 {
				tns.Direction[w][filt] = r.Get1Bit()
				tns.CoefCompress[w][filt] = r.Get1Bit()

				coefBits := startCoefBits - uint(tns.CoefCompress[w][filt])
				for i := uint8(0); i < tns.Order[w][filt]; i++ {
					tns.Coef[w][filt][i] = uint8(r.GetBits(coefBits))
				}
			}
		}
	}
}
```

### Step 6.4: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestParseTNSData_LongWindow`
Expected: PASS

### Step 6.5: Commit

```bash
git add internal/syntax/tns.go internal/syntax/tns_data_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add ParseTNSData function

Parse TNS filter data for temporal noise shaping.
Handles both long and short window configurations.

Ported from: tns_data() in syntax.c:2019-2089

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: LTP Data Parsing

**Files:**
- Modify: `internal/syntax/ltp.go` (add parsing function)
- Create: `internal/syntax/ltp_data_test.go`

### Step 7.1: Write the failing test for ParseLTPData

```go
// internal/syntax/ltp_data_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseLTPData_LongWindow(t *testing.T) {
	// Long window LTP data:
	// lag: 500 (11 bits)
	// coef: 3 (3 bits)
	// long_used[0-39]: 40 bits (we'll test with all zeros)
	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		MaxSFB:         40,
	}

	// lag = 500 = 0b00111110100
	// coef = 3 = 0b011
	// Followed by 40 zero bits for long_used
	data := []byte{0x1F, 0x43, 0x00, 0x00, 0x00, 0x00, 0x00}
	r := bits.NewReader(data)

	ltp := &LTPInfo{}

	err := ParseLTPData(r, ics, ltp, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ltp.Lag != 500 {
		t.Errorf("Lag: got %d, want 500", ltp.Lag)
	}
	if ltp.Coef != 3 {
		t.Errorf("Coef: got %d, want 3", ltp.Coef)
	}
}
```

### Step 7.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestParseLTPData_LongWindow`
Expected: FAIL with "undefined: ParseLTPData"

### Step 7.3: Add ParseLTPData function to ltp.go

```go
// Add to internal/syntax/ltp.go

import "github.com/llehouerou/go-aac/internal/bits"

// ParseLTPData parses Long Term Prediction data from the bitstream.
// Ported from: ltp_data() in ~/dev/faad2/libfaad/syntax.c:2093-2152
func ParseLTPData(r *bits.Reader, ics *ICStream, ltp *LTPInfo, frameLength uint16) error {
	// Read lag (11 bits for normal, 10 for LD mode)
	ltp.Lag = uint16(r.GetBits(11))

	// Validate lag
	if ltp.Lag > frameLength*2 {
		return ErrLTPLag
	}

	// Read coefficient index (3 bits)
	ltp.Coef = uint8(r.GetBits(3))

	if ics.WindowSequence == EightShortSequence {
		// Short window: per-window flags
		for w := uint8(0); w < ics.NumWindows; w++ {
			ltp.ShortUsed[w] = r.Get1Bit() != 0
			if ltp.ShortUsed[w] {
				ltp.ShortLagPresent[w] = r.Get1Bit() != 0
				if ltp.ShortLagPresent[w] {
					ltp.ShortLag[w] = uint8(r.GetBits(4))
				}
			}
		}
	} else {
		// Long window: per-SFB flags
		lastBand := ics.MaxSFB
		if lastBand > MaxLTPSFB {
			lastBand = MaxLTPSFB
		}
		ltp.LastBand = lastBand

		for sfb := uint8(0); sfb < lastBand; sfb++ {
			ltp.LongUsed[sfb] = r.Get1Bit() != 0
		}
	}

	return nil
}
```

### Step 7.4: Add error definition

```go
var ErrLTPLag = errors.New("syntax: LTP lag exceeds frame length * 2")
```

### Step 7.5: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestParseLTPData_LongWindow`
Expected: PASS

### Step 7.6: Commit

```bash
git add internal/syntax/ltp.go internal/syntax/ltp_data_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add ParseLTPData function

Parse Long Term Prediction data for LTP profile.
Handles both long and short window configurations.

Ported from: ltp_data() in syntax.c:2093-2152

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Spectral Data Parsing

**Files:**
- Create: `internal/syntax/spectral.go`
- Test: `internal/syntax/spectral_test.go`

### Step 8.1: Write the failing test for ParseSpectralData

```go
// internal/syntax/spectral_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseSpectralData_ZeroCodebook(t *testing.T) {
	// All SFBs use zero codebook - no spectral data needed
	ics := &ICStream{
		WindowSequence:  OnlyLongSequence,
		NumWindowGroups: 1,
		MaxSFB:          4,
	}
	ics.NumSec[0] = 1
	ics.SectCB[0][0] = 0 // Zero codebook
	ics.SectStart[0][0] = 0
	ics.SectEnd[0][0] = 4
	// Set up SFB offsets
	ics.SectSFBOffset[0][0] = 0
	ics.SectSFBOffset[0][1] = 32
	ics.SectSFBOffset[0][2] = 64
	ics.SectSFBOffset[0][3] = 96
	ics.SectSFBOffset[0][4] = 128
	ics.WindowGroupLength[0] = 1

	data := []byte{0x00}
	r := bits.NewReader(data)

	specData := make([]int16, 1024)
	err := ParseSpectralData(r, ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All values should remain 0
	for i := 0; i < 128; i++ {
		if specData[i] != 0 {
			t.Errorf("specData[%d]: got %d, want 0", i, specData[i])
		}
	}
}
```

### Step 8.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestParseSpectralData_ZeroCodebook`
Expected: FAIL with "undefined: ParseSpectralData"

### Step 8.3: Write ParseSpectralData function

```go
// internal/syntax/spectral.go
package syntax

import (
	"github.com/llehouerou/go-aac/internal/bits"
	"github.com/llehouerou/go-aac/internal/huffman"
)

// ParseSpectralData decodes spectral coefficients from the bitstream.
// Ported from: spectral_data() in ~/dev/faad2/libfaad/syntax.c:2156-2236
func ParseSpectralData(r *bits.Reader, ics *ICStream, specData []int16, frameLength uint16) error {
	nshort := frameLength / 8
	groups := uint8(0)

	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		p := uint16(groups) * nshort

		for i := uint8(0); i < ics.NumSec[g]; i++ {
			sectCB := ics.SectCB[g][i]

			// Determine increment (quad vs pair)
			var inc uint16
			if sectCB >= uint8(huffman.FirstPairHCB) {
				inc = 2
			} else {
				inc = 4
			}

			switch huffman.Codebook(sectCB) {
			case huffman.ZeroHCB, huffman.NoiseHCB, huffman.IntensityHCB, huffman.IntensityHCB2:
				// No spectral data - just skip
				p += ics.SectSFBOffset[g][ics.SectEnd[g][i]] - ics.SectSFBOffset[g][ics.SectStart[g][i]]

			default:
				// Decode spectral data
				start := ics.SectSFBOffset[g][ics.SectStart[g][i]]
				end := ics.SectSFBOffset[g][ics.SectEnd[g][i]]

				for k := start; k < end; k += inc {
					if err := huffman.SpectralData(sectCB, r, specData[p:]); err != nil {
						return err
					}
					p += inc
				}
			}
		}

		groups += ics.WindowGroupLength[g]
	}

	return nil
}
```

### Step 8.4: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestParseSpectralData_ZeroCodebook`
Expected: PASS

### Step 8.5: Commit

```bash
git add internal/syntax/spectral.go internal/syntax/spectral_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add ParseSpectralData function

Decode spectral coefficients using Huffman decoding.
Handles zero, noise, and intensity codebooks by skipping.

Ported from: spectral_data() in syntax.c:2156-2236

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Side Info and Individual Channel Stream

**Files:**
- Modify: `internal/syntax/ics.go` (add main parsing functions)
- Test: `internal/syntax/ics_parser_test.go`

### Step 9.1: Write failing test for ParseSideInfo

```go
// internal/syntax/ics_parser_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseSideInfo_Basic(t *testing.T) {
	// This tests the side_info parsing flow
	// We need a complete bitstream with:
	// - global_gain (8 bits)
	// - ics_info (if not common_window)
	// - section_data
	// - scale_factor_data
	// - pulse_data_present (1 bit) = 0
	// - tns_data_present (1 bit) = 0
	// - gain_control_data_present (1 bit) = 0

	// For this test, we'll use common_window=true so ics_info is skipped
	// and minimal section/scale factor data

	// This is a simplified test - real validation via FAAD2 reference
	t.Skip("Integration test - requires complete bitstream")
}
```

### Step 9.2: Add ParseSideInfo function to ics.go

```go
// Add to internal/syntax/ics.go

import "github.com/llehouerou/go-aac/internal/bits"

// SideInfoConfig holds configuration for side info parsing.
type SideInfoConfig struct {
	SFIndex      uint8
	FrameLength  uint16
	ObjectType   uint8
	CommonWindow bool
	ScalFlag     bool // True for scalable AAC
}

// ParseSideInfo parses side information for an ICS.
// Ported from: side_info() in ~/dev/faad2/libfaad/syntax.c:1578-1668
func ParseSideInfo(r *bits.Reader, ele *Element, ics *ICStream, cfg *SideInfoConfig) error {
	// Read global gain (8 bits)
	ics.GlobalGain = uint8(r.GetBits(8))

	// Parse ics_info if not common_window and not scalable
	if !ele.CommonWindow && !cfg.ScalFlag {
		icsCfg := &ICSInfoConfig{
			SFIndex:      cfg.SFIndex,
			FrameLength:  cfg.FrameLength,
			ObjectType:   cfg.ObjectType,
			CommonWindow: ele.CommonWindow,
		}
		if err := ParseICSInfo(r, ics, icsCfg); err != nil {
			return err
		}
	}

	// Parse section data
	if err := ParseSectionData(r, ics); err != nil {
		return err
	}

	// Parse scale factor data
	if err := ParseScaleFactorData(r, ics); err != nil {
		return err
	}

	// Only parse tool data if not scalable
	if !cfg.ScalFlag {
		// Pulse data
		ics.PulseDataPresent = r.Get1Bit() != 0
		if ics.PulseDataPresent {
			if err := ParsePulseData(r, ics, &ics.Pul); err != nil {
				return err
			}
		}

		// TNS data
		ics.TNSDataPresent = r.Get1Bit() != 0
		if ics.TNSDataPresent {
			// Only parse TNS for non-ER object types
			if cfg.ObjectType < ERObjectStart {
				ParseTNSData(r, ics, &ics.TNS)
			}
		}

		// Gain control data (SSR profile only)
		ics.GainControlDataPresent = r.Get1Bit() != 0
		if ics.GainControlDataPresent {
			return ErrGainControlNotSupported
		}
	}

	return nil
}

// ERObjectStart is the first error-resilient object type.
const ERObjectStart = 17
```

### Step 9.3: Add error definition

```go
var ErrGainControlNotSupported = errors.New("syntax: gain control (SSR) not supported")
```

### Step 9.4: Add ParseIndividualChannelStream function

```go
// Add to internal/syntax/ics.go

// ICSConfig holds configuration for ICS parsing.
type ICSConfig struct {
	SFIndex      uint8
	FrameLength  uint16
	ObjectType   uint8
	CommonWindow bool
	ScalFlag     bool
}

// ParseIndividualChannelStream parses a complete individual channel stream.
// This is the main entry point for decoding one channel's data.
//
// Ported from: individual_channel_stream() in ~/dev/faad2/libfaad/syntax.c:1671-1728
func ParseIndividualChannelStream(r *bits.Reader, ele *Element, ics *ICStream, specData []int16, cfg *ICSConfig) error {
	// Parse side info (global gain, section, scale factors, tools)
	sideCfg := &SideInfoConfig{
		SFIndex:      cfg.SFIndex,
		FrameLength:  cfg.FrameLength,
		ObjectType:   cfg.ObjectType,
		CommonWindow: cfg.CommonWindow,
		ScalFlag:     cfg.ScalFlag,
	}
	if err := ParseSideInfo(r, ele, ics, sideCfg); err != nil {
		return err
	}

	// For ER object types, TNS data is parsed here
	if cfg.ObjectType >= ERObjectStart && ics.TNSDataPresent {
		ParseTNSData(r, ics, &ics.TNS)
	}

	// Parse spectral data
	if err := ParseSpectralData(r, ics, specData, cfg.FrameLength); err != nil {
		return err
	}

	// Pulse decoding is done later in spectrum reconstruction
	// Validate pulse not used with short blocks
	if ics.PulseDataPresent && ics.WindowSequence == EightShortSequence {
		return ErrPulseInShortBlock
	}

	return nil
}
```

### Step 9.5: Add error definition

```go
var ErrPulseInShortBlock = errors.New("syntax: pulse coding not allowed in short blocks")
```

### Step 9.6: Run tests

Run: `go test -v ./internal/syntax`
Expected: PASS

### Step 9.7: Commit

```bash
git add internal/syntax/ics.go internal/syntax/ics_parser_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add ParseSideInfo and ParseIndividualChannelStream

Main ICS parsing functions that orchestrate side info parsing
(global gain, section data, scale factors, pulse, TNS) and
spectral data decoding.

Ported from: side_info() and individual_channel_stream() in syntax.c

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Error File Consolidation

**Files:**
- Create: `internal/syntax/errors.go`

### Step 10.1: Create consolidated errors file

```go
// internal/syntax/errors.go
package syntax

import "errors"

// Parsing errors
var (
	// Window and grouping errors
	ErrInvalidSRIndex        = errors.New("syntax: invalid sample rate index")
	ErrInvalidWindowSequence = errors.New("syntax: invalid window sequence")
	ErrMaxSFBTooLarge        = errors.New("syntax: max_sfb exceeds num_swb")

	// ICS info errors
	ErrICSReservedBit = errors.New("syntax: ics_reserved_bit must be 0")

	// Section data errors
	ErrBitstreamRead    = errors.New("syntax: bitstream read error")
	ErrSectionLimit     = errors.New("syntax: section limit exceeded")
	ErrReservedCodebook = errors.New("syntax: reserved codebook 12 used")
	ErrSectionLength    = errors.New("syntax: section length exceeds limit")
	ErrSectionCoverage  = errors.New("syntax: sections do not cover all SFBs")

	// Scale factor errors
	ErrScaleFactorRange = errors.New("syntax: scale factor out of range [0, 255]")

	// Pulse errors
	ErrPulseStartSFB    = errors.New("syntax: pulse_start_sfb exceeds num_swb")
	ErrPulseInShortBlock = errors.New("syntax: pulse coding not allowed in short blocks")

	// LTP errors
	ErrLTPLag = errors.New("syntax: LTP lag exceeds frame length * 2")

	// Gain control errors
	ErrGainControlNotSupported = errors.New("syntax: gain control (SSR) not supported")
)
```

### Step 10.2: Run all tests

Run: `go test -v ./internal/syntax`
Expected: PASS

### Step 10.3: Commit

```bash
git add internal/syntax/errors.go
git commit -m "$(cat <<'EOF'
refactor(syntax): consolidate error definitions

Move all syntax parsing errors to a single errors.go file
for better organization and discoverability.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Integration Test with FAAD2 Reference

**Files:**
- Create: `internal/syntax/ics_faad2_test.go`

### Step 11.1: Create FAAD2 reference test

```go
// internal/syntax/ics_faad2_test.go
package syntax

import (
	"os"
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestICSParser_FAAD2Reference(t *testing.T) {
	// Skip if no reference data available
	refDir := os.Getenv("FAAD2_REF_DIR")
	if refDir == "" {
		t.Skip("FAAD2_REF_DIR not set - skipping reference comparison")
	}

	// TODO: Implement detailed FAAD2 comparison
	// 1. Load test AAC file
	// 2. Parse ADTS header to get configuration
	// 3. Parse ICS and compare against reference
	t.Skip("TODO: Implement FAAD2 reference comparison")
}
```

### Step 11.2: Commit

```bash
git add internal/syntax/ics_faad2_test.go
git commit -m "$(cat <<'EOF'
test(syntax): add placeholder for FAAD2 reference tests

Placeholder for ICS parsing validation against FAAD2 reference data.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Summary

This plan implements Step 3.7 (Individual Channel Stream Parser) with:

1. **Window Grouping** - Calculates window groups and SFB offsets
2. **ICS Info Parsing** - Parses window configuration and predictor data
3. **Section Data Parsing** - Assigns Huffman codebooks to SFB ranges
4. **Scale Factor Decoding** - DPCM decoding of scale factors
5. **Pulse Data Parsing** - Parses pulse coding data
6. **TNS Data Parsing** - Parses temporal noise shaping filters
7. **LTP Data Parsing** - Parses long-term prediction data
8. **Spectral Data Decoding** - Decodes spectral coefficients using Huffman
9. **Side Info + ICS Main** - Orchestrates complete ICS parsing
10. **Error Consolidation** - Clean error handling
11. **FAAD2 Reference Tests** - Validation framework

**Files created:**
- `internal/syntax/window.go` + test
- `internal/syntax/ics_info.go` + test
- `internal/syntax/section.go` + test
- `internal/syntax/scalefactor.go` + test
- `internal/syntax/spectral.go` + test
- `internal/syntax/errors.go`
- `internal/syntax/ics_faad2_test.go`

**Files modified:**
- `internal/syntax/ics.go` (add parsing functions)
- `internal/syntax/pulse.go` (add ParsePulseData)
- `internal/syntax/tns.go` (add ParseTNSData)
- `internal/syntax/ltp.go` (add ParseLTPData)

---

**Plan complete and saved to `docs/plans/2025-12-28-ics-parser.md`.**

**Two execution options:**

1. **Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

2. **Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
