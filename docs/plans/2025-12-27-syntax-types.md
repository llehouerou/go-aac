# Phase 3.1: Core Data Structures Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Port all core AAC data structures from FAAD2's structs.h to Go, enabling syntax parsing in subsequent steps.

**Architecture:** Organize structures into multiple files by responsibility: `types.go` for ICStream and Element (the two core structures used throughout decoding), `pulse.go`, `tns.go`, `adts.go`, `adif.go`, `pce.go`, `drc.go` for specialized structures, and `ltp.go` for optional LTP profile support.

**Tech Stack:** Pure Go, no dependencies beyond stdlib. Uses existing `internal/syntax/constants.go` and `internal/syntax/limits.go`.

---

## Context

**Source:** `~/dev/faad2/libfaad/structs.h` (446 lines)

**Current State:**
- `internal/syntax/constants.go` - ElementID, WindowSequence, etc.
- `internal/syntax/limits.go` - MaxChannels, MaxSFB, etc.
- `internal/bits/reader.go` - Bit reader (complete)
- `internal/huffman/` - All codebooks and decoder (complete)

**Target Files to Create:**
| File | FAAD2 Source Lines | Go Lines (est.) |
|------|-------------------|-----------------|
| `internal/syntax/pulse.go` | 210-216 | ~30 |
| `internal/syntax/tns.go` | 218-227 | ~40 |
| `internal/syntax/ics.go` | 240-301 | ~100 |
| `internal/syntax/element.go` | 303-313 | ~30 |
| `internal/syntax/pce.go` | 103-144 | ~60 |
| `internal/syntax/adts.go` | 146-168 | ~50 |
| `internal/syntax/adif.go` | 170-183 | ~40 |
| `internal/syntax/drc.go` | 85-101 | ~40 |
| `internal/syntax/ltp.go` | 186-197 | ~30 |

---

## Task 1: PulseInfo Structure

**Files:**
- Create: `internal/syntax/pulse.go`
- Test: `internal/syntax/pulse_test.go`

**Step 1.1: Write the test for PulseInfo structure**

Create test file that validates the structure exists with correct field types and sizes.

```go
// internal/syntax/pulse_test.go
package syntax

import (
	"testing"
	"unsafe"
)

func TestPulseInfo_FieldTypes(t *testing.T) {
	var p PulseInfo

	// Verify field existence and types by assignment
	p.NumberPulse = 0
	p.PulseStartSFB = 0
	p.PulseOffset[0] = 0
	p.PulseAmp[0] = 0

	// Verify array sizes
	if len(p.PulseOffset) != 4 {
		t.Errorf("PulseOffset should have 4 elements, got %d", len(p.PulseOffset))
	}
	if len(p.PulseAmp) != 4 {
		t.Errorf("PulseAmp should have 4 elements, got %d", len(p.PulseAmp))
	}
}

func TestPulseInfo_ZeroValue(t *testing.T) {
	var p PulseInfo

	// Zero value should indicate no pulse data
	if p.NumberPulse != 0 {
		t.Errorf("Zero PulseInfo should have NumberPulse=0")
	}
}
```

**Step 1.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: PulseInfo"

**Step 1.3: Write PulseInfo implementation**

```go
// internal/syntax/pulse.go
package syntax

// PulseInfo contains pulse data for spectral coefficient modification.
// Up to 4 pulses can be added to the spectral data in long blocks only.
//
// Ported from: pulse_info in ~/dev/faad2/libfaad/structs.h:210-216
type PulseInfo struct {
	NumberPulse   uint8    // Number of pulses (0-4)
	PulseStartSFB uint8    // Starting scale factor band
	PulseOffset   [4]uint8 // Offset from start of SFB for each pulse
	PulseAmp      [4]uint8 // Amplitude of each pulse
}
```

**Step 1.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 1.5: Commit**

```bash
git add internal/syntax/pulse.go internal/syntax/pulse_test.go
git commit -m "feat(syntax): add PulseInfo structure

Ported from FAAD2 structs.h:210-216. Used for pulse data in spectral
coefficient modification (long blocks only)."
```

---

## Task 2: TNSInfo Structure

**Files:**
- Create: `internal/syntax/tns.go`
- Test: `internal/syntax/tns_test.go`

**Step 2.1: Write the test for TNSInfo structure**

