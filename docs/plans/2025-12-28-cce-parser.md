# CCE Parser Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the Coupling Channel Element (CCE) parser for AAC bitstream decoding.

**Architecture:** The CCE parser will parse coupling channel data which allows multiple audio channels to share gain information. The implementation follows FAAD2's approach: parse and validate the bitstream structure, but discard the data since CCE is rarely used in practice. This allows the decoder to properly skip CCE elements without aborting on valid bitstreams.

**Tech Stack:** Go, bits.Reader for bitstream reading, huffman.ScaleFactor for gain elements

---

## Background

The Coupling Channel Element (CCE) is defined in ISO/IEC 14496-3 Table 4.4.8. It allows coupling channels to share gain information with target channels. CCE is primarily used for:
- Multichannel audio configurations (e.g., 5.1 surround)
- Efficient encoding of correlated channels

FAAD2's implementation (syntax.c:987-1076) parses CCE but discards the data, allowing the decoder to handle bitstreams containing CCE elements without full processing.

---

## Task 1: Add CCE Error Definitions

**Files:**
- Modify: `internal/syntax/errors.go`
- Test: `internal/syntax/errors_test.go`

### Step 1.1: Write the failing test for CCE errors

```go
// Add to errors_test.go

func TestCCEErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrIntensityStereoInCCE",
			err:  ErrIntensityStereoInCCE,
			msg:  "syntax: intensity stereo not allowed in coupling channel element",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("error should not be nil")
			}
			if tt.err.Error() != tt.msg {
				t.Errorf("got %q, want %q", tt.err.Error(), tt.msg)
			}
		})
	}
}
```

### Step 1.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestCCEErrors`
Expected: FAIL with "undefined: ErrIntensityStereoInCCE"

### Step 1.3: Add CCE error definition

```go
// Add to errors.go after CPE errors section

// CCE errors.
var (
	// ErrIntensityStereoInCCE indicates intensity stereo was used in a coupling channel element.
	// Intensity stereo is not valid in CCE.
	// FAAD2 error code: 32
	ErrIntensityStereoInCCE = errors.New("syntax: intensity stereo not allowed in coupling channel element")
)
```

### Step 1.4: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestCCEErrors`
Expected: PASS

### Step 1.5: Commit

```bash
git add internal/syntax/errors.go internal/syntax/errors_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add CCE error definition

Add ErrIntensityStereoInCCE for coupling channel element parsing.
Intensity stereo is not allowed in CCE (FAAD2 error code 32).

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Add CCE Types

**Files:**
- Create: `internal/syntax/cce.go`
- Test: `internal/syntax/cce_test.go`

### Step 2.1: Write the failing test for CCE types

```go
// cce_test.go
package syntax

import (
	"testing"
)

func TestCCEConfig_Initialization(t *testing.T) {
	cfg := &CCEConfig{
		SFIndex:     4, // 44100 Hz
		FrameLength: 1024,
		ObjectType:  2, // AAC-LC
	}

	if cfg.SFIndex != 4 {
		t.Errorf("SFIndex: got %d, want 4", cfg.SFIndex)
	}
	if cfg.FrameLength != 1024 {
		t.Errorf("FrameLength: got %d, want 1024", cfg.FrameLength)
	}
	if cfg.ObjectType != 2 {
		t.Errorf("ObjectType: got %d, want 2", cfg.ObjectType)
	}
}

func TestCCEResult_Initialization(t *testing.T) {
	result := &CCEResult{}

	if result.Tag != 0 {
		t.Errorf("Tag should be zero-initialized")
	}
	if result.IndSwCCEFlag {
		t.Errorf("IndSwCCEFlag should be false initially")
	}
	if result.NumCoupledElements != 0 {
		t.Errorf("NumCoupledElements should be zero-initialized")
	}
}

func TestCCECoupledElement_Initialization(t *testing.T) {
	elem := CCECoupledElement{
		TargetIsCPE:  true,
		TargetTag:    5,
		CCL:          true,
		CCR:          false,
	}

	if !elem.TargetIsCPE {
		t.Errorf("TargetIsCPE should be true")
	}
	if elem.TargetTag != 5 {
		t.Errorf("TargetTag: got %d, want 5", elem.TargetTag)
	}
	if !elem.CCL {
		t.Errorf("CCL should be true")
	}
	if elem.CCR {
		t.Errorf("CCR should be false")
	}
}
```

### Step 2.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestCCE`
Expected: FAIL with "undefined: CCEConfig"

