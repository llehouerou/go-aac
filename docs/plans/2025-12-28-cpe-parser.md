# Channel Pair Element (CPE) Parser Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement CPE parsing for stereo AAC decoding (Step 3.9 of MIGRATION_STEPS.md)

**Architecture:** CPE parsing follows the SCE pattern but handles two channels with optional shared window configuration (common_window) and M/S stereo mask. When common_window is set, both channels share the same ICS info and may use M/S stereo coding.

**Tech Stack:** Go, bits.Reader, existing syntax package types (Element, ICStream)

---

## Background

### FAAD2 Source Reference
- **File:** `~/dev/faad2/libfaad/syntax.c:698-826`
- **Function:** `channel_pair_element()`

### Key Differences from SCE
1. Two channels (uses both ICS1 and ICS2)
2. `common_window` flag - when set, both channels share ics_info
3. M/S stereo mask (`ms_mask_present`, `ms_used[][]`)
4. `ms_mask_present == 3` is a reserved/error value

### CPE Bitstream Structure (Table 4.4.5)
```
element_instance_tag          4 bits
common_window                 1 bit
if (common_window) {
    ics_info()                variable
    ms_mask_present           2 bits
    if (ms_mask_present == 1) {
        for (g = 0; g < num_window_groups; g++)
            for (sfb = 0; sfb < max_sfb; sfb++)
                ms_used[g][sfb]  1 bit
    }
}
individual_channel_stream()   variable (channel 1)
individual_channel_stream()   variable (channel 2)
```

---

### Task 1: Add ErrMSMaskReserved Error

**Files:**
- Modify: `internal/syntax/errors.go:76` (add after SCE/LFE errors section)

**Step 1: Write the failing test**

Create test in `internal/syntax/errors_test.go` (add to existing file or create):

```go
func TestErrMSMaskReserved(t *testing.T) {
	if ErrMSMaskReserved == nil {
		t.Error("ErrMSMaskReserved should not be nil")
	}

	expectedMsg := "syntax: ms_mask_present value 3 is reserved"
	if ErrMSMaskReserved.Error() != expectedMsg {
		t.Errorf("Error message = %q, want %q", ErrMSMaskReserved.Error(), expectedMsg)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: ErrMSMaskReserved"

**Step 3: Write minimal implementation**

Add to `internal/syntax/errors.go` after line 76 (after SCE/LFE errors section):

```go
// CPE errors.
var (
	// ErrMSMaskReserved indicates ms_mask_present has reserved value 3.
	// FAAD2 error code: 32
	ErrMSMaskReserved = errors.New("syntax: ms_mask_present value 3 is reserved")
)
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/errors.go internal/syntax/errors_test.go
git commit -m "feat(syntax): add ErrMSMaskReserved error for CPE parsing"
```

---

### Task 2: Create CPE Config and Result Types

**Files:**
- Create: `internal/syntax/cpe.go`
- Test: `internal/syntax/cpe_test.go`

**Step 1: Write the failing test**

Create `internal/syntax/cpe_test.go`:

```go
// internal/syntax/cpe_test.go
package syntax

import (
	"testing"
)

func TestCPEConfig_Fields(t *testing.T) {
	cfg := &CPEConfig{
		SFIndex:     4,
		FrameLength: 1024,
		ObjectType:  ObjectTypeLC,
	}

	if cfg.SFIndex != 4 {
		t.Errorf("SFIndex = %d, want 4", cfg.SFIndex)
	}
	if cfg.FrameLength != 1024 {
		t.Errorf("FrameLength = %d, want 1024", cfg.FrameLength)
	}
	if cfg.ObjectType != ObjectTypeLC {
		t.Errorf("ObjectType = %d, want %d", cfg.ObjectType, ObjectTypeLC)
	}
}

func TestCPEResult_Fields(t *testing.T) {
	result := &CPEResult{
		Tag:       5,
		SpecData1: make([]int16, 1024),
		SpecData2: make([]int16, 1024),
	}

	if result.Tag != 5 {
		t.Errorf("Tag = %d, want 5", result.Tag)
	}
	if len(result.SpecData1) != 1024 {
		t.Errorf("len(SpecData1) = %d, want 1024", len(result.SpecData1))
	}
	if len(result.SpecData2) != 1024 {
		t.Errorf("len(SpecData2) = %d, want 1024", len(result.SpecData2))
	}
}