```go
// internal/syntax/tns_test.go
package syntax

import "testing"

func TestTNSInfo_FieldTypes(t *testing.T) {
	var tns TNSInfo

	// Verify field existence - 8 window groups max
	tns.NFilt[0] = 0
	tns.CoefRes[0] = 0
	tns.Length[0][0] = 0
	tns.Order[0][0] = 0
	tns.Direction[0][0] = 0
	tns.CoefCompress[0][0] = 0
	tns.Coef[0][0][0] = 0

	// Verify dimensions
	if len(tns.NFilt) != MaxWindowGroups {
		t.Errorf("NFilt should have %d elements", MaxWindowGroups)
	}
	if len(tns.Length) != MaxWindowGroups || len(tns.Length[0]) != 4 {
		t.Errorf("Length should be [%d][4]", MaxWindowGroups)
	}
	if len(tns.Coef) != MaxWindowGroups || len(tns.Coef[0]) != 4 || len(tns.Coef[0][0]) != 32 {
		t.Errorf("Coef should be [%d][4][32]", MaxWindowGroups)
	}
}

func TestTNSInfo_MaxFilters(t *testing.T) {
	// TNS allows up to 4 filters per window
	var tns TNSInfo
	for w := 0; w < MaxWindowGroups; w++ {
		for f := 0; f < 4; f++ {
			tns.Order[w][f] = 1
		}
	}
}
```

**Step 2.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: TNSInfo"

**Step 2.3: Write TNSInfo implementation**

```go
// internal/syntax/tns.go
package syntax

// TNSInfo contains Temporal Noise Shaping filter data.
// TNS applies an all-pole filter to shape the quantization noise.
// Up to 4 filters can be applied per window group.
//
// Ported from: tns_info in ~/dev/faad2/libfaad/structs.h:218-227
type TNSInfo struct {
	NFilt        [MaxWindowGroups]uint8       // Number of filters per window group (0-4)
	CoefRes      [MaxWindowGroups]uint8       // Coefficient resolution (3 or 4 bits)
	Length       [MaxWindowGroups][4]uint8    // Filter length (region) per filter
	Order        [MaxWindowGroups][4]uint8    // Filter order (0-20 for long, 0-7 for short)
	Direction    [MaxWindowGroups][4]uint8    // Filter direction (0=upward, 1=downward)
	CoefCompress [MaxWindowGroups][4]uint8    // Coefficient compression flag
	Coef         [MaxWindowGroups][4][32]uint8 // Filter coefficients (up to 32 per filter)
}
```

**Step 2.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 2.5: Commit**

```bash
git add internal/syntax/tns.go internal/syntax/tns_test.go
git commit -m "feat(syntax): add TNSInfo structure

Ported from FAAD2 structs.h:218-227. Contains Temporal Noise Shaping
filter parameters for spectral shaping."
```

---

## Task 3: ICStream Structure (Main)

**Files:**
- Create: `internal/syntax/ics.go`
- Test: `internal/syntax/ics_test.go`

**Step 3.1: Write the test for ICStream structure**

```go
// internal/syntax/ics_test.go
package syntax

import "testing"

func TestICStream_CoreFields(t *testing.T) {
	var ics ICStream

	// Core fields
	ics.MaxSFB = 0
	ics.GlobalGain = 0
	ics.NumSWB = 0
	ics.NumWindowGroups = 0
	ics.NumWindows = 0
	ics.WindowSequence = OnlyLongSequence
	ics.WindowShape = 0
	ics.ScaleFactorGrouping = 0
}

func TestICStream_WindowGroupArrays(t *testing.T) {
	var ics ICStream

	// Window group length array - 8 groups max
	if len(ics.WindowGroupLength) != MaxWindowGroups {
		t.Errorf("WindowGroupLength should have %d elements", MaxWindowGroups)
	}

	// SFB offset arrays
	if len(ics.SectSFBOffset) != MaxWindowGroups {
		t.Errorf("SectSFBOffset should have %d window groups", MaxWindowGroups)
	}
	if len(ics.SectSFBOffset[0]) != 15*8 {
		t.Errorf("SectSFBOffset[n] should have 120 elements, got %d", len(ics.SectSFBOffset[0]))
	}

	if len(ics.SWBOffset) != 52 {
		t.Errorf("SWBOffset should have 52 elements, got %d", len(ics.SWBOffset))
	}
}

func TestICStream_SectionData(t *testing.T) {
	var ics ICStream

	// Section data arrays
	if len(ics.SectCB) != MaxWindowGroups || len(ics.SectCB[0]) != 15*8 {
		t.Errorf("SectCB dimensions wrong")
	}
	if len(ics.SectStart) != MaxWindowGroups || len(ics.SectStart[0]) != 15*8 {
		t.Errorf("SectStart dimensions wrong")
	}
	if len(ics.SectEnd) != MaxWindowGroups || len(ics.SectEnd[0]) != 15*8 {
		t.Errorf("SectEnd dimensions wrong")
	}
	if len(ics.SFBCB) != MaxWindowGroups || len(ics.SFBCB[0]) != 8*15 {
		t.Errorf("SFBCB dimensions wrong")
	}
	if len(ics.NumSec) != MaxWindowGroups {
		t.Errorf("NumSec should have %d elements", MaxWindowGroups)
	}
}

func TestICStream_ScaleFactors(t *testing.T) {
	var ics ICStream

	// Scale factors array - [window_groups][sfb]
	if len(ics.ScaleFactors) != MaxWindowGroups {
		t.Errorf("ScaleFactors should have %d window groups", MaxWindowGroups)
	}
	if len(ics.ScaleFactors[0]) != MaxSFB {
		t.Errorf("ScaleFactors[n] should have %d elements", MaxSFB)
	}
}

func TestICStream_MSInfo(t *testing.T) {
	var ics ICStream

	// M/S stereo data
	ics.MSMaskPresent = 0
	if len(ics.MSUsed) != MaxWindowGroups || len(ics.MSUsed[0]) != MaxSFB {
		t.Errorf("MSUsed dimensions should be [%d][%d]", MaxWindowGroups, MaxSFB)
	}
}

func TestICStream_EmbeddedStructs(t *testing.T) {
	var ics ICStream

	// Verify embedded structs exist
	_ = ics.Pul.NumberPulse
	_ = ics.TNS.NFilt[0]
}

func TestICStream_Flags(t *testing.T) {
	var ics ICStream

	// Parsing flags
	ics.NoiseUsed = false
	ics.IsUsed = false
	ics.PulseDataPresent = false
	ics.TNSDataPresent = false
	ics.GainControlDataPresent = false
	ics.PredictorDataPresent = false
}
```