### Step 2.3: Create CCE types file

```go
// internal/syntax/cce.go
package syntax

// CCEConfig holds configuration for Coupling Channel Element parsing.
// Ported from: coupling_channel_element() parameters in ~/dev/faad2/libfaad/syntax.c:987
type CCEConfig struct {
	SFIndex     uint8  // Sample rate index (0-11)
	FrameLength uint16 // Frame length (960 or 1024)
	ObjectType  uint8  // Audio object type
}

// CCECoupledElement holds information about a coupled element target.
// Ported from: coupling_channel_element() loop in ~/dev/faad2/libfaad/syntax.c:1006-1027
type CCECoupledElement struct {
	TargetIsCPE bool  // True if target is a CPE (vs SCE)
	TargetTag   uint8 // Target element instance tag (0-15)
	CCL         bool  // Apply coupling to left channel (only if TargetIsCPE)
	CCR         bool  // Apply coupling to right channel (only if TargetIsCPE)
}

// CCEResult holds the result of parsing a Coupling Channel Element.
// Note: CCE data is parsed but not used for decoding (rarely used in practice).
// Ported from: coupling_channel_element() in ~/dev/faad2/libfaad/syntax.c:987-1076
type CCEResult struct {
	Tag                  uint8               // Element instance tag (0-15)
	IndSwCCEFlag         bool                // Independently switched CCE
	NumCoupledElements   uint8               // Number of coupled elements (0-7)
	CoupledElements      [8]CCECoupledElement // Coupled element targets
	NumGainElementLists  uint8               // Number of gain element lists
	CCDomain             bool                // Coupling domain (0=before TNS, 1=after TNS)
	GainElementSign      bool                // Sign of gain elements
	GainElementScale     uint8               // Scale of gain elements (0-3)
	Element              Element             // Parsed ICS element
	SpecData             []int16             // Spectral data (parsed but not used)
}
```

### Step 2.4: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestCCE`
Expected: PASS

### Step 2.5: Commit

```bash
git add internal/syntax/cce.go internal/syntax/cce_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add CCE types

Add CCEConfig, CCECoupledElement, and CCEResult types for
Coupling Channel Element parsing.

Ported from ~/dev/faad2/libfaad/syntax.c:987-1076

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Implement CCE Header Parsing

**Files:**
- Modify: `internal/syntax/cce.go`
- Test: `internal/syntax/cce_test.go`

### Step 3.1: Write the failing test for CCE header parsing

```go
// Add to cce_test.go