func TestCPEResult_ElementInitialization(t *testing.T) {
	result := &CPEResult{}

	// Verify Element can hold two channels
	result.Element.Channel = 0
	result.Element.PairedChannel = 1
	result.Element.CommonWindow = true

	if result.Element.PairedChannel != 1 {
		t.Errorf("PairedChannel = %d, want 1", result.Element.PairedChannel)
	}
	if !result.Element.CommonWindow {
		t.Error("CommonWindow should be true")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: CPEConfig"

**Step 3: Write minimal implementation**

Create `internal/syntax/cpe.go`:

```go
// internal/syntax/cpe.go
package syntax

// CPEConfig holds configuration for Channel Pair Element parsing.
// Ported from: channel_pair_element() parameters in ~/dev/faad2/libfaad/syntax.c:698
type CPEConfig struct {
	SFIndex     uint8  // Sample rate index (0-11)
	FrameLength uint16 // Frame length (960 or 1024)
	ObjectType  uint8  // Audio object type
}

// CPEResult holds the result of parsing a Channel Pair Element.
// Ported from: channel_pair_element() return values in ~/dev/faad2/libfaad/syntax.c:698-826
type CPEResult struct {
	Element   Element // Parsed element data (contains ICS1 and ICS2)
	SpecData1 []int16 // Spectral coefficients for channel 1 (1024 or 960 values)
	SpecData2 []int16 // Spectral coefficients for channel 2 (1024 or 960 values)
	Tag       uint8   // Element instance tag (for channel mapping)
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/cpe.go internal/syntax/cpe_test.go
git commit -m "feat(syntax): add CPEConfig and CPEResult types"
```

---

### Task 3: Implement ParseChannelPairElement - Element Tag and Common Window

**Files:**
- Modify: `internal/syntax/cpe.go`
- Modify: `internal/syntax/cpe_test.go`

**Step 1: Write the failing test**

Add to `internal/syntax/cpe_test.go`:

```go
import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseChannelPairElement_ElementTag(t *testing.T) {
	// Test parsing element_instance_tag (4 bits) and common_window (1 bit)
	testCases := []struct {
		name         string
		tag          uint8
		commonWindow bool
	}{
		{"tag 0, no common window", 0, false},
		{"tag 7, common window", 7, true},
		{"tag 15, no common window", 15, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build bitstream: tag (4 bits) + common_window (1 bit)
			// For no common_window case, we need minimal ICS data for both channels
			// For common_window case, we need ics_info + ms_mask + ICS data

			// This is a basic structure test - full parsing tested separately
			if LenTag != 4 {
				t.Errorf("LenTag = %d, want 4", LenTag)
			}
		})
	}
}

func TestParseChannelPairElement_Signature(t *testing.T) {
	// Verify ParseChannelPairElement has the expected signature
	type parserFunc func(*bits.Reader, uint8, *CPEConfig) (*CPEResult, error)
	var _ parserFunc = ParseChannelPairElement
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: ParseChannelPairElement"

**Step 3: Write minimal implementation**

Add to `internal/syntax/cpe.go`:

```go
import "github.com/llehouerou/go-aac/internal/bits"

// ParseChannelPairElement parses a Channel Pair Element (CPE).
// CPE contains two audio channels that may share window configuration
// and use M/S (Mid/Side) stereo coding.
//
// This function:
// 1. Reads the element_instance_tag (4 bits)
// 2. Reads the common_window flag (1 bit)
// 3. If common_window, parses shared ics_info and M/S mask
// 4. Parses individual_channel_stream for both channels
//
// The spectral reconstruction (M/S decoding, inverse quantization, filter bank)
// is handled separately in Phase 4.
//
// Ported from: channel_pair_element() in ~/dev/faad2/libfaad/syntax.c:698-826
func ParseChannelPairElement(r *bits.Reader, channels uint8, cfg *CPEConfig) (*CPEResult, error) {
	result := &CPEResult{
		SpecData1: make([]int16, cfg.FrameLength),
		SpecData2: make([]int16, cfg.FrameLength),
	}

	// Initialize element
	// Ported from: syntax.c:709-710
	result.Element.Channel = channels
	result.Element.PairedChannel = int16(channels + 1)

	// Read element_instance_tag (4 bits)
	// Ported from: syntax.c:712-714
	result.Element.ElementInstanceTag = uint8(r.GetBits(LenTag))
	result.Tag = result.Element.ElementInstanceTag

	// Read common_window flag (1 bit)
	// Ported from: syntax.c:716-717
	result.Element.CommonWindow = r.Get1Bit() != 0

	if result.Element.CommonWindow {
		// Parse shared ics_info
		// Ported from: syntax.c:719-721
		icsCfg := &ICSInfoConfig{
			SFIndex:      cfg.SFIndex,
			FrameLength:  cfg.FrameLength,
			ObjectType:   cfg.ObjectType,
			CommonWindow: true,
		}
		if err := ParseICSInfo(r, &result.Element.ICS1, icsCfg); err != nil {
			return nil, err
		}

		// Parse M/S mask
		// Ported from: syntax.c:723-741
		if err := parseMSMask(r, &result.Element.ICS1); err != nil {
			return nil, err
		}

		// Copy ICS1 to ICS2 (they share window configuration)
		// Ported from: syntax.c:764
		result.Element.ICS2 = result.Element.ICS1
	} else {
		// No common window - M/S stereo not used
		// Ported from: syntax.c:765-767
		result.Element.ICS1.MSMaskPresent = 0
	}

	// Parse individual channel stream for channel 1
	// Ported from: syntax.c:769-773
	ics1Cfg := &ICSConfig{
		SFIndex:      cfg.SFIndex,
		FrameLength:  cfg.FrameLength,
		ObjectType:   cfg.ObjectType,
		CommonWindow: result.Element.CommonWindow,
		ScalFlag:     false,
	}
	if err := ParseIndividualChannelStream(r, &result.Element, &result.Element.ICS1, result.SpecData1, ics1Cfg); err != nil {
		return nil, err
	}

	// Parse individual channel stream for channel 2
	// Ported from: syntax.c:797-801
	ics2Cfg := &ICSConfig{
		SFIndex:      cfg.SFIndex,
		FrameLength:  cfg.FrameLength,
		ObjectType:   cfg.ObjectType,
		CommonWindow: result.Element.CommonWindow,
		ScalFlag:     false,
	}
	if err := ParseIndividualChannelStream(r, &result.Element, &result.Element.ICS2, result.SpecData2, ics2Cfg); err != nil {
		return nil, err
	}

	// Note: SBR fill element handling is done in Phase 8
	// Note: reconstruct_channel_pair is done in Phase 4

	return result, nil
}

// parseMSMask parses the M/S stereo mask from the bitstream.
// Ported from: channel_pair_element() ms_mask section in syntax.c:723-741
func parseMSMask(r *bits.Reader, ics *ICStream) error {
	// Read ms_mask_present (2 bits)
	ics.MSMaskPresent = uint8(r.GetBits(2))

	// Value 3 is reserved
	if ics.MSMaskPresent == 3 {
		return ErrMSMaskReserved
	}

	// If ms_mask_present == 1, read per-band mask
	if ics.MSMaskPresent == 1 {
		for g := uint8(0); g < ics.NumWindowGroups; g++ {
			for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
				ics.MSUsed[g][sfb] = r.Get1Bit()
			}
		}
	}
	// If ms_mask_present == 2, all bands use M/S (handled in spectrum reconstruction)
	// If ms_mask_present == 0, no M/S stereo

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/cpe.go internal/syntax/cpe_test.go
git commit -m "feat(syntax): implement ParseChannelPairElement function"
```

---

### Task 4: Add M/S Mask Parsing Tests

**Files:**
- Modify: `internal/syntax/cpe_test.go`

**Step 1: Write the failing test**

Add to `internal/syntax/cpe_test.go`:

```go
func TestParseMSMask_Reserved(t *testing.T) {
	// Test that ms_mask_present == 3 returns error
	// Build bitstream with ms_mask_present = 3 (binary: 11)
	data := []byte{0xC0} // 11 + padding
	r := bits.NewReader(data)

	ics := &ICStream{
		NumWindowGroups: 1,
		MaxSFB:          10,
	}

	err := parseMSMask(r, ics)
	if err != ErrMSMaskReserved {
		t.Errorf("Expected ErrMSMaskReserved, got %v", err)
	}
}

func TestParseMSMask_NoMS(t *testing.T) {
	// Test ms_mask_present == 0 (no M/S stereo)
	data := []byte{0x00} // 00 + padding
	r := bits.NewReader(data)

	ics := &ICStream{
		NumWindowGroups: 1,
		MaxSFB:          10,
	}

	err := parseMSMask(r, ics)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if ics.MSMaskPresent != 0 {
		t.Errorf("MSMaskPresent = %d, want 0", ics.MSMaskPresent)
	}
}

func TestParseMSMask_AllMS(t *testing.T) {
	// Test ms_mask_present == 2 (all bands use M/S)
	data := []byte{0x80} // 10 + padding
	r := bits.NewReader(data)

	ics := &ICStream{
		NumWindowGroups: 1,
		MaxSFB:          10,
	}

	err := parseMSMask(r, ics)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if ics.MSMaskPresent != 2 {
		t.Errorf("MSMaskPresent = %d, want 2", ics.MSMaskPresent)
	}
}

func TestParseMSMask_PerBand(t *testing.T) {
	// Test ms_mask_present == 1 (per-band mask)
	// With NumWindowGroups=1, MaxSFB=4: need 4 bits for mask
	// Bitstream: 01 (ms_mask=1) + 1010 (mask bits) + padding
	// = 0101_0100 = 0x54
	data := []byte{0x54}
	r := bits.NewReader(data)

	ics := &ICStream{
		NumWindowGroups: 1,
		MaxSFB:          4,
	}

	err := parseMSMask(r, ics)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if ics.MSMaskPresent != 1 {
		t.Errorf("MSMaskPresent = %d, want 1", ics.MSMaskPresent)
	}

	// Check mask bits: 1, 0, 1, 0
	expected := []uint8{1, 0, 1, 0}
	for i, exp := range expected {
		if ics.MSUsed[0][i] != exp {
			t.Errorf("MSUsed[0][%d] = %d, want %d", i, ics.MSUsed[0][i], exp)
		}
	}
}
```

**Step 2: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS (tests should pass with existing implementation)

**Step 3: Commit**

```bash
git add internal/syntax/cpe_test.go
git commit -m "test(syntax): add M/S mask parsing tests for CPE"
```

---

### Task 5: Add CPE Integration Tests

**Files:**
- Create: `internal/syntax/cpe_faad2_test.go`

**Step 1: Write placeholder test**

Create `internal/syntax/cpe_faad2_test.go`:

```go
// internal/syntax/cpe_faad2_test.go
package syntax

import "testing"

// TestParseCPE_FAAD2Reference tests CPE parsing against FAAD2 reference data.
// This test will be expanded once we have real stereo AAC test files and
// FAAD2 reference data generated by scripts/check_faad2.
//
// Test methodology:
// 1. Generate stereo AAC file: ffmpeg -f lavfi -i "sine=frequency=440:duration=1" \
//    -ac 2 -c:a aac -b:a 128k testdata/stereo.aac
// 2. Generate reference: ./scripts/check_faad2 testdata/stereo.aac
// 3. Compare parsed CPE fields against reference data
func TestParseCPE_FAAD2Reference(t *testing.T) {
	t.Skip("TODO: Add FAAD2 reference comparison once stereo test files are generated")

	// Placeholder for future FAAD2 reference tests:
	// - Compare element_instance_tag
	// - Compare common_window flag
	// - Compare ms_mask_present
	// - Compare parsed ICS1 and ICS2 fields
	// - Compare spectral data
}

// TestParseCPE_StereoFile tests CPE parsing with a real stereo AAC file.
func TestParseCPE_StereoFile(t *testing.T) {
	t.Skip("TODO: Add integration test with real stereo AAC file")

	// Placeholder for future integration tests:
	// 1. Read stereo AAC file
	// 2. Skip ADTS header
	// 3. Parse raw_data_block to find CPE
	// 4. Verify CPE parsing succeeds
	// 5. Verify spectral data is populated
}
```

**Step 2: Run tests**

Run: `make test PKG=./internal/syntax`
Expected: PASS (skipped tests)

**Step 3: Commit**

```bash
git add internal/syntax/cpe_faad2_test.go
git commit -m "test(syntax): add placeholder for CPE FAAD2 reference tests"
```

---

### Task 6: Verify Full Test Suite Passes

**Step 1: Run full test suite**

Run: `make check`
Expected: All tests pass, no lint errors

**Step 2: Verify CPE is ready for integration**

Checklist:
- [ ] `CPEConfig` and `CPEResult` types defined
- [ ] `ParseChannelPairElement` function implemented
- [ ] `parseMSMask` helper function implemented
- [ ] `ErrMSMaskReserved` error defined
- [ ] M/S mask parsing tests pass
- [ ] No regressions in existing tests

**Step 3: Final commit with all verification**

```bash
git status
# Verify all changes are committed
```

---

## Acceptance Criteria (from MIGRATION_STEPS.md)

- [x] Parse element instance tag (4 bits)
- [x] Parse common_window flag (1 bit)
- [x] Parse M/S mask if common_window
- [x] Parse both ICS (individual channel streams)
- [x] Handle stereo coupling (common_window + M/S mask)
- [x] Error on reserved ms_mask_present value (3)
- [ ] Unit tests pass
- [ ] Output matches FAAD2 reference for test files (deferred to FAAD2 integration)

---

## Notes

### Error Resilience (Deferred)
FAAD2 handles ER (Error Resilient) objects specially in CPE:
- For ER objects with LTP, second LTP data is parsed between channel 1 and channel 2
- This is handled in `syntax.c:775-794`
- Currently deferred as ER support is Phase 10

### SBR Handling (Deferred)
FAAD2 checks for a fill element after CPE for SBR data:
- This is handled in `syntax.c:803-816`
- Currently deferred as SBR support is Phase 8

### Spectral Reconstruction (Deferred)
`reconstruct_channel_pair()` is called after CPE parsing in FAAD2:
- Applies M/S decoding, intensity stereo, PNS correlation
- This is Phase 4 work (Step 4.12)