**Step 3.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: ICStream"

**Step 3.3: Write ICStream implementation**

```go
// internal/syntax/ics.go
package syntax

// ICStream represents an Individual Channel Stream.
// This is the core data structure for a single audio channel,
// containing window info, section data, scale factors, and tool flags.
//
// Ported from: ic_stream in ~/dev/faad2/libfaad/structs.h:240-301
type ICStream struct {
	// Window configuration
	MaxSFB              uint8                          // Maximum scale factor band used
	GlobalGain          uint8                          // Global gain value (0-255)
	NumSWB              uint8                          // Number of scale factor bands
	NumWindowGroups     uint8                          // Number of window groups (1 for long, 1-8 for short)
	NumWindows          uint8                          // Number of windows (1 for long, 8 for short)
	WindowSequence      WindowSequence                 // Window sequence type
	WindowGroupLength   [MaxWindowGroups]uint8         // Number of windows per group
	WindowShape         uint8                          // Window shape (0=sine, 1=KBD)
	ScaleFactorGrouping uint8                          // Scale factor band grouping pattern

	// Scale factor band offsets (calculated from tables)
	SectSFBOffset [MaxWindowGroups][15 * 8]uint16 // Section SFB offsets per group
	SWBOffset     [52]uint16                      // SFB offsets for this frame
	SWBOffsetMax  uint16                          // Maximum SFB offset

	// Section data (Huffman codebook assignment)
	SectCB    [MaxWindowGroups][15 * 8]uint8  // Codebook index per section
	SectStart [MaxWindowGroups][15 * 8]uint16 // Section start SFB
	SectEnd   [MaxWindowGroups][15 * 8]uint16 // Section end SFB
	SFBCB     [MaxWindowGroups][8 * 15]uint8  // Codebook per SFB (derived)
	NumSec    [MaxWindowGroups]uint8          // Number of sections per group

	// Scale factors
	ScaleFactors [MaxWindowGroups][MaxSFB]int16 // Scale factors (0-255, noise/intensity special)

	// M/S stereo info (only used for CPE)
	MSMaskPresent uint8                           // 0=none, 1=per-band, 2=all
	MSUsed        [MaxWindowGroups][MaxSFB]uint8  // M/S mask per SFB

	// Tool usage flags
	NoiseUsed              bool // True if noise (PNS) bands present
	IsUsed                 bool // True if intensity stereo bands present
	PulseDataPresent       bool // True if pulse data follows
	TNSDataPresent         bool // True if TNS data follows
	GainControlDataPresent bool // True if gain control (SSR) data follows
	PredictorDataPresent   bool // True if predictor (MAIN/LTP) data follows

	// Embedded tool data
	Pul PulseInfo // Pulse data (if present)
	TNS TNSInfo   // TNS data (if present)
}
```

**Step 3.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 3.5: Commit**

```bash
git add internal/syntax/ics.go internal/syntax/ics_test.go
git commit -m "feat(syntax): add ICStream structure

Ported from FAAD2 structs.h:240-301. Individual Channel Stream contains
all per-channel data: window info, sections, scale factors, and tool flags."
```

---

## Task 4: Element Structure

**Files:**
- Create: `internal/syntax/element.go`
- Test: `internal/syntax/element_test.go`

**Step 4.1: Write the test for Element structure**