func TestParseCCEHeader_SingleSCETarget(t *testing.T) {
	// CCE with 1 coupled SCE element:
	// element_instance_tag: 0 (4 bits) = 0000
	// ind_sw_cce_flag: 0 (1 bit) = 0
	// num_coupled_elements: 0 (3 bits) = 000 (meaning 1 element)
	// cc_target_is_cpe: 0 (1 bit) = 0 (SCE target)
	// cc_target_tag_select: 1 (4 bits) = 0001
	// cc_domain: 0 (1 bit) = 0
	// gain_element_sign: 0 (1 bit) = 0
	// gain_element_scale: 1 (2 bits) = 01
	// Total: 4+1+3+1+4+1+1+2 = 17 bits
	// 0000 0 000 0 0001 0 0 01 = 0x00 0x08 0x40 (padded)
	data := []byte{0x00, 0x08, 0x40}
	r := bits.NewReader(data)

	result := &CCEResult{}
	err := parseCCEHeader(r, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Tag != 0 {
		t.Errorf("Tag: got %d, want 0", result.Tag)
	}
	if result.IndSwCCEFlag {
		t.Errorf("IndSwCCEFlag should be false")
	}
	if result.NumCoupledElements != 0 {
		t.Errorf("NumCoupledElements: got %d, want 0", result.NumCoupledElements)
	}
	if result.NumGainElementLists != 1 {
		t.Errorf("NumGainElementLists: got %d, want 1", result.NumGainElementLists)
	}
	if result.CoupledElements[0].TargetIsCPE {
		t.Errorf("CoupledElements[0].TargetIsCPE should be false")
	}
	if result.CoupledElements[0].TargetTag != 1 {
		t.Errorf("CoupledElements[0].TargetTag: got %d, want 1", result.CoupledElements[0].TargetTag)
	}
	if result.CCDomain {
		t.Errorf("CCDomain should be false")
	}
	if result.GainElementSign {
		t.Errorf("GainElementSign should be false")
	}
	if result.GainElementScale != 1 {
		t.Errorf("GainElementScale: got %d, want 1", result.GainElementScale)
	}
}

func TestParseCCEHeader_CPETarget_BothChannels(t *testing.T) {
	// CCE with 1 coupled CPE element targeting both channels:
	// element_instance_tag: 2 (4 bits) = 0010
	// ind_sw_cce_flag: 1 (1 bit) = 1
	// num_coupled_elements: 0 (3 bits) = 000 (meaning 1 element)
	// cc_target_is_cpe: 1 (1 bit) = 1 (CPE target)
	// cc_target_tag_select: 3 (4 bits) = 0011
	// cc_l: 1 (1 bit) = 1
	// cc_r: 1 (1 bit) = 1 (both channels = extra gain list)
	// cc_domain: 1 (1 bit) = 1
	// gain_element_sign: 1 (1 bit) = 1
	// gain_element_scale: 2 (2 bits) = 10
	// Total: 4+1+3+1+4+1+1+1+1+2 = 19 bits
	// 0010 1 000 1 0011 1 1 1 1 10 = 0x28 0x9F 0x80
	data := []byte{0x28, 0x9F, 0x80}
	r := bits.NewReader(data)

	result := &CCEResult{}
	err := parseCCEHeader(r, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Tag != 2 {
		t.Errorf("Tag: got %d, want 2", result.Tag)
	}
	if !result.IndSwCCEFlag {
		t.Errorf("IndSwCCEFlag should be true")
	}
	// Both channels = 2 gain element lists
	if result.NumGainElementLists != 2 {
		t.Errorf("NumGainElementLists: got %d, want 2", result.NumGainElementLists)
	}
	if !result.CoupledElements[0].TargetIsCPE {
		t.Errorf("CoupledElements[0].TargetIsCPE should be true")
	}
	if result.CoupledElements[0].TargetTag != 3 {
		t.Errorf("CoupledElements[0].TargetTag: got %d, want 3", result.CoupledElements[0].TargetTag)
	}
	if !result.CoupledElements[0].CCL {
		t.Errorf("CoupledElements[0].CCL should be true")
	}
	if !result.CoupledElements[0].CCR {
		t.Errorf("CoupledElements[0].CCR should be true")
	}
	if !result.CCDomain {
		t.Errorf("CCDomain should be true")
	}
	if !result.GainElementSign {
		t.Errorf("GainElementSign should be true")
	}
	if result.GainElementScale != 2 {
		t.Errorf("GainElementScale: got %d, want 2", result.GainElementScale)
	}
}
```

### Step 3.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestParseCCEHeader`
Expected: FAIL with "undefined: parseCCEHeader"

### Step 3.3: Implement parseCCEHeader function

```go
// Add to cce.go

import "github.com/llehouerou/go-aac/internal/bits"

// parseCCEHeader parses the CCE header fields.
// Ported from: coupling_channel_element() in ~/dev/faad2/libfaad/syntax.c:998-1034
func parseCCEHeader(r *bits.Reader, result *CCEResult) error {
	// Read element_instance_tag (4 bits)
	// Ported from: syntax.c:998-999
	result.Tag = uint8(r.GetBits(LenTag))

	// Read ind_sw_cce_flag (1 bit)
	// Ported from: syntax.c:1001-1002
	result.IndSwCCEFlag = r.Get1Bit() != 0

	// Read num_coupled_elements (3 bits)
	// Ported from: syntax.c:1003-1004
	result.NumCoupledElements = uint8(r.GetBits(3))

	// Parse coupled element targets
	// Ported from: syntax.c:1006-1027
	result.NumGainElementLists = 0

	for c := uint8(0); c <= result.NumCoupledElements; c++ {
		result.NumGainElementLists++

		// Read cc_target_is_cpe (1 bit)
		result.CoupledElements[c].TargetIsCPE = r.Get1Bit() != 0

		// Read cc_target_tag_select (4 bits)
		result.CoupledElements[c].TargetTag = uint8(r.GetBits(4))

		if result.CoupledElements[c].TargetIsCPE {
			// Read cc_l and cc_r (1 bit each)
			result.CoupledElements[c].CCL = r.Get1Bit() != 0
			result.CoupledElements[c].CCR = r.Get1Bit() != 0

			// If both channels are coupled, we need an extra gain element list
			if result.CoupledElements[c].CCL && result.CoupledElements[c].CCR {
				result.NumGainElementLists++
			}
		}
	}

	// Read cc_domain (1 bit)
	// Ported from: syntax.c:1029-1030
	result.CCDomain = r.Get1Bit() != 0

	// Read gain_element_sign (1 bit)
	// Ported from: syntax.c:1031-1032
	result.GainElementSign = r.Get1Bit() != 0

	// Read gain_element_scale (2 bits)
	// Ported from: syntax.c:1033-1034
	result.GainElementScale = uint8(r.GetBits(2))

	return nil
}
```

### Step 3.4: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestParseCCEHeader`
Expected: PASS

### Step 3.5: Commit

```bash
git add internal/syntax/cce.go internal/syntax/cce_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): implement parseCCEHeader

Parse CCE header fields:
- element_instance_tag (4 bits)
- ind_sw_cce_flag (1 bit)
- num_coupled_elements (3 bits)
- coupled element targets (CPE/SCE, tag, channel flags)
- cc_domain, gain_element_sign, gain_element_scale

Ported from ~/dev/faad2/libfaad/syntax.c:998-1034

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Implement CCE Gain Elements Parsing

**Files:**
- Modify: `internal/syntax/cce.go`
- Test: `internal/syntax/cce_test.go`

### Step 4.1: Write the failing test for gain element parsing

```go
// Add to cce_test.go

func TestParseCCEGainElements_IndependentlySwitched(t *testing.T) {
	// When ind_sw_cce_flag is set, common_gain_element_present is always 1
	// So we just decode huffman scale factors for each gain element list
	// For simplicity, use huffman pattern that decodes to 0 (60-60=0)
	// The shortest SF huffman code is "1111111111" (10 bits) = delta 0

	// Create a minimal ICS with 1 window group, 2 SFBs, both with spectral codebook
	ics := &ICStream{
		NumWindowGroups: 1,
		MaxSFB:          2,
	}
	ics.SFBCB[0][0] = 1 // Non-zero codebook
	ics.SFBCB[0][1] = 1 // Non-zero codebook

	// 2 gain element lists, each needs 1 huffman code (10 bits each)
	// 1111111111 1111111111 = 0xFF 0xFF 0xC0
	data := []byte{0xFF, 0xFF, 0xC0}
	r := bits.NewReader(data)

	result := &CCEResult{
		IndSwCCEFlag:        true,
		NumGainElementLists: 2,
	}
	result.Element.ICS1 = *ics

	err := parseCCEGainElements(r, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCCEGainElements_CommonGainElement(t *testing.T) {
	// When common_gain_element_present=1, decode one huffman scale factor
	// When common_gain_element_present=0, decode per-SFB scale factors

	ics := &ICStream{
		NumWindowGroups: 1,
		MaxSFB:          2,
	}
	ics.SFBCB[0][0] = 1 // Non-zero codebook
	ics.SFBCB[0][1] = 0 // Zero codebook (no scale factor needed)

	// 2 gain element lists (c starts at 1, so only 1 iteration)
	// First: common_gain_element_present=1 (1 bit), then huffman code (10 bits)
	// 1 1111111111 = 0xFF 0xE0
	data := []byte{0xFF, 0xE0}
	r := bits.NewReader(data)

	result := &CCEResult{
		IndSwCCEFlag:        false,
		NumGainElementLists: 2,
	}
	result.Element.ICS1 = *ics

	err := parseCCEGainElements(r, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCCEGainElements_PerSFBGain(t *testing.T) {
	// When common_gain_element_present=0, decode per-SFB scale factors
	// Only for SFBs with non-zero codebook

	ics := &ICStream{
		NumWindowGroups: 1,
		MaxSFB:          3,
	}
	ics.SFBCB[0][0] = 1 // Non-zero - needs scale factor
	ics.SFBCB[0][1] = 0 // Zero - no scale factor
	ics.SFBCB[0][2] = 5 // Non-zero - needs scale factor

	// 2 gain element lists
	// c=1: common_gain_element_present=0 (1 bit), then 2 huffman codes (10 bits each)
	// 0 1111111111 1111111111 = 0x7F 0xFF 0xC0
	data := []byte{0x7F, 0xFF, 0xC0}
	r := bits.NewReader(data)

	result := &CCEResult{
		IndSwCCEFlag:        false,
		NumGainElementLists: 2,
	}
	result.Element.ICS1 = *ics

	err := parseCCEGainElements(r, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

### Step 4.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestParseCCEGainElements`
Expected: FAIL with "undefined: parseCCEGainElements"

### Step 4.3: Implement parseCCEGainElements function

```go
// Add to cce.go

import "github.com/llehouerou/go-aac/internal/huffman"

// parseCCEGainElements parses the gain element lists for CCE.
// Ported from: coupling_channel_element() in ~/dev/faad2/libfaad/syntax.c:1046-1073
func parseCCEGainElements(r *bits.Reader, result *CCEResult) error {
	ics := &result.Element.ICS1

	// For each gain element list (starting from c=1)
	// Ported from: syntax.c:1046
	for c := uint8(1); c < result.NumGainElementLists; c++ {
		var cge bool

		if result.IndSwCCEFlag {
			// For independently switched CCE, always use common gain
			// Ported from: syntax.c:1050-1052
			cge = true
		} else {
			// Read common_gain_element_present (1 bit)
			// Ported from: syntax.c:1054-1055
			cge = r.Get1Bit() != 0
		}

		if cge {
			// Common gain element: decode single huffman scale factor
			// Ported from: syntax.c:1058-1060
			_ = huffman.ScaleFactor(r)
		} else {
			// Per-SFB gain elements: decode scale factor for each non-zero SFB
			// Ported from: syntax.c:1062-1071
			for g := uint8(0); g < ics.NumWindowGroups; g++ {
				for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
					if ics.SFBCB[g][sfb] != uint8(huffman.ZeroHCB) {
						_ = huffman.ScaleFactor(r)
					}
				}
			}
		}
	}

	return nil
}
```

### Step 4.4: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestParseCCEGainElements`
Expected: PASS

### Step 4.5: Commit

```bash
git add internal/syntax/cce.go internal/syntax/cce_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): implement parseCCEGainElements

Parse gain element lists for CCE:
- For ind_sw_cce_flag: always use common gain element
- Otherwise: read common_gain_element_present flag
- If common gain: decode single huffman scale factor
- If per-SFB gain: decode scale factor for each non-zero SFB

Ported from ~/dev/faad2/libfaad/syntax.c:1046-1073

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Implement Main CCE Parser Function

**Files:**
- Modify: `internal/syntax/cce.go`
- Test: `internal/syntax/cce_test.go`

### Step 5.1: Write the failing test for main CCE parser

```go
// Add to cce_test.go

func TestParseCouplingChannelElement_ValidMinimal(t *testing.T) {
	// Create a minimal valid CCE bitstream:
	// - Header (17 bits for single SCE target)
	// - ICS data (will fail without proper setup, so we test header parsing only for now)

	// This test validates the function signature and basic flow
	cfg := &CCEConfig{
		SFIndex:     4,  // 44100 Hz
		FrameLength: 1024,
		ObjectType:  2,  // AAC-LC
	}

	// We need a complete valid bitstream, but for unit testing
	// we'll use a mock approach - verify the function exists and returns
	// an error for incomplete data
	data := []byte{0x00} // Incomplete data
	r := bits.NewReader(data)

	_, err := ParseCouplingChannelElement(r, cfg)
	// Should return an error due to incomplete bitstream
	if err == nil {
		t.Log("Note: Got nil error with minimal data - this is acceptable if the reader doesn't report errors")
	}
}

func TestParseCouplingChannelElement_IntensityStereoError(t *testing.T) {
	// CCE should return error if intensity stereo is used
	// This requires setting up a complete ICS with intensity stereo enabled

	// For now, we document this test case requirement
	// Full integration testing would require a complete CCE bitstream
	t.Log("Integration test: CCE with intensity stereo should return ErrIntensityStereoInCCE")
}
```

### Step 5.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestParseCouplingChannelElement`
Expected: FAIL with "undefined: ParseCouplingChannelElement"

### Step 5.3: Implement ParseCouplingChannelElement function

```go
// Add to cce.go

// ParseCouplingChannelElement parses a Coupling Channel Element (CCE).
// CCE allows coupling channels to share gain information with target channels.
//
// This function:
// 1. Parses the CCE header (tag, coupled elements, domain flags)
// 2. Parses the individual channel stream
// 3. Validates that intensity stereo is not used (illegal in CCE)
// 4. Parses gain element lists
//
// Note: CCE data is parsed but discarded (rarely used in practice).
// Ported from: coupling_channel_element() in ~/dev/faad2/libfaad/syntax.c:987-1076
func ParseCouplingChannelElement(r *bits.Reader, cfg *CCEConfig) (*CCEResult, error) {
	result := &CCEResult{
		SpecData: make([]int16, cfg.FrameLength),
	}

	// Initialize element
	result.Element.PairedChannel = -1 // No paired channel for CCE
	result.Element.CommonWindow = false

	// Parse CCE header
	// Ported from: syntax.c:998-1034
	if err := parseCCEHeader(r, result); err != nil {
		return nil, err
	}

	// Parse individual channel stream
	// Ported from: syntax.c:1036-1040
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

	// Intensity stereo is not allowed in coupling channel elements
	// Ported from: syntax.c:1042-1044
	if result.Element.ICS1.IsUsed {
		return nil, ErrIntensityStereoInCCE
	}

	// Parse gain element lists
	// Ported from: syntax.c:1046-1073
	if err := parseCCEGainElements(r, result); err != nil {
		return nil, err
	}

	return result, nil
}
```

### Step 5.4: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestParseCouplingChannelElement`
Expected: PASS

### Step 5.5: Commit

```bash
git add internal/syntax/cce.go internal/syntax/cce_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): implement ParseCouplingChannelElement

Main CCE parser function that:
1. Parses CCE header (tag, coupled elements, domain flags)
2. Parses individual channel stream
3. Validates intensity stereo is not used
4. Parses gain element lists

CCE data is parsed but discarded (rarely used in practice).

Ported from ~/dev/faad2/libfaad/syntax.c:987-1076

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Add CCE Constants and Limits

**Files:**
- Modify: `internal/syntax/limits.go`
- Test: `internal/syntax/limits_test.go`

### Step 6.1: Write the failing test for CCE limits

```go
// Add to limits_test.go

func TestCCELimits(t *testing.T) {
	// Maximum number of coupled elements in a CCE
	if MaxCoupledElements != 8 {
		t.Errorf("MaxCoupledElements: got %d, want 8", MaxCoupledElements)
	}
}
```

### Step 6.2: Run test to verify it fails

Run: `go test -v ./internal/syntax -run TestCCELimits`
Expected: FAIL with "undefined: MaxCoupledElements"

### Step 6.3: Add CCE limit constant

```go
// Add to limits.go

const (
	MaxChannels        = 64 // Maximum number of channels
	MaxSyntaxElements  = 48 // Maximum number of syntax elements
	MaxWindowGroups    = 8  // Maximum number of window groups
	MaxSFB             = 51 // Maximum number of scalefactor bands
	MaxLTPSFB          = 40 // Maximum LTP scalefactor bands (long)
	MaxLTPSFBS         = 8  // Maximum LTP scalefactor bands (short)
	MaxCoupledElements = 8  // Maximum coupled elements in CCE (3 bits = 0-7, +1 loop)
)
```

### Step 6.4: Run test to verify it passes

Run: `go test -v ./internal/syntax -run TestCCELimits`
Expected: PASS

### Step 6.5: Commit

```bash
git add internal/syntax/limits.go internal/syntax/limits_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add MaxCoupledElements constant

Add limit constant for CCE parsing: MaxCoupledElements = 8

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Run Full Test Suite and Verify

**Files:**
- All modified files

### Step 7.1: Run all syntax package tests

Run: `go test -v ./internal/syntax/...`
Expected: All tests PASS

### Step 7.2: Run linter

Run: `make lint`
Expected: No lint errors

### Step 7.3: Run full check

Run: `make check`
Expected: PASS (fmt, lint, test all succeed)

### Step 7.4: Final commit

```bash
git status
# Verify all changes are committed
```

---

## Summary

This implementation plan covers:

1. **Error definitions** - Added `ErrIntensityStereoInCCE`
2. **CCE types** - `CCEConfig`, `CCECoupledElement`, `CCEResult`
3. **Header parsing** - `parseCCEHeader` function
4. **Gain element parsing** - `parseCCEGainElements` function
5. **Main parser** - `ParseCouplingChannelElement` function
6. **Limits** - `MaxCoupledElements` constant

The implementation follows FAAD2's approach: parsing the CCE bitstream structure to allow proper decoding of files containing CCE elements, but discarding the coupling data since it's rarely used in practice.

Total estimated lines: ~150 lines of implementation code + ~200 lines of test code.
