# ADTS Header Parser Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the ADTS (Audio Data Transport Stream) header parser to extract frame metadata from AAC streams.

**Architecture:** The parser reads ADTS frames using the existing `bits.Reader`, populating the already-defined `ADTSHeader` struct. It includes sync word recovery (like FAAD2) to handle streams that may be corrupted or joined mid-stream. The implementation follows FAAD2's `adts_frame()` which combines `adts_fixed_header()`, `adts_variable_header()`, and `adts_error_check()`.

**Tech Stack:** Pure Go, no dependencies. Uses `internal/bits.Reader` for bitstream operations.

---

## Current State

**Already implemented:**
- `internal/syntax/adts.go` - `ADTSHeader` struct with all fields
- `internal/bits/reader.go` - Full bit reader with `GetBits`, `ShowBits`, etc.
- `internal/bits/reader_adts_test.go` - Manual ADTS parsing tests (validates reader works)
- `testdata/sine1k.aac` - Test AAC file (FFmpeg-generated)
- `scripts/faad2_debug.c` - FAAD2 reference data generator

**Missing:**
- `ParseADTS(r *bits.Reader) (*ADTSHeader, error)` - The actual parsing function
- `FindSyncword(r *bits.Reader) error` - Sync word recovery
- FAAD2 reference comparison tests

---

## Task 1: Add FindSyncword Function

**Files:**
- Modify: `internal/syntax/adts.go`
- Test: `internal/syntax/adts_test.go`

**Step 1: Write the failing test**

Add to `internal/syntax/adts_test.go`:

```go
func TestFindSyncword_AtStart(t *testing.T) {
	// Valid ADTS frame starts with 0xFFF
	data := []byte{0xFF, 0xF1, 0x4C, 0x80, 0x00, 0x00, 0x00}
	r := bits.NewReader(data)

	err := FindSyncword(r)
	if err != nil {
		t.Fatalf("FindSyncword failed: %v", err)
	}

	// Should have consumed the 12-bit syncword
	consumed := r.GetProcessedBits()
	if consumed != 12 {
		t.Errorf("consumed %d bits, want 12", consumed)
	}
}

func TestFindSyncword_WithGarbage(t *testing.T) {
	// 3 bytes of garbage, then valid ADTS sync
	data := []byte{0x00, 0xAA, 0xBB, 0xFF, 0xF1, 0x4C, 0x80, 0x00}
	r := bits.NewReader(data)

	err := FindSyncword(r)
	if err != nil {
		t.Fatalf("FindSyncword failed: %v", err)
	}

	// Should have skipped 3 bytes (24 bits) + consumed 12-bit syncword = 36 bits
	consumed := r.GetProcessedBits()
	if consumed != 36 {
		t.Errorf("consumed %d bits, want 36", consumed)
	}
}

func TestFindSyncword_NotFound(t *testing.T) {
	// No syncword in data
	data := make([]byte, 800)
	for i := range data {
		data[i] = 0xAA
	}
	r := bits.NewReader(data)

	err := FindSyncword(r)
	if err == nil {
		t.Fatal("expected error for missing syncword")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: FindSyncword"

**Step 3: Write minimal implementation**

Add to `internal/syntax/adts.go`:

```go
import (
	"github.com/llehouerou/go-aac/internal/bits"
)

// MaxSyncSearchBytes is the maximum bytes to search for ADTS syncword.
// Matches FAAD2's limit of 768 bytes.
const MaxSyncSearchBytes = 768

// FindSyncword searches for the ADTS syncword (0xFFF) in the bitstream.
// It will skip up to MaxSyncSearchBytes looking for the sync pattern.
// After finding the syncword, the 12 syncword bits are consumed.
// Returns ErrADTSSyncwordNotFound if no syncword is found.
//
// Ported from: adts_fixed_header() sync recovery loop in
// ~/dev/faad2/libfaad/syntax.c:2466-2482
func FindSyncword(r *bits.Reader) error {
	for i := 0; i < MaxSyncSearchBytes; i++ {
		syncword := r.ShowBits(12)
		if syncword == ADTSSyncword {
			// Found it - consume the syncword
			r.FlushBits(12)
			return nil
		}
		// Skip 8 bits and try again
		r.FlushBits(8)
	}
	return ErrADTSSyncwordNotFound
}
```

Also add at the top of the file (after imports):

```go
import "errors"