```go
// internal/syntax/element_test.go
package syntax

import "testing"

func TestElement_Fields(t *testing.T) {
	var e Element

	// Core fields
	e.Channel = 0
	e.PairedChannel = -1 // -1 indicates no pair
	e.ElementInstanceTag = 0
	e.CommonWindow = false
}

func TestElement_ICStreams(t *testing.T) {
	var e Element

	// Should have two ICStream fields for CPE
	e.ICS1.GlobalGain = 100
	e.ICS2.GlobalGain = 100

	if e.ICS1.GlobalGain != 100 || e.ICS2.GlobalGain != 100 {
		t.Error("ICStream fields not accessible")
	}
}

func TestElement_SCEUsage(t *testing.T) {
	// For SCE (Single Channel Element), only ICS1 is used
	var e Element
	e.Channel = 0
	e.PairedChannel = -1
	e.CommonWindow = false
	e.ICS1.WindowSequence = OnlyLongSequence
}

func TestElement_CPEUsage(t *testing.T) {
	// For CPE (Channel Pair Element), both ICS1 and ICS2 are used
	var e Element
	e.Channel = 0
	e.PairedChannel = 1
	e.CommonWindow = true
	e.ICS1.WindowSequence = OnlyLongSequence
	e.ICS2.WindowSequence = OnlyLongSequence
}
```

**Step 4.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: Element"

**Step 4.3: Write Element implementation**

```go
// internal/syntax/element.go
package syntax

// Element represents a syntax element (SCE, CPE, or LFE).
// SCE (Single Channel Element) uses only ICS1.
// CPE (Channel Pair Element) uses both ICS1 and ICS2.
// LFE (Low Frequency Effects) uses only ICS1.
//
// Ported from: element in ~/dev/faad2/libfaad/structs.h:303-313
type Element struct {
	Channel            uint8    // Output channel index
	PairedChannel      int16    // Paired channel for CPE (-1 if none)
	ElementInstanceTag uint8    // Element instance tag (0-15)
	CommonWindow       bool     // True if CPE shares window info

	ICS1 ICStream // First (or only) channel stream
	ICS2 ICStream // Second channel stream (CPE only)
}
```

**Step 4.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 4.5: Commit**

```bash
git add internal/syntax/element.go internal/syntax/element_test.go
git commit -m "feat(syntax): add Element structure

Ported from FAAD2 structs.h:303-313. Represents SCE/CPE/LFE syntax
elements containing one or two Individual Channel Streams."
```

---

## Task 5: ProgramConfig Structure

**Files:**
- Create: `internal/syntax/pce.go`
- Test: `internal/syntax/pce_test.go`

**Step 5.1: Write the test for ProgramConfig structure**

```go
// internal/syntax/pce_test.go
package syntax

import "testing"

func TestProgramConfig_CoreFields(t *testing.T) {
	var pce ProgramConfig

	pce.ElementInstanceTag = 0
	pce.ObjectType = 0
	pce.SFIndex = 0
	pce.Channels = 0
}

func TestProgramConfig_ChannelElements(t *testing.T) {
	var pce ProgramConfig

	// Element counts
	pce.NumFrontChannelElements = 0
	pce.NumSideChannelElements = 0
	pce.NumBackChannelElements = 0
	pce.NumLFEChannelElements = 0
	pce.NumAssocDataElements = 0
	pce.NumValidCCElements = 0

	// Element arrays (up to 16 each)
	if len(pce.FrontElementIsCPE) != 16 {
		t.Errorf("FrontElementIsCPE should have 16 elements")
	}
	if len(pce.FrontElementTagSelect) != 16 {
		t.Errorf("FrontElementTagSelect should have 16 elements")
	}
}

func TestProgramConfig_MixdownInfo(t *testing.T) {
	var pce ProgramConfig

	pce.MonoMixdownPresent = false
	pce.MonoMixdownElementNumber = 0
	pce.StereoMixdownPresent = false
	pce.StereoMixdownElementNumber = 0
	pce.MatrixMixdownIdxPresent = false
	pce.PseudoSurroundEnable = false
	pce.MatrixMixdownIdx = 0
}

func TestProgramConfig_Comment(t *testing.T) {
	var pce ProgramConfig

	pce.CommentFieldBytes = 0
	if len(pce.CommentFieldData) != 257 {
		t.Errorf("CommentFieldData should have 257 bytes")
	}
}

func TestProgramConfig_DerivedFields(t *testing.T) {
	var pce ProgramConfig

	// Derived channel counts
	pce.NumFrontChannels = 0
	pce.NumSideChannels = 0
	pce.NumBackChannels = 0
	pce.NumLFEChannels = 0

	// Channel mapping
	if len(pce.SCEChannel) != 16 {
		t.Errorf("SCEChannel should have 16 elements")
	}
	if len(pce.CPEChannel) != 16 {
		t.Errorf("CPEChannel should have 16 elements")
	}
}
```

**Step 5.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: ProgramConfig"

**Step 5.3: Write ProgramConfig implementation**

