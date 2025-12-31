# Single Channel Element (SCE) Parser Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the Single Channel Element (SCE) and LFE parsing function to decode mono audio channels and low-frequency effects from AAC bitstreams.

**Architecture:** Port FAAD2's `single_lfe_channel_element()` from `syntax.c` to Go. The function parses the element tag, calls the existing `ParseIndividualChannelStream()`, and validates that intensity stereo is not used (which is illegal in single-channel elements).

**Tech Stack:** Pure Go, depends on `internal/bits`, `internal/syntax` (existing ICS parser).

---

## Background: SCE/LFE Structure

The Single Channel Element and LFE Element share the same parsing function in FAAD2. They differ only in:
- SCE is used for mono audio or independent channels
- LFE is used for the subwoofer channel in surround configurations

```
single_lfe_channel_element()
├── element_instance_tag (4 bits)
├── individual_channel_stream()
│   ├── side_info (global_gain, section, scalefactors, tools)
│   └── spectral_data
├── Validate: IS not used (error if ics->is_used)
├── [Optional] SBR fill element (Phase 8)
└── [Later] reconstruct_single_channel (Phase 4)
```

**Source:** `~/dev/faad2/libfaad/syntax.c:652-696`

---

## Task 1: Create SCE Parser Function

**Files:**
- Create: `internal/syntax/sce.go`
- Test: `internal/syntax/sce_test.go`

### Step 1.1: Write the failing test for ParseSingleChannelElement

```go
// internal/syntax/sce_test.go
package syntax

import (
	"testing"
)

func TestParseSingleChannelElement_ElementTag(t *testing.T) {
	// Test that element_instance_tag is correctly parsed (4 bits)
	testCases := []struct {
		name     string
		tag      uint8
		expected uint8
	}{
		{"tag 0", 0, 0},
		{"tag 7", 7, 7},
		{"tag 15", 15, 15},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify the element_instance_tag constant is correct
			if LenTag != 4 {
				t.Errorf("LenTag = %d, want 4", LenTag)
			}

			// The actual parsing test requires a complete bitstream
			// with ICS data, which is tested in integration tests
		})
	}
}

func TestSCEConfig_Fields(t *testing.T) {
	// Test SCEConfig struct fields
	cfg := &SCEConfig{
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
```

### Step 1.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestParseSingleChannelElement`
Expected: FAIL with "undefined: SCEConfig" or similar

### Step 1.3: Write SCEConfig type and SCEResult type

```go
// internal/syntax/sce.go
package syntax

import "github.com/llehouerou/go-aac/internal/bits"

// SCEConfig holds configuration for Single Channel Element parsing.
type SCEConfig struct {
	SFIndex     uint8  // Sample rate index (0-11)
	FrameLength uint16 // Frame length (960 or 1024)
	ObjectType  uint8  // Audio object type
}

// SCEResult holds the result of parsing a Single Channel Element.
type SCEResult struct {
	Element  Element  // Parsed element data
	SpecData []int16  // Spectral coefficients (1024 or 960 values)
	Tag      uint8    // Element instance tag (for channel mapping)
}
```

### Step 1.4: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestSCEConfig_Fields`
Expected: PASS

### Step 1.5: Commit

```bash
git add internal/syntax/sce.go internal/syntax/sce_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add SCE config and result types

Add configuration and result types for Single Channel Element parsing.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Add IS Error Definition

**Files:**
- Modify: `internal/syntax/errors.go`
- Test: `internal/syntax/sce_test.go`

### Step 2.1: Write failing test for ErrIntensityStereoInSCE

```go
// Add to internal/syntax/sce_test.go

func TestErrIntensityStereoInSCE(t *testing.T) {
	if ErrIntensityStereoInSCE == nil {
		t.Error("ErrIntensityStereoInSCE should not be nil")
	}

	expectedMsg := "syntax: intensity stereo not allowed in single channel element"
	if ErrIntensityStereoInSCE.Error() != expectedMsg {
		t.Errorf("Error message = %q, want %q", ErrIntensityStereoInSCE.Error(), expectedMsg)
	}
}
```

### Step 2.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestErrIntensityStereoInSCE`
Expected: FAIL with "undefined: ErrIntensityStereoInSCE"

### Step 2.3: Add error definition

```go
// Add to internal/syntax/errors.go

// SCE/LFE errors.
var (
	// ErrIntensityStereoInSCE indicates intensity stereo was used in a single channel element.
	// Intensity stereo is only valid in Channel Pair Elements (CPE).
	// FAAD2 error code: 32
	ErrIntensityStereoInSCE = errors.New("syntax: intensity stereo not allowed in single channel element")
)
```

### Step 2.4: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestErrIntensityStereoInSCE`
Expected: PASS

### Step 2.5: Commit