// ErrADTSSyncwordNotFound is returned when no ADTS syncword is found.
var ErrADTSSyncwordNotFound = errors.New("unable to find ADTS syncword")
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/adts.go internal/syntax/adts_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add FindSyncword function for ADTS sync recovery

Implements sync word recovery matching FAAD2's behavior:
- Searches up to 768 bytes for 0xFFF pattern
- Skips 8 bits at a time when searching
- Consumes the 12-bit syncword when found

Ported from: adts_fixed_header() in ~/dev/faad2/libfaad/syntax.c:2466-2482

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Add parseFixedHeader Helper

**Files:**
- Modify: `internal/syntax/adts.go`
- Test: `internal/syntax/adts_test.go`

**Step 1: Write the failing test**

Add to `internal/syntax/adts_test.go`:

```go
func TestParseFixedHeader(t *testing.T) {
	// Manually construct ADTS fixed header (28 bits after syncword):
	// syncword=0xFFF (12 bits) - already consumed by FindSyncword
	// id=0 (1 bit) - MPEG-4
	// layer=00 (2 bits)
	// protection_absent=1 (1 bit) - no CRC
	// profile=01 (2 bits) - LC
	// sf_index=0100 (4 bits) - 44100 Hz
	// private_bit=0 (1 bit)
	// channel_config=010 (3 bits) - stereo
	// original=0 (1 bit)
	// home=0 (1 bit)
	//
	// Binary: 0 00 1 01 0100 0 010 0 0 = 0x29 0x40
	// Full frame: FF F1 4C 80 (first 4 bytes of typical ADTS)
	// After syncword: 1 4C 80 (remaining bits)

	data := []byte{0xFF, 0xF1, 0x4C, 0x80, 0x00, 0x1F, 0xFC}
	r := bits.NewReader(data)

	// Skip syncword (would be done by FindSyncword)
	r.FlushBits(12)

	h := &ADTSHeader{Syncword: ADTSSyncword}
	err := parseFixedHeader(r, h)
	if err != nil {
		t.Fatalf("parseFixedHeader failed: %v", err)
	}

	// Verify parsed values
	if h.ID != 0 {
		t.Errorf("ID = %d, want 0 (MPEG-4)", h.ID)
	}
	if h.Layer != 0 {
		t.Errorf("Layer = %d, want 0", h.Layer)
	}
	if !h.ProtectionAbsent {
		t.Error("ProtectionAbsent = false, want true")
	}
	if h.Profile != 1 {
		t.Errorf("Profile = %d, want 1 (LC)", h.Profile)
	}
	if h.SFIndex != 4 {
		t.Errorf("SFIndex = %d, want 4 (44100Hz)", h.SFIndex)
	}
	if h.ChannelConfiguration != 2 {
		t.Errorf("ChannelConfiguration = %d, want 2 (stereo)", h.ChannelConfiguration)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: parseFixedHeader"

**Step 3: Write minimal implementation**

Add to `internal/syntax/adts.go`:

```go
// parseFixedHeader parses the ADTS fixed header (16 bits after syncword).
// The syncword must already be consumed before calling this function.
//
// Ported from: adts_fixed_header() in ~/dev/faad2/libfaad/syntax.c:2484-2511
func parseFixedHeader(r *bits.Reader, h *ADTSHeader) error {
	h.ID = r.Get1Bit()
	h.Layer = uint8(r.GetBits(2))
	h.ProtectionAbsent = r.Get1Bit() == 1
	h.Profile = uint8(r.GetBits(2))
	h.SFIndex = uint8(r.GetBits(4))
	h.PrivateBit = r.Get1Bit() == 1
	h.ChannelConfiguration = uint8(r.GetBits(3))
	h.Original = r.Get1Bit() == 1
	h.Home = r.Get1Bit() == 1

	// Old ADTS format (removed in corrigendum 14496-3:2002)
	// Only for MPEG-2 (id=1) with old_format flag
	if h.OldFormat && h.ID == 0 {
		h.Emphasis = uint8(r.GetBits(2))
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/adts.go internal/syntax/adts_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add parseFixedHeader for ADTS fixed header parsing

Parses 16 bits of ADTS fixed header after syncword:
- id, layer, protection_absent
- profile, sf_index, private_bit
- channel_configuration, original, home
- emphasis (old format only)

Ported from: adts_fixed_header() in ~/dev/faad2/libfaad/syntax.c:2484-2511

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Add parseVariableHeader Helper

**Files:**
- Modify: `internal/syntax/adts.go`
- Test: `internal/syntax/adts_test.go`

**Step 1: Write the failing test**

Add to `internal/syntax/adts_test.go`:

```go
func TestParseVariableHeader(t *testing.T) {
	// Variable header is 28 bits:
	// copyright_id_bit=0 (1 bit)
	// copyright_id_start=0 (1 bit)
	// frame_length=0x0180 (13 bits) = 384 bytes
	// buffer_fullness=0x7FF (11 bits) = VBR marker
	// num_raw_blocks=0 (2 bits) = 1 raw block

	// Construct: 0 0 0000110000000 11111111111 00
	// = 00 0000 1100 0000 0111 1111 1111 00
	// = 0x00 0xC0 0x7F 0xFC
	data := []byte{0x00, 0xC0, 0x7F, 0xFC}
	r := bits.NewReader(data)

	h := &ADTSHeader{}
	parseVariableHeader(r, h)

	if h.CopyrightIDBit {
		t.Error("CopyrightIDBit = true, want false")
	}
	if h.CopyrightIDStart {
		t.Error("CopyrightIDStart = true, want false")
	}
	if h.AACFrameLength != 384 {
		t.Errorf("AACFrameLength = %d, want 384", h.AACFrameLength)
	}
	if h.ADTSBufferFullness != 0x7FF {
		t.Errorf("ADTSBufferFullness = 0x%X, want 0x7FF", h.ADTSBufferFullness)
	}
	if h.NoRawDataBlocksInFrame != 0 {
		t.Errorf("NoRawDataBlocksInFrame = %d, want 0", h.NoRawDataBlocksInFrame)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: parseVariableHeader"

**Step 3: Write minimal implementation**

Add to `internal/syntax/adts.go`:

```go
// parseVariableHeader parses the ADTS variable header (28 bits).
//
// Ported from: adts_variable_header() in ~/dev/faad2/libfaad/syntax.c:2517-2528
func parseVariableHeader(r *bits.Reader, h *ADTSHeader) {
	h.CopyrightIDBit = r.Get1Bit() == 1
	h.CopyrightIDStart = r.Get1Bit() == 1
	h.AACFrameLength = uint16(r.GetBits(13))
	h.ADTSBufferFullness = uint16(r.GetBits(11))
	h.NoRawDataBlocksInFrame = uint8(r.GetBits(2))
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/adts.go internal/syntax/adts_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add parseVariableHeader for ADTS variable header

Parses 28 bits of ADTS variable header:
- copyright_identification_bit/start
- aac_frame_length (13 bits)
- adts_buffer_fullness (11 bits)
- no_raw_data_blocks_in_frame (2 bits)

Ported from: adts_variable_header() in ~/dev/faad2/libfaad/syntax.c:2517-2528

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Add parseErrorCheck Helper

**Files:**
- Modify: `internal/syntax/adts.go`
- Test: `internal/syntax/adts_test.go`

**Step 1: Write the failing test**

Add to `internal/syntax/adts_test.go`:

```go
func TestParseErrorCheck_WithCRC(t *testing.T) {
	// CRC is 16 bits, read when protection_absent=0
	data := []byte{0xAB, 0xCD, 0x00, 0x00}
	r := bits.NewReader(data)

	h := &ADTSHeader{ProtectionAbsent: false}
	parseErrorCheck(r, h)

	if h.CRCCheck != 0xABCD {
		t.Errorf("CRCCheck = 0x%X, want 0xABCD", h.CRCCheck)
	}

	consumed := r.GetProcessedBits()
	if consumed != 16 {
		t.Errorf("consumed %d bits, want 16", consumed)
	}
}

func TestParseErrorCheck_NoCRC(t *testing.T) {
	data := []byte{0xAB, 0xCD, 0x00, 0x00}
	r := bits.NewReader(data)

	h := &ADTSHeader{ProtectionAbsent: true}
	parseErrorCheck(r, h)

	// Should not consume any bits when protection_absent=true
	consumed := r.GetProcessedBits()
	if consumed != 0 {
		t.Errorf("consumed %d bits, want 0 (no CRC)", consumed)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: parseErrorCheck"

**Step 3: Write minimal implementation**

Add to `internal/syntax/adts.go`:

```go
// parseErrorCheck reads the CRC if protection is enabled.
//
// Ported from: adts_error_check() in ~/dev/faad2/libfaad/syntax.c:2532-2538
func parseErrorCheck(r *bits.Reader, h *ADTSHeader) {
	if !h.ProtectionAbsent {
		h.CRCCheck = uint16(r.GetBits(16))
	}
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/adts.go internal/syntax/adts_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add parseErrorCheck for ADTS CRC handling

Reads 16-bit CRC when protection_absent=0.
Skips CRC read when protection_absent=1.

Ported from: adts_error_check() in ~/dev/faad2/libfaad/syntax.c:2532-2538

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Add Main ParseADTS Function

**Files:**
- Modify: `internal/syntax/adts.go`
- Test: `internal/syntax/adts_test.go`

**Step 1: Write the failing test**

Add to `internal/syntax/adts_test.go`:

```go
func TestParseADTS_RealFile(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sine1k.aac")
	if err != nil {
		t.Skipf("Test file not available: %v", err)
	}

	r := bits.NewReader(data)
	h, err := ParseADTS(r)
	if err != nil {
		t.Fatalf("ParseADTS failed: %v", err)
	}

	// Validate syncword was found
	if h.Syncword != ADTSSyncword {
		t.Errorf("Syncword = 0x%X, want 0x%X", h.Syncword, ADTSSyncword)
	}

	// Layer must be 0
	if h.Layer != 0 {
		t.Errorf("Layer = %d, want 0", h.Layer)
	}

	// Profile should be reasonable (0-3)
	if h.Profile > 3 {
		t.Errorf("Profile = %d, out of range", h.Profile)
	}

	// SFIndex should be valid (0-12)
	if h.SFIndex > 12 {
		t.Errorf("SFIndex = %d, out of range", h.SFIndex)
	}

	// Frame length should be reasonable
	if h.AACFrameLength < 7 || h.AACFrameLength > 8192 {
		t.Errorf("AACFrameLength = %d, out of range", h.AACFrameLength)
	}

	// HeaderSize should match ProtectionAbsent
	expectedSize := 7
	if !h.ProtectionAbsent {
		expectedSize = 9
	}
	if h.HeaderSize() != expectedSize {
		t.Errorf("HeaderSize() = %d, want %d", h.HeaderSize(), expectedSize)
	}

	t.Logf("Parsed ADTS header: Profile=%d, SFIndex=%d, Channels=%d, FrameLen=%d",
		h.Profile, h.SFIndex, h.ChannelConfiguration, h.AACFrameLength)
}

func TestParseADTS_MultipleFrames(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sine1k.aac")
	if err != nil {
		t.Skipf("Test file not available: %v", err)
	}

	offset := 0
	frameCount := 0
	maxFrames := 5

	for offset < len(data) && frameCount < maxFrames {
		r := bits.NewReader(data[offset:])
		h, err := ParseADTS(r)
		if err != nil {
			t.Fatalf("Frame %d: ParseADTS failed: %v", frameCount, err)
		}

		if h.AACFrameLength < 7 {
			t.Fatalf("Frame %d: invalid frame length %d", frameCount, h.AACFrameLength)
		}

		t.Logf("Frame %d: length=%d, channels=%d", frameCount, h.AACFrameLength, h.ChannelConfiguration)

		offset += int(h.AACFrameLength)
		frameCount++
	}

	if frameCount == 0 {
		t.Error("No frames parsed")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: ParseADTS"

**Step 3: Write minimal implementation**

Add to `internal/syntax/adts.go`:

```go
// ParseADTS parses a complete ADTS frame header from the bitstream.
// It searches for the syncword, then parses fixed header, variable header,
// and CRC (if present).
//
// Returns the parsed header or an error if no syncword is found.
//
// Ported from: adts_frame() in ~/dev/faad2/libfaad/syntax.c:2449-2458
func ParseADTS(r *bits.Reader) (*ADTSHeader, error) {
	h := &ADTSHeader{}

	// Find and consume syncword
	if err := FindSyncword(r); err != nil {
		return nil, err
	}
	h.Syncword = ADTSSyncword

	// Parse fixed header (16 bits)
	if err := parseFixedHeader(r, h); err != nil {
		return nil, err
	}

	// Parse variable header (28 bits)
	parseVariableHeader(r, h)

	// Parse error check (CRC if present)
	parseErrorCheck(r, h)

	return h, nil
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/adts.go internal/syntax/adts_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add ParseADTS for complete ADTS header parsing

Main entry point for ADTS parsing that combines:
- FindSyncword: sync word recovery
- parseFixedHeader: id, layer, profile, sample rate, channels
- parseVariableHeader: frame length, buffer fullness
- parseErrorCheck: CRC handling

Ported from: adts_frame() in ~/dev/faad2/libfaad/syntax.c:2449-2458

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Add FAAD2 Reference Comparison Test

**Files:**
- Test: `internal/syntax/adts_faad2_test.go` (new file)

**Step 1: Write the test file**

Create `internal/syntax/adts_faad2_test.go`:

```go
package syntax

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

// TestParseADTS_FAAD2Reference compares parsed ADTS headers against
// FAAD2 reference data generated by scripts/faad2_debug.
func TestParseADTS_FAAD2Reference(t *testing.T) {
	testFile := "../../testdata/sine1k.aac"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test file not available")
	}

	// Generate reference data using faad2_debug
	refDir := t.TempDir()

	// Check if faad2_debug exists
	faad2Debug := "../../scripts/faad2_debug"
	if _, err := os.Stat(faad2Debug); os.IsNotExist(err) {
		t.Skip("faad2_debug not built - run 'make' in scripts/")
	}

	cmd := exec.Command(faad2Debug, testFile, refDir, "5")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("faad2_debug failed: %v\n%s", err, output)
	}

	// Read test file
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Parse frames and compare
	offset := 0
	for frame := 0; frame < 5; frame++ {
		refPath := filepath.Join(refDir, "frame_%04d_adts.bin")
		refPath = filepath.Join(refDir,
			"frame_"+padFrame(frame)+"_adts.bin")

		refData, err := os.ReadFile(refPath)
		if err != nil {
			t.Logf("Frame %d: no reference data (end of file?)", frame)
			break
		}
		if len(refData) != 16 {
			t.Fatalf("Frame %d: invalid reference size %d", frame, len(refData))
		}

		// Parse with Go
		r := bits.NewReader(data[offset:])
		h, err := ParseADTS(r)
		if err != nil {
			t.Fatalf("Frame %d: ParseADTS failed: %v", frame, err)
		}

		// Compare against reference (see faad2_debug.c dump_adts_header format)
		// buf[0-1]: syncword (big-endian)
		// buf[2]: id
		// buf[3]: layer
		// buf[4]: protection_absent
		// buf[5]: profile
		// buf[6]: sf_index
		// buf[7]: private_bit
		// buf[8]: channel_config
		// buf[9]: original
		// buf[10]: home
		// buf[11-12]: frame_length (big-endian)
		// buf[13-14]: buffer_fullness (big-endian)
		// buf[15]: num_raw_blocks

		refSyncword := uint16(refData[0])<<8 | uint16(refData[1])
		if h.Syncword != refSyncword {
			t.Errorf("Frame %d: Syncword = 0x%X, ref = 0x%X",
				frame, h.Syncword, refSyncword)
		}

		if h.ID != refData[2] {
			t.Errorf("Frame %d: ID = %d, ref = %d", frame, h.ID, refData[2])
		}

		if h.Layer != refData[3] {
			t.Errorf("Frame %d: Layer = %d, ref = %d", frame, h.Layer, refData[3])
		}

		refProtAbsent := refData[4] == 1
		if h.ProtectionAbsent != refProtAbsent {
			t.Errorf("Frame %d: ProtectionAbsent = %v, ref = %v",
				frame, h.ProtectionAbsent, refProtAbsent)
		}

		if h.Profile != refData[5] {
			t.Errorf("Frame %d: Profile = %d, ref = %d", frame, h.Profile, refData[5])
		}

		if h.SFIndex != refData[6] {
			t.Errorf("Frame %d: SFIndex = %d, ref = %d", frame, h.SFIndex, refData[6])
		}

		if h.ChannelConfiguration != refData[8] {
			t.Errorf("Frame %d: ChannelConfig = %d, ref = %d",
				frame, h.ChannelConfiguration, refData[8])
		}

		refFrameLen := uint16(refData[11])<<8 | uint16(refData[12])
		if h.AACFrameLength != refFrameLen {
			t.Errorf("Frame %d: FrameLength = %d, ref = %d",
				frame, h.AACFrameLength, refFrameLen)
		}

		refBufFull := uint16(refData[13])<<8 | uint16(refData[14])
		if h.ADTSBufferFullness != refBufFull {
			t.Errorf("Frame %d: BufferFullness = %d, ref = %d",
				frame, h.ADTSBufferFullness, refBufFull)
		}

		if h.NoRawDataBlocksInFrame != refData[15] {
			t.Errorf("Frame %d: NumRawBlocks = %d, ref = %d",
				frame, h.NoRawDataBlocksInFrame, refData[15])
		}

		t.Logf("Frame %d: PASS (length=%d, channels=%d)",
			frame, h.AACFrameLength, h.ChannelConfiguration)

		offset += int(h.AACFrameLength)
	}
}

func padFrame(n int) string {
	return fmt.Sprintf("%04d", n)
}
```

Add at the top after package:

```go
import "fmt"
```

**Step 2: Run test**

Run: `make test PKG=./internal/syntax`
Expected: PASS (or SKIP if faad2_debug not built)

**Step 3: Commit**

```bash
git add internal/syntax/adts_faad2_test.go
git commit -m "$(cat <<'EOF'
test(syntax): add FAAD2 reference comparison tests for ADTS

Validates ParseADTS output against faad2_debug reference data:
- Compares all header fields per frame
- Tests first 5 frames of sine1k.aac
- Skips gracefully if faad2_debug not available

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Add ParseADTSWithOptions for OldFormat Support

**Files:**
- Modify: `internal/syntax/adts.go`
- Test: `internal/syntax/adts_test.go`

**Step 1: Write the failing test**

Add to `internal/syntax/adts_test.go`:

```go
func TestParseADTSWithOptions_OldFormat(t *testing.T) {
	// Test that OldFormat flag is respected
	data := []byte{0xFF, 0xF1, 0x4C, 0x80, 0x00, 0x1F, 0xFC, 0x00}
	r := bits.NewReader(data)

	opts := ADTSOptions{OldFormat: true}
	h, err := ParseADTSWithOptions(r, opts)
	if err != nil {
		t.Fatalf("ParseADTSWithOptions failed: %v", err)
	}

	if !h.OldFormat {
		t.Error("OldFormat flag not preserved")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: ParseADTSWithOptions" or "undefined: ADTSOptions"

**Step 3: Write minimal implementation**

Add to `internal/syntax/adts.go`:

```go
// ADTSOptions contains optional settings for ADTS parsing.
type ADTSOptions struct {
	// OldFormat enables old ADTS format parsing (emphasis field for MPEG-2).
	// This was removed in corrigendum 14496-3:2002.
	OldFormat bool
}

// ParseADTSWithOptions parses an ADTS header with additional options.
func ParseADTSWithOptions(r *bits.Reader, opts ADTSOptions) (*ADTSHeader, error) {
	h := &ADTSHeader{
		OldFormat: opts.OldFormat,
	}

	if err := FindSyncword(r); err != nil {
		return nil, err
	}
	h.Syncword = ADTSSyncword

	if err := parseFixedHeader(r, h); err != nil {
		return nil, err
	}

	parseVariableHeader(r, h)
	parseErrorCheck(r, h)

	return h, nil
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/adts.go internal/syntax/adts_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add ParseADTSWithOptions for old format support

Adds ADTSOptions struct and ParseADTSWithOptions function to support:
- OldFormat flag for MPEG-2 emphasis field parsing

This matches FAAD2's adts.old_format configuration option.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Run Full Test Suite and Cleanup

**Step 1: Run make check**

Run: `make check`
Expected: PASS (all formatting, linting, and tests pass)

**Step 2: Fix any issues**

If there are any formatting or lint issues, fix them:

Run: `make fmt`
Run: `make lint`

Fix any reported issues.

**Step 3: Final commit if needed**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore(syntax): cleanup ADTS parser implementation

Final cleanup after implementation:
- Fixed any formatting issues
- Resolved lint warnings
- All tests passing

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Summary

After completing all tasks, the ADTS parser will provide:

1. **FindSyncword** - Sync word recovery (up to 768 bytes search)
2. **ParseADTS** - Simple parsing with defaults
3. **ParseADTSWithOptions** - Parsing with configurable options
4. **Helper functions** - parseFixedHeader, parseVariableHeader, parseErrorCheck

**Total new/modified lines:** ~150 lines of implementation + ~250 lines of tests

**Files modified:**
- `internal/syntax/adts.go` - Added parsing functions
- `internal/syntax/adts_test.go` - Added unit tests
- `internal/syntax/adts_faad2_test.go` - Added FAAD2 comparison tests (new file)

**Validation:**
- Unit tests for each parsing component
- Integration tests with real AAC files
- FAAD2 reference comparison (when faad2_debug is available)