```go
// internal/syntax/pce.go
package syntax

// ProgramConfig contains Program Configuration Element data.
// The PCE describes the channel configuration for complex streams,
// mapping syntax elements to output channels.
//
// Ported from: program_config in ~/dev/faad2/libfaad/structs.h:103-144
type ProgramConfig struct {
	// Basic info
	ElementInstanceTag uint8 // Element instance tag
	ObjectType         uint8 // Audio object type
	SFIndex            uint8 // Sample frequency index

	// Element counts
	NumFrontChannelElements uint8 // Front channel element count
	NumSideChannelElements  uint8 // Side channel element count
	NumBackChannelElements  uint8 // Back channel element count
	NumLFEChannelElements   uint8 // LFE channel element count
	NumAssocDataElements    uint8 // Associated data element count
	NumValidCCElements      uint8 // Valid coupling channel count

	// Mixdown info
	MonoMixdownPresent         bool  // Mono mixdown element present
	MonoMixdownElementNumber   uint8 // Mono mixdown element number
	StereoMixdownPresent       bool  // Stereo mixdown element present
	StereoMixdownElementNumber uint8 // Stereo mixdown element number
	MatrixMixdownIdxPresent    bool  // Matrix mixdown present
	PseudoSurroundEnable       bool  // Pseudo surround enabled
	MatrixMixdownIdx           uint8 // Matrix mixdown index

	// Element configuration (up to 16 of each type)
	FrontElementIsCPE       [16]bool  // True if front element is CPE
	FrontElementTagSelect   [16]uint8 // Front element instance tags
	SideElementIsCPE        [16]bool  // True if side element is CPE
	SideElementTagSelect    [16]uint8 // Side element instance tags
	BackElementIsCPE        [16]bool  // True if back element is CPE
	BackElementTagSelect    [16]uint8 // Back element instance tags
	LFEElementTagSelect     [16]uint8 // LFE element instance tags
	AssocDataElementTagSelect [16]uint8 // Assoc data element tags
	CCElementIsIndSW        [16]bool  // CC element is independently switched
	ValidCCElementTagSelect [16]uint8 // Valid CC element tags

	// Total channel count (computed)
	Channels uint8

	// Comment field
	CommentFieldBytes uint8      // Comment length
	CommentFieldData  [257]uint8 // Comment data

	// Derived values (computed after parsing)
	NumFrontChannels uint8     // Total front channels
	NumSideChannels  uint8     // Total side channels
	NumBackChannels  uint8     // Total back channels
	NumLFEChannels   uint8     // Total LFE channels
	SCEChannel       [16]uint8 // SCE to channel mapping
	CPEChannel       [16]uint8 // CPE to channel mapping
}
```

**Step 5.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 5.5: Commit**

```bash
git add internal/syntax/pce.go internal/syntax/pce_test.go
git commit -m "feat(syntax): add ProgramConfig structure

Ported from FAAD2 structs.h:103-144. Program Configuration Element
describes channel layout for complex multi-channel streams."
```

---

## Task 6: ADTSHeader Structure

**Files:**
- Create: `internal/syntax/adts.go`
- Test: `internal/syntax/adts_test.go`

**Step 6.1: Write the test for ADTSHeader structure**

```go
// internal/syntax/adts_test.go
package syntax

import "testing"

func TestADTSHeader_Fields(t *testing.T) {
	var h ADTSHeader

	// Fixed header (28 bits)
	h.Syncword = 0x0FFF
	h.ID = 0 // MPEG-4
	h.Layer = 0
	h.ProtectionAbsent = true
	h.Profile = 1 // AAC LC
	h.SFIndex = 4 // 44100 Hz
	h.PrivateBit = false
	h.ChannelConfiguration = 2 // Stereo

	// Variable header
	h.Original = false
	h.Home = false
	h.CopyrightIDBit = false
	h.CopyrightIDStart = false
	h.AACFrameLength = 0
	h.ADTSBufferFullness = 0
	h.CRCCheck = 0
	h.NoRawDataBlocksInFrame = 0

	// Control
	h.OldFormat = false
}

func TestADTSHeader_Syncword(t *testing.T) {
	var h ADTSHeader
	h.Syncword = 0x0FFF

	if h.Syncword != 0x0FFF {
		t.Errorf("Syncword should be 0x0FFF, got 0x%X", h.Syncword)
	}
}

func TestADTSHeader_FrameLength(t *testing.T) {
	var h ADTSHeader

	// Frame length is 13 bits, max 8191
	h.AACFrameLength = 8191
	if h.AACFrameLength != 8191 {
		t.Errorf("AACFrameLength max should be 8191")
	}
}
```

**Step 6.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: ADTSHeader"

**Step 6.3: Write ADTSHeader implementation**