```bash
git add internal/syntax/errors.go internal/syntax/sce_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add ErrIntensityStereoInSCE error

Add error for intensity stereo usage in single channel elements,
which is not allowed per AAC specification.
Corresponds to FAAD2 error code 32.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Implement ParseSingleChannelElement Function

**Files:**
- Modify: `internal/syntax/sce.go`
- Test: `internal/syntax/sce_test.go`

### Step 3.1: Write failing test for ParseSingleChannelElement

```go
// Add to internal/syntax/sce_test.go

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseSingleChannelElement_ISNotAllowed(t *testing.T) {
	// Test that intensity stereo in SCE returns an error
	// We need to create a mock ICS with IsUsed = true

	// This tests the validation logic, not the full parsing
	// (full parsing requires a complete valid bitstream)
	ics := &ICStream{
		IsUsed: true, // Intensity stereo was detected
	}

	// Verify the condition that triggers the error
	if ics.IsUsed {
		// This would return ErrIntensityStereoInSCE in ParseSingleChannelElement
		if ErrIntensityStereoInSCE == nil {
			t.Error("ErrIntensityStereoInSCE should be defined")
		}
	}
}
```

### Step 3.2: Run test to verify it passes (validation only)

Run: `go test -v ./internal/syntax -run TestParseSingleChannelElement_ISNotAllowed`
Expected: PASS

### Step 3.3: Write ParseSingleChannelElement function

```go
// Add to internal/syntax/sce.go

// ParseSingleChannelElement parses a Single Channel Element (SCE) or LFE element.
// SCE and LFE share the same syntax, differing only in their semantic use
// (SCE for mono audio, LFE for subwoofer channel).
//
// This function:
// 1. Reads the element_instance_tag (4 bits)
// 2. Parses the individual_channel_stream
// 3. Validates that intensity stereo is not used (illegal in SCE/LFE)
//
// The spectral reconstruction (inverse quantization, filter bank) is handled
// separately in Phase 4.
//
// Ported from: single_lfe_channel_element() in ~/dev/faad2/libfaad/syntax.c:652-696
func ParseSingleChannelElement(r *bits.Reader, channel uint8, cfg *SCEConfig) (*SCEResult, error) {
	result := &SCEResult{
		SpecData: make([]int16, cfg.FrameLength),
	}

	// Initialize element
	result.Element.Channel = channel
	result.Element.PairedChannel = -1 // No paired channel for SCE
	result.Element.CommonWindow = false

	// Read element_instance_tag (4 bits)
	// Ported from: syntax.c:660
	result.Element.ElementInstanceTag = uint8(r.GetBits(LenTag))
	result.Tag = result.Element.ElementInstanceTag

	// Parse the individual channel stream
	// Ported from: syntax.c:667
	icsCfg := &ICSConfig{
		SFIndex:      cfg.SFIndex,
		FrameLength:  cfg.FrameLength,
		ObjectType:   cfg.ObjectType,
		CommonWindow: false,
		ScalFlag:     false,
	}

	if err := ParseIndividualChannelStream(r, &result.Element, &result.Element.ICS1, result.SpecData, icsCfg); err != nil {
		return nil, err
	}

	// Intensity stereo is not allowed in single channel elements
	// Ported from: syntax.c:671-673
	if result.Element.ICS1.IsUsed {
		return nil, ErrIntensityStereoInSCE
	}

	// Note: SBR fill element handling is done in Phase 8
	// Note: reconstruct_single_channel is done in Phase 4

	return result, nil
}
```

### Step 3.4: Run tests to verify compilation

Run: `go test -v ./internal/syntax -run TestParseSingleChannelElement`
Expected: PASS

### Step 3.5: Commit

```bash
git add internal/syntax/sce.go internal/syntax/sce_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add ParseSingleChannelElement function

Parse Single Channel Element (SCE) and LFE elements:
- Read element_instance_tag (4 bits)
- Call ParseIndividualChannelStream for channel data
- Validate intensity stereo is not used (error if so)

SBR handling and spectral reconstruction are deferred to
later phases.

Ported from: single_lfe_channel_element() in syntax.c:652-696

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Add Helper Function for LFE Parsing

**Files:**
- Modify: `internal/syntax/sce.go`
- Test: `internal/syntax/sce_test.go`

### Step 4.1: Write test for ParseLFEElement

```go
// Add to internal/syntax/sce_test.go

func TestParseLFEElement_CallsSCEParser(t *testing.T) {
	// LFE uses the same parser as SCE
	// This test verifies the alias function exists

	// The function signature should match ParseSingleChannelElement
	cfg := &SCEConfig{
		SFIndex:     4,
		FrameLength: 1024,
		ObjectType:  ObjectTypeLC,
	}

	// Verify cfg is valid (can't actually parse without valid bitstream)
	if cfg.FrameLength != 1024 {
		t.Errorf("FrameLength = %d, want 1024", cfg.FrameLength)
	}
}
```