```go
// internal/syntax/adts.go
package syntax

// ADTSSyncword is the 12-bit sync pattern for ADTS frames.
const ADTSSyncword = 0x0FFF

// ADTSHeader contains Audio Data Transport Stream header data.
// ADTS is the most common AAC transport format (used in .aac files).
//
// Header structure (56 bits fixed + 16 bits CRC if present):
//   - syncword: 12 bits (0xFFF)
//   - id: 1 bit (0=MPEG-4, 1=MPEG-2)
//   - layer: 2 bits (always 0)
//   - protection_absent: 1 bit (1=no CRC)
//   - profile: 2 bits (0=Main, 1=LC, 2=SSR, 3=LTP)
//   - sf_index: 4 bits (sample rate index)
//   - private_bit: 1 bit
//   - channel_configuration: 3 bits
//   - original: 1 bit
//   - home: 1 bit
//   - copyright_id_bit: 1 bit
//   - copyright_id_start: 1 bit
//   - frame_length: 13 bits (includes header)
//   - buffer_fullness: 11 bits
//   - no_raw_data_blocks: 2 bits
//   - crc_check: 16 bits (if protection_absent=0)
//
// Ported from: adts_header in ~/dev/faad2/libfaad/structs.h:146-168
type ADTSHeader struct {
	Syncword             uint16 // 12 bits, must be 0xFFF
	ID                   uint8  // 1 bit: 0=MPEG-4, 1=MPEG-2
	Layer                uint8  // 2 bits: always 0
	ProtectionAbsent     bool   // 1 bit: true=no CRC
	Profile              uint8  // 2 bits: object type - 1
	SFIndex              uint8  // 4 bits: sample frequency index
	PrivateBit           bool   // 1 bit
	ChannelConfiguration uint8  // 3 bits: channel config
	Original             bool   // 1 bit
	Home                 bool   // 1 bit
	Emphasis             uint8  // 2 bits (MPEG-2 only)

	// Variable header
	CopyrightIDBit          bool   // 1 bit
	CopyrightIDStart        bool   // 1 bit
	AACFrameLength          uint16 // 13 bits: total frame bytes
	ADTSBufferFullness      uint16 // 11 bits: buffer fullness
	CRCCheck                uint16 // 16 bits (if protection_absent=0)
	NoRawDataBlocksInFrame  uint8  // 2 bits: num blocks - 1

	// Control parameter
	OldFormat bool // Use old ADTS format parsing
}

// HeaderSize returns the ADTS header size in bytes.
// Returns 7 if CRC is absent, 9 if CRC is present.
func (h *ADTSHeader) HeaderSize() int {
	if h.ProtectionAbsent {
		return 7
	}
	return 9
}

// DataSize returns the raw audio data size (frame length minus header).
func (h *ADTSHeader) DataSize() int {
	return int(h.AACFrameLength) - h.HeaderSize()
}
```

**Step 6.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 6.5: Commit**

```bash
git add internal/syntax/adts.go internal/syntax/adts_test.go
git commit -m "feat(syntax): add ADTSHeader structure

Ported from FAAD2 structs.h:146-168. Audio Data Transport Stream header
for the most common AAC transport format."
```

---

## Task 7: ADIFHeader Structure

**Files:**
- Create: `internal/syntax/adif.go`
- Test: `internal/syntax/adif_test.go`

**Step 7.1: Write the test for ADIFHeader structure**

```go
// internal/syntax/adif_test.go
package syntax

import "testing"

func TestADIFHeader_Fields(t *testing.T) {
	var h ADIFHeader

	h.CopyrightIDPresent = false
	h.OriginalCopy = false
	h.Bitrate = 0
	h.ADIFBufferFullness = 0
	h.NumProgramConfigElements = 0
	h.Home = false
	h.BitstreamType = 0
}

func TestADIFHeader_CopyrightID(t *testing.T) {
	var h ADIFHeader

	// Copyright ID is 9 bytes (72 bits)
	if len(h.CopyrightID) != 9 {
		t.Errorf("CopyrightID should have 9 bytes, got %d", len(h.CopyrightID))
	}
}

func TestADIFHeader_PCEs(t *testing.T) {
	var h ADIFHeader

	// Up to 16 PCEs
	if len(h.PCE) != 16 {
		t.Errorf("PCE should have 16 elements, got %d", len(h.PCE))
	}

	// Each PCE should be a ProgramConfig
	h.PCE[0].Channels = 2
}
```

**Step 7.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: ADIFHeader"

**Step 7.3: Write ADIFHeader implementation**

```go
// internal/syntax/adif.go
package syntax

// ADIFMagic is the 4-byte magic number for ADIF files.
var ADIFMagic = [4]byte{'A', 'D', 'I', 'F'}

// ADIFHeader contains Audio Data Interchange Format header data.
// ADIF is a simple header-at-beginning format, less common than ADTS.
// It contains one or more Program Configuration Elements.
//
// Ported from: adif_header in ~/dev/faad2/libfaad/structs.h:170-183
type ADIFHeader struct {
	CopyrightIDPresent       bool         // Copyright ID field present
	CopyrightID              [9]int8      // 72-bit copyright ID
	OriginalCopy             bool         // Original/copy flag
	Bitrate                  uint32       // Bit rate (bits/sec)
	ADIFBufferFullness       uint32       // Buffer fullness
	NumProgramConfigElements uint8        // Number of PCEs (0-15)
	Home                     bool         // Home flag
	BitstreamType            uint8        // 0=constant rate, 1=variable rate

	// Program Configuration Elements (up to 16)
	PCE [16]ProgramConfig
}
```

**Step 7.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 7.5: Commit**

```bash
git add internal/syntax/adif.go internal/syntax/adif_test.go
git commit -m "feat(syntax): add ADIFHeader structure

Ported from FAAD2 structs.h:170-183. Audio Data Interchange Format
header with embedded Program Configuration Elements."
```

---

## Task 8: DRCInfo Structure

**Files:**
- Create: `internal/syntax/drc.go`
- Test: `internal/syntax/drc_test.go`

**Step 8.1: Write the test for DRCInfo structure**

```go
// internal/syntax/drc_test.go
package syntax

import "testing"

func TestDRCInfo_Fields(t *testing.T) {
	var drc DRCInfo

	drc.Present = false
	drc.NumBands = 0
	drc.PCEInstanceTag = 0
	drc.ExcludedChnsPresent = false
	drc.ProgRefLevel = 0
}

func TestDRCInfo_Bands(t *testing.T) {
	var drc DRCInfo

	// Up to 17 DRC bands
	if len(drc.BandTop) != 17 {
		t.Errorf("BandTop should have 17 elements")
	}
	if len(drc.DynRngSgn) != 17 {
		t.Errorf("DynRngSgn should have 17 elements")
	}
	if len(drc.DynRngCtl) != 17 {
		t.Errorf("DynRngCtl should have 17 elements")
	}
}

func TestDRCInfo_ExcludeMask(t *testing.T) {
	var drc DRCInfo

	if len(drc.ExcludeMask) != MaxChannels {
		t.Errorf("ExcludeMask should have %d elements", MaxChannels)
	}
	if len(drc.AdditionalExcludedChns) != MaxChannels {
		t.Errorf("AdditionalExcludedChns should have %d elements", MaxChannels)
	}
}

func TestDRCInfo_Control(t *testing.T) {
	var drc DRCInfo

	// Control parameters are float32
	drc.Ctrl1 = 1.0
	drc.Ctrl2 = 1.0
}
```

**Step 8.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: DRCInfo"

**Step 8.3: Write DRCInfo implementation**

```go
// internal/syntax/drc.go
package syntax

// DRCInfo contains Dynamic Range Control information.
// DRC allows controlling the dynamic range of the output signal
// for different playback environments.
//
// Ported from: drc_info in ~/dev/faad2/libfaad/structs.h:85-101
type DRCInfo struct {
	Present             bool  // DRC data present in stream
	NumBands            uint8 // Number of DRC bands
	PCEInstanceTag      uint8 // Associated PCE instance tag
	ExcludedChnsPresent bool  // Excluded channels present

	BandTop   [17]uint8 // Top of each DRC band (SFB)
	ProgRefLevel uint8  // Program reference level

	DynRngSgn [17]uint8 // Dynamic range sign
	DynRngCtl [17]uint8 // Dynamic range control

	ExcludeMask           [MaxChannels]uint8 // Channel exclude mask
	AdditionalExcludedChns [MaxChannels]uint8 // Additional excluded channels

	// Control parameters (set by application)
	Ctrl1 float32 // DRC cut control (0.0-1.0)
	Ctrl2 float32 // DRC boost control (0.0-1.0)
}
```

**Step 8.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 8.5: Commit**

```bash
git add internal/syntax/drc.go internal/syntax/drc_test.go
git commit -m "feat(syntax): add DRCInfo structure

Ported from FAAD2 structs.h:85-101. Dynamic Range Control for
adjusting output dynamics for different playback environments."
```

---

## Task 9: LTPInfo Structure (Optional Profile)

**Files:**
- Create: `internal/syntax/ltp.go`
- Test: `internal/syntax/ltp_test.go`

**Step 9.1: Write the test for LTPInfo structure**