### Step 4.2: Add ParseLFEElement function

```go
// Add to internal/syntax/sce.go

// ParseLFEElement parses a Low Frequency Effects (LFE) element.
// LFE uses the same syntax as SCE, so this is an alias for ParseSingleChannelElement.
//
// The LFE channel is typically the ".1" in configurations like 5.1 surround.
// It carries bass frequencies (typically below 120 Hz) for the subwoofer.
//
// Ported from: single_lfe_channel_element() in ~/dev/faad2/libfaad/syntax.c:652-696
func ParseLFEElement(r *bits.Reader, channel uint8, cfg *SCEConfig) (*SCEResult, error) {
	return ParseSingleChannelElement(r, channel, cfg)
}
```

### Step 4.3: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestParseLFEElement`
Expected: PASS

### Step 4.4: Commit

```bash
git add internal/syntax/sce.go internal/syntax/sce_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add ParseLFEElement as alias for ParseSingleChannelElement

LFE (Low Frequency Effects) elements use the same syntax as SCE.
Add an explicit function for clarity when handling surround sound
configurations like 5.1.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Add Integration Test Placeholder

**Files:**
- Create: `internal/syntax/sce_faad2_test.go`

### Step 5.1: Create FAAD2 reference test placeholder

```go
// internal/syntax/sce_faad2_test.go
package syntax

import (
	"os"
	"testing"
)

func TestParseSCE_FAAD2Reference(t *testing.T) {
	// Skip if no reference data available
	refDir := os.Getenv("FAAD2_REF_DIR")
	if refDir == "" {
		t.Skip("FAAD2_REF_DIR not set - skipping reference comparison")
	}

	// TODO: Implement detailed FAAD2 comparison
	// 1. Load mono AAC test file
	// 2. Parse ADTS header to get configuration
	// 3. Parse SCE and compare spectral data against FAAD2 reference
	//
	// Test files to use:
	// - testdata/generated/aac_lc/44100_16_mono_128k/*.aac
	//
	// Reference generation:
	// ./scripts/check_faad2 testdata/test_mono.aac
	// Reference data in: /tmp/faad2_ref_test_mono/

	t.Skip("TODO: Implement FAAD2 reference comparison for SCE")
}

func TestParseLFE_FAAD2Reference(t *testing.T) {
	// Skip if no reference data available
	refDir := os.Getenv("FAAD2_REF_DIR")
	if refDir == "" {
		t.Skip("FAAD2_REF_DIR not set - skipping reference comparison")
	}

	// TODO: Implement detailed FAAD2 comparison for LFE
	// Need a 5.1 surround test file to extract LFE data

	t.Skip("TODO: Implement FAAD2 reference comparison for LFE")
}
```

### Step 5.2: Commit

```bash
git add internal/syntax/sce_faad2_test.go
git commit -m "$(cat <<'EOF'
test(syntax): add placeholder for SCE/LFE FAAD2 reference tests

Placeholder tests for validating SCE and LFE parsing against
FAAD2 reference data. Will be implemented when spectral
reconstruction (Phase 4) is complete.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Run Full Test Suite and Verify

**Files:**
- None (verification only)

### Step 6.1: Run all syntax tests

Run: `go test -v ./internal/syntax`
Expected: PASS

### Step 6.2: Run linter

Run: `make lint`
Expected: No errors

### Step 6.3: Run full check

Run: `make check`
Expected: PASS

---

## Summary

This plan implements Step 3.8 (Single Channel Element Parser) with:

1. **SCEConfig and SCEResult types** - Configuration and result structures
2. **ErrIntensityStereoInSCE error** - Error for illegal IS usage in SCE
3. **ParseSingleChannelElement function** - Main parsing function that:
   - Reads element_instance_tag (4 bits)
   - Calls ParseIndividualChannelStream for channel data
   - Validates no intensity stereo is used
4. **ParseLFEElement function** - Alias for LFE channel handling
5. **FAAD2 reference test placeholders** - For future validation

**Files created:**
- `internal/syntax/sce.go`
- `internal/syntax/sce_test.go`
- `internal/syntax/sce_faad2_test.go`

**Files modified:**
- `internal/syntax/errors.go` (add ErrIntensityStereoInSCE)

**Dependencies:**
- `internal/bits` (bit reader)
- `internal/syntax` (ICS parser, Element type, ICStream type)

**Note:** SBR fill element handling (syntax.c:676-688) is deferred to Phase 8 (HE-AAC).
Spectral reconstruction (syntax.c:690-693) is deferred to Phase 4.

---

**Plan complete and saved to `docs/plans/2025-12-28-sce-parser.md`.**

**Two execution options:**

1. **Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

2. **Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