```go
// internal/syntax/ltp_test.go
package syntax

import "testing"

func TestLTPInfo_Fields(t *testing.T) {
	var ltp LTPInfo

	ltp.LastBand = 0
	ltp.DataPresent = false
	ltp.Lag = 0
	ltp.LagUpdate = false
	ltp.Coef = 0
}

func TestLTPInfo_LongWindow(t *testing.T) {
	var ltp LTPInfo

	if len(ltp.LongUsed) != MaxSFB {
		t.Errorf("LongUsed should have %d elements", MaxSFB)
	}
}

func TestLTPInfo_ShortWindows(t *testing.T) {
	var ltp LTPInfo

	// 8 short windows
	if len(ltp.ShortUsed) != 8 {
		t.Errorf("ShortUsed should have 8 elements")
	}
	if len(ltp.ShortLagPresent) != 8 {
		t.Errorf("ShortLagPresent should have 8 elements")
	}
	if len(ltp.ShortLag) != 8 {
		t.Errorf("ShortLag should have 8 elements")
	}
}
```

**Step 9.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: LTPInfo"

**Step 9.3: Write LTPInfo implementation**

```go
// internal/syntax/ltp.go
package syntax

// LTPInfo contains Long Term Prediction data.
// LTP uses previously decoded samples to improve coding efficiency.
// This is used only for the LTP audio object type.
//
// Ported from: ltp_info in ~/dev/faad2/libfaad/structs.h:186-197
type LTPInfo struct {
	LastBand    uint8  // Last band using LTP
	DataPresent bool   // LTP data present
	Lag         uint16 // LTP lag in samples (0-2047)
	LagUpdate   bool   // Lag value updated
	Coef        uint8  // LTP coefficient index (0-7)

	// Per-SFB usage for long windows
	LongUsed [MaxSFB]bool

	// Per-window info for short windows
	ShortUsed       [8]bool   // LTP used per short window
	ShortLagPresent [8]bool   // Short lag present per window
	ShortLag        [8]uint8  // Short window lag values
}
```

**Step 9.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 9.5: Commit**

```bash
git add internal/syntax/ltp.go internal/syntax/ltp_test.go
git commit -m "feat(syntax): add LTPInfo structure

Ported from FAAD2 structs.h:186-197. Long Term Prediction data
for the LTP audio object type profile."
```

---

## Task 10: Update ICStream with LTPInfo

**Files:**
- Modify: `internal/syntax/ics.go`
- Modify: `internal/syntax/ics_test.go`

**Step 10.1: Add test for LTP fields in ICStream**

Add to `ics_test.go`:

```go
func TestICStream_LTPFields(t *testing.T) {
	var ics ICStream

	// LTP info (for LTP profile)
	_ = ics.LTP.DataPresent
	_ = ics.LTP2.DataPresent
}
```

**Step 10.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "ics.LTP undefined"

**Step 10.3: Update ICStream with LTP fields**

Add to ICStream struct in `ics.go`, after the TNS field:

```go
	// Optional profile data
	LTP  LTPInfo // LTP data (LTP profile, first predictor)
	LTP2 LTPInfo // LTP data (LTP profile, second predictor for CPE)
```

**Step 10.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 10.5: Commit**

```bash
git add internal/syntax/ics.go internal/syntax/ics_test.go
git commit -m "feat(syntax): add LTP fields to ICStream

Extended ICStream with LTP and LTP2 fields for Long Term Prediction
support in the LTP audio object type."
```

---

## Task 11: Final Validation

**Step 11.1: Run all tests**

```bash
make check
```

Expected: All tests pass, no lint errors.

**Step 11.2: Verify file sizes**

All files should be under ~300 lines:
- `pulse.go`: ~30 lines
- `tns.go`: ~40 lines
- `ics.go`: ~100 lines
- `element.go`: ~30 lines
- `pce.go`: ~80 lines
- `adts.go`: ~80 lines
- `adif.go`: ~40 lines
- `drc.go`: ~50 lines
- `ltp.go`: ~40 lines

**Step 11.3: Final commit with all changes**

If any uncommitted changes remain:

```bash
git add .
git commit -m "chore(syntax): phase 3.1 complete - core data structures

All core AAC data structures ported from FAAD2 structs.h:
- PulseInfo, TNSInfo, ICStream, Element
- ProgramConfig, ADTSHeader, ADIFHeader
- DRCInfo, LTPInfo

Ready for Phase 3.2: syntax parsing functions."
```

---

## Summary

| Task | Structure | FAAD2 Lines | Status |
|------|-----------|-------------|--------|
| 1 | PulseInfo | 210-216 | ☐ |
| 2 | TNSInfo | 218-227 | ☐ |
| 3 | ICStream | 240-301 | ☐ |
| 4 | Element | 303-313 | ☐ |
| 5 | ProgramConfig | 103-144 | ☐ |
| 6 | ADTSHeader | 146-168 | ☐ |
| 7 | ADIFHeader | 170-183 | ☐ |
| 8 | DRCInfo | 85-101 | ☐ |
| 9 | LTPInfo | 186-197 | ☐ |
| 10 | ICStream+LTP | - | ☐ |
| 11 | Validation | - | ☐ |

**Total estimated lines:** ~500 lines of Go code
**Total tasks:** 11 (with ~50 individual steps)
