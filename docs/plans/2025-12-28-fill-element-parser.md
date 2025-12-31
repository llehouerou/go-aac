# Fill Element & Extension Payload Parser Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the Fill Element parser (ID_FIL) and its extension payloads including Dynamic Range Control (DRC) parsing.

**Architecture:** The Fill Element contains extension data such as DRC and SBR. We parse the fill element count, then dispatch to extension_payload which handles different extension types. DRC parsing populates the existing DRCInfo struct. SBR parsing is stubbed for Phase 8.

**Tech Stack:** Pure Go, bits.Reader for bitstream reading, existing DRCInfo struct.

---

## Background

From FAAD2 `syntax.c`:
- `fill_element()` (lines 1110-1197): Parses ID_FIL elements containing extension payloads
- `extension_payload()` (lines 2240-2299): Dispatches to specific extension handlers
- `dynamic_range_info()` (lines 2302-2364): Parses DRC data into drc_info struct
- `excluded_channels()` (lines 2367-2394): Parses DRC excluded channel mask
- `data_stream_element()` (lines 1080-1107): Parses ID_DSE elements (data stream)

The DRCInfo struct already exists in `internal/syntax/drc.go`.

---

## Task 1: Add Extension Type Constants for SBR

**Files:**
- Modify: `internal/syntax/constants.go`

**Step 1.1: Add SBR extension types**

Add to the ExtensionType constants section:

```go
// Extension Types.
const (
	ExtFil          ExtensionType = 0  // Filler extension
	ExtFillData     ExtensionType = 1  // Fill with MPEG surround data
	ExtDataElement  ExtensionType = 2  // Data element
	ExtDynamicRange ExtensionType = 11 // Dynamic Range Control
	ExtSBRData      ExtensionType = 13 // SBR data (without CRC)
	ExtSBRDataCRC   ExtensionType = 14 // SBR data (with CRC)
)
```

**Step 1.2: Run tests to verify no regressions**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 1.3: Commit**

```bash
git add internal/syntax/constants.go
git commit -m "feat(syntax): add SBR extension type constants"
```

---

## Task 2: Create Data Stream Element Parser

**Files:**
- Create: `internal/syntax/dse.go`
- Create: `internal/syntax/dse_test.go`

**Step 2.1: Write the failing test for ParseDataStreamElement**

```go
// internal/syntax/dse_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseDataStreamElement_Basic(t *testing.T) {
	// DSE format:
	// - element_instance_tag: 4 bits = 0x5
	// - data_byte_align_flag: 1 bit = 0 (not aligned)
	// - count: 8 bits = 3 (3 bytes of data)
	// - data_stream_byte[0-2]: 3 bytes = 0xAA, 0xBB, 0xCC
	// Binary: 0101 0 00000011 10101010 10111011 11001100
	// Hex: 5 (4b) | 0 (1b) | 03 (8b) | AA BB CC
	// = 0101_0000_0001_1101_0101_0101_1101_1110_0110_0
	// Byte aligned: 50 1D 55 5E E6 (with padding)
	data := []byte{0x50, 0x1A, 0xAB, 0xBC, 0xC0} // Adjusted for bit alignment
	r := bits.NewReader(data)

	// Actually let's compute this more carefully:
	// element_instance_tag (4 bits): 0101 = 5
	// data_byte_align_flag (1 bit): 0
	// count (8 bits): 00000011 = 3
	// data_stream_byte[0] (8 bits): 10101010 = 0xAA
	// data_stream_byte[1] (8 bits): 10111011 = 0xBB
	// data_stream_byte[2] (8 bits): 11001100 = 0xCC
	//
	// Combined: 0101 0 00000011 10101010 10111011 11001100
	// = 0101_0000_0001_1101_0101_0101_1101_1110_0110_0xxx
	// Byte 0: 0101_0000 = 0x50
	// Byte 1: 0001_1101 = 0x1D (wait, let me recalculate)

	// Let's be precise:
	// Bits: 0101 0 00000011 10101010 10111011 11001100
	// Position 0-3: 0101 (tag=5)
	// Position 4: 0 (align=false)
	// Position 5-12: 00000011 (count=3)
	// Position 13-20: 10101010 (0xAA)
	// Position 21-28: 10111011 (0xBB)
	// Position 29-36: 11001100 (0xCC)
	//
	// Bytes:
	// Byte 0 (bits 0-7): 0101_0000 = 0x50
	// Byte 1 (bits 8-15): 0001_1101 = 0x1D -- wait that's not right
	// Let me re-number: bits 0-3 = 0101, bit 4 = 0, bits 5-12 = 00000011
	// So byte 0 = 0101_0_000 = 0x50 (with bit 4 and partial bit 5-7)
	// Actually bits 5-7 of first byte are the first 3 bits of count
	// count = 00000011, so first 3 bits = 000
	// byte 0 = 0101_0_000 = 0x50
	// byte 1 = 00011_101 = 0x1D -- still not right

	// Let me just build it byte by byte:
	// Bit stream: tag(4) align(1) count(8) data(24)
	// = 0101 | 0 | 00000011 | 10101010 10111011 11001100
	// Concat: 0101_0_00000011_10101010_10111011_11001100
	// = 0101000000011101010101011101111001100 (37 bits)
	// Byte 0: 01010000 = 0x50
	// Byte 1: 00011101 = 0x1D
	// Actually byte 1 should be: 0_0000001_1 from align(0) + count(00000011)
	// Hmm, let me trace through more carefully.

	// tag = 5 = 0101 (bits 0-3)
	// align = 0 (bit 4)
	// count = 3 = 00000011 (bits 5-12)
	// data[0] = 0xAA = 10101010 (bits 13-20)
	// data[1] = 0xBB = 10111011 (bits 21-28)
	// data[2] = 0xCC = 11001100 (bits 29-36)
	//
	// Byte layout (MSB first):
	// Byte 0: bits 0-7 = 0101_0_000 = 0x50 (tag + align + 3 bits of count=000)
	// Byte 1: bits 8-15 = 00011_101 -- wait count is 00000011
	// If byte 0 has bits 0-7, and bits 5-7 are first 3 bits of count (000)
	// Then byte 1 has bits 8-15 = remaining 5 bits of count (00011) + first 3 bits of data[0] (101)
	// = 00011_101 = 0x1D
	// Byte 2: bits 16-23 = remaining 5 bits of data[0] (01010) + first 3 bits of data[1] (101)
	// = 01010_101 = 0x55
	// Byte 3: bits 24-31 = remaining 5 bits of data[1] (11011) + first 3 bits of data[2] (110)
	// = 11011_110 = 0xDE
	// Byte 4: bits 32-36 = remaining 5 bits of data[2] (01100) + padding
	// = 01100_xxx = 0x60 (if padding is 0)

	data = []byte{0x50, 0x1D, 0x55, 0xDE, 0x60}
	r = bits.NewReader(data)

	bytesRead := ParseDataStreamElement(r)

	if bytesRead != 3 {
		t.Errorf("bytesRead = %d, want 3", bytesRead)
	}
}

func TestParseDataStreamElement_Extended(t *testing.T) {
	// Test count == 255 case (extended count)
	// element_instance_tag: 4 bits = 0
	// data_byte_align_flag: 1 bit = 0
	// count: 8 bits = 255 (triggers extended count)
	// extra_count: 8 bits = 10 (total = 265 bytes)
	// We don't actually care about the data content for this test

	// Build bit stream:
	// tag(4) = 0000
	// align(1) = 0
	// count(8) = 11111111 (255)
	// extra(8) = 00001010 (10)
	// data(265*8) = zeros
	//
	// Byte 0: 0000_0_111 = 0x07
	// Byte 1: 11111_000 = 0xF8
	// Byte 2: 01010_xxx = 0x50 (with data starting)
	// Actually: 00001010 for extra, then data

	// Let me trace:
	// bits 0-3: 0000 (tag)
	// bit 4: 0 (align)
	// bits 5-12: 11111111 (count=255)
	// bits 13-20: 00001010 (extra=10)
	// bits 21+: data bytes

	// Byte 0: bits 0-7 = 0000_0_111 = 0x07
	// Byte 1: bits 8-15 = 11111_000 = 0xF8
	// Byte 2: bits 16-23 = 01010_xxx (extra bits 3-7 = 01010, then data starts)

	// Total bytes needed: 3 bytes header + 265 bytes data = 268 bytes
	// But we only care about the return value

	data := make([]byte, 300)
	data[0] = 0x07 // tag=0, align=0, count[0:2]=111
	data[1] = 0xF8 // count[3:7]=11111, extra[0:2]=000
	data[2] = 0x50 // extra[3:7]=01010, data[0][0:2]=000

	r := bits.NewReader(data)

	bytesRead := ParseDataStreamElement(r)

	// count = 255, extra = 10, total = 265
	if bytesRead != 265 {
		t.Errorf("bytesRead = %d, want 265", bytesRead)
	}
}

func TestParseDataStreamElement_ByteAligned(t *testing.T) {
	// Test with byte alignment enabled
	// element_instance_tag: 4 bits = 3
	// data_byte_align_flag: 1 bit = 1 (aligned)
	// count: 8 bits = 2
	// [byte_align to next boundary]
	// data_stream_byte[0-1]: 2 bytes

	// bits 0-3: 0011 (tag=3)
	// bit 4: 1 (align=true)
	// bits 5-12: 00000010 (count=2)
	// bits 13: padding to byte boundary (need 3 bits of padding to reach bit 16)
	// bits 16-23: data[0]
	// bits 24-31: data[1]

	// Byte 0: 0011_1_000 = 0x38
	// Byte 1: 00010_xxx = padding needed at bit 13
	// After reading count at bit 13, we byte_align
	// Current bit position is 13, next byte boundary is 16
	// So 3 bits of padding, then data starts at byte 2

	// Byte 0: 0011_1_000 = 0x38 (tag + align + count[0:2])
	// Byte 1: 00010_000 = 0x10 (count[3:7] + padding to byte boundary)
	// Byte 2: 10101010 = 0xAA (data[0])
	// Byte 3: 10111011 = 0xBB (data[1])

	data := []byte{0x38, 0x10, 0xAA, 0xBB}
	r := bits.NewReader(data)

	bytesRead := ParseDataStreamElement(r)

	if bytesRead != 2 {
		t.Errorf("bytesRead = %d, want 2", bytesRead)
	}
}
```

**Step 2.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: ParseDataStreamElement"

**Step 2.3: Write minimal implementation**

```go
// internal/syntax/dse.go
package syntax

import "github.com/llehouerou/go-aac/internal/bits"

// ParseDataStreamElement parses a Data Stream Element (DSE).
// DSE carries auxiliary data that is not part of the audio bitstream.
// The data is simply skipped after reading.
//
// Returns the number of data bytes in the element.
//
// Ported from: data_stream_element() in ~/dev/faad2/libfaad/syntax.c:1080-1107
func ParseDataStreamElement(r *bits.Reader) uint16 {
	// element_instance_tag (4 bits) - discarded
	_ = r.GetBits(LenTag)

	// data_byte_align_flag (1 bit)
	byteAligned := r.Get1Bit() == 1

	// count (8 bits)
	count := uint16(r.GetBits(8))

	// If count == 255, read extended count
	if count == 255 {
		count += uint16(r.GetBits(8))
	}

	// Byte align if requested
	if byteAligned {
		r.ByteAlign()
	}

	// Skip data_stream_bytes
	for i := uint16(0); i < count; i++ {
		r.GetBits(LenByte)
	}

	return count
}
```

**Step 2.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 2.5: Commit**

```bash
git add internal/syntax/dse.go internal/syntax/dse_test.go
git commit -m "feat(syntax): implement ParseDataStreamElement"
```

---

## Task 3: Create Excluded Channels Parser

**Files:**
- Create: `internal/syntax/fill.go` (start of file)
- Create: `internal/syntax/fill_test.go` (start of file)

**Step 3.1: Write the failing test for parseExcludedChannels**

```go
// internal/syntax/fill_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseExcludedChannels_SevenChannels(t *testing.T) {
	// excluded_channels format (Table 4.4.32):
	// - exclude_mask[0-6]: 7 x 1 bit
	// - additional_excluded_chns: 1 bit (0 = no more)
	//
	// Test case: 7 channels with mask 1010101, no additional
	// Binary: 1010101 0 = 0xAA

	data := []byte{0xAA}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseExcludedChannels(r, drc)

	if bytesRead != 1 {
		t.Errorf("bytesRead = %d, want 1", bytesRead)
	}

	// Check exclude mask
	expected := []uint8{1, 0, 1, 0, 1, 0, 1}
	for i := 0; i < 7; i++ {
		if drc.ExcludeMask[i] != expected[i] {
			t.Errorf("ExcludeMask[%d] = %d, want %d", i, drc.ExcludeMask[i], expected[i])
		}
	}
}

func TestParseExcludedChannels_Extended(t *testing.T) {
	// Test with additional excluded channels
	// - exclude_mask[0-6]: 1111111 (7 bits)
	// - additional_excluded_chns: 1 (continue)
	// - exclude_mask[7-13]: 0000000 (7 bits)
	// - additional_excluded_chns: 0 (stop)
	//
	// Binary: 1111111 1 0000000 0 = 0xFF 0x00

	data := []byte{0xFF, 0x00}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseExcludedChannels(r, drc)

	if bytesRead != 2 {
		t.Errorf("bytesRead = %d, want 2", bytesRead)
	}

	// First 7 channels excluded
	for i := 0; i < 7; i++ {
		if drc.ExcludeMask[i] != 1 {
			t.Errorf("ExcludeMask[%d] = %d, want 1", i, drc.ExcludeMask[i])
		}
	}

	// Next 7 channels not excluded
	for i := 7; i < 14; i++ {
		if drc.ExcludeMask[i] != 0 {
			t.Errorf("ExcludeMask[%d] = %d, want 0", i, drc.ExcludeMask[i])
		}
	}

	// Additional excluded flags
	if drc.AdditionalExcludedChns[0] != 1 {
		t.Errorf("AdditionalExcludedChns[0] = %d, want 1", drc.AdditionalExcludedChns[0])
	}
	if drc.AdditionalExcludedChns[1] != 0 {
		t.Errorf("AdditionalExcludedChns[1] = %d, want 0", drc.AdditionalExcludedChns[1])
	}
}
```

**Step 3.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: parseExcludedChannels"

**Step 3.3: Write minimal implementation**

```go
// internal/syntax/fill.go
package syntax

import "github.com/llehouerou/go-aac/internal/bits"

// parseExcludedChannels parses the excluded_channels() element for DRC.
// Returns the number of bytes consumed (for byte counting in DRC parsing).
//
// Ported from: excluded_channels() in ~/dev/faad2/libfaad/syntax.c:2367-2394
func parseExcludedChannels(r *bits.Reader, drc *DRCInfo) uint8 {
	var n uint8
	numExclChan := 7

	// Read first 7 exclude_mask bits
	for i := 0; i < 7; i++ {
		drc.ExcludeMask[i] = r.Get1Bit()
	}
	n++

	// Read additional excluded channels groups
	for {
		additionalBit := r.Get1Bit()
		drc.AdditionalExcludedChns[n-1] = additionalBit

		if additionalBit == 0 {
			break
		}

		// Check bounds
		if numExclChan >= MaxChannels-7 {
			return n
		}

		// Read next 7 exclude_mask bits
		for i := numExclChan; i < numExclChan+7; i++ {
			if i < MaxChannels {
				drc.ExcludeMask[i] = r.Get1Bit()
			}
		}
		n++
		numExclChan += 7
	}

	return n
}
```

**Step 3.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 3.5: Commit**

```bash
git add internal/syntax/fill.go internal/syntax/fill_test.go
git commit -m "feat(syntax): implement parseExcludedChannels for DRC"
```

---

## Task 4: Implement Dynamic Range Info Parser

**Files:**
- Modify: `internal/syntax/fill.go`
- Modify: `internal/syntax/fill_test.go`

**Step 4.1: Write the failing test for parseDynamicRangeInfo**

```go
// Add to internal/syntax/fill_test.go

func TestParseDynamicRangeInfo_Minimal(t *testing.T) {
	// Minimal DRC info:
	// - has_instance_tag: 1 bit = 0 (no instance tag)
	// - excluded_chns_present: 1 bit = 0 (no excluded channels)
	// - has_bands_data: 1 bit = 0 (no band data, single band)
	// - has_prog_ref_level: 1 bit = 0 (no program reference level)
	// - dyn_rng_sgn[0]: 1 bit = 0
	// - dyn_rng_ctl[0]: 7 bits = 0x55 (85)
	//
	// Binary: 0 0 0 0 0 1010101 = 0x00 0x55 (with padding)
	// Actually: 0000_0101_0101_xxxx = 0x05 0x5x

	data := []byte{0x05, 0x50}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseDynamicRangeInfo(r, drc)

	if bytesRead != 1 {
		t.Errorf("bytesRead = %d, want 1", bytesRead)
	}

	if drc.NumBands != 1 {
		t.Errorf("NumBands = %d, want 1", drc.NumBands)
	}

	if drc.DynRngSgn[0] != 0 {
		t.Errorf("DynRngSgn[0] = %d, want 0", drc.DynRngSgn[0])
	}

	if drc.DynRngCtl[0] != 0x55 {
		t.Errorf("DynRngCtl[0] = %d, want 85", drc.DynRngCtl[0])
	}
}

func TestParseDynamicRangeInfo_WithInstanceTag(t *testing.T) {
	// DRC with instance tag:
	// - has_instance_tag: 1 bit = 1
	// - pce_instance_tag: 4 bits = 0x5
	// - reserved: 4 bits = 0
	// - excluded_chns_present: 1 bit = 0
	// - has_bands_data: 1 bit = 0
	// - has_prog_ref_level: 1 bit = 0
	// - dyn_rng_sgn[0]: 1 bit = 1
	// - dyn_rng_ctl[0]: 7 bits = 0x7F (127)
	//
	// Binary: 1 0101 0000 0 0 0 1 1111111
	// = 1_0101_0000_0_0_0_1_1111111
	// Byte 0: 10101000 = 0xA8
	// Byte 1: 00001111 = 0x0F
	// Byte 2: 1111xxxx = 0xF0

	data := []byte{0xA8, 0x0F, 0xF0}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseDynamicRangeInfo(r, drc)

	if bytesRead != 2 {
		t.Errorf("bytesRead = %d, want 2", bytesRead)
	}

	if drc.PCEInstanceTag != 5 {
		t.Errorf("PCEInstanceTag = %d, want 5", drc.PCEInstanceTag)
	}

	if drc.DynRngSgn[0] != 1 {
		t.Errorf("DynRngSgn[0] = %d, want 1", drc.DynRngSgn[0])
	}

	if drc.DynRngCtl[0] != 0x7F {
		t.Errorf("DynRngCtl[0] = %d, want 127", drc.DynRngCtl[0])
	}
}

func TestParseDynamicRangeInfo_WithProgRefLevel(t *testing.T) {
	// DRC with program reference level:
	// - has_instance_tag: 1 bit = 0
	// - excluded_chns_present: 1 bit = 0
	// - has_bands_data: 1 bit = 0
	// - has_prog_ref_level: 1 bit = 1
	// - prog_ref_level: 7 bits = 0x40 (64)
	// - reserved: 1 bit = 0
	// - dyn_rng_sgn[0]: 1 bit = 0
	// - dyn_rng_ctl[0]: 7 bits = 0x20 (32)
	//
	// Binary: 0 0 0 1 1000000 0 0 0100000
	// = 0001_1000_0000_0010_0000
	// Byte 0: 00011000 = 0x18
	// Byte 1: 00000010 = 0x02
	// Byte 2: 00000xxx = 0x00

	data := []byte{0x18, 0x02, 0x00}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseDynamicRangeInfo(r, drc)

	if bytesRead != 2 {
		t.Errorf("bytesRead = %d, want 2", bytesRead)
	}

	if drc.ProgRefLevel != 64 {
		t.Errorf("ProgRefLevel = %d, want 64", drc.ProgRefLevel)
	}

	if drc.DynRngCtl[0] != 32 {
		t.Errorf("DynRngCtl[0] = %d, want 32", drc.DynRngCtl[0])
	}
}

func TestParseDynamicRangeInfo_MultiBand(t *testing.T) {
	// DRC with multiple bands:
	// - has_instance_tag: 1 bit = 0
	// - excluded_chns_present: 1 bit = 0
	// - has_bands_data: 1 bit = 1
	// - band_incr: 4 bits = 2 (3 bands total: 1 + 2)
	// - reserved: 4 bits = 0
	// - band_top[0]: 8 bits = 10
	// - band_top[1]: 8 bits = 20
	// - band_top[2]: 8 bits = 30
	// - has_prog_ref_level: 1 bit = 0
	// - dyn_rng_sgn[0]: 1 bit = 0, dyn_rng_ctl[0]: 7 bits = 10
	// - dyn_rng_sgn[1]: 1 bit = 0, dyn_rng_ctl[1]: 7 bits = 20
	// - dyn_rng_sgn[2]: 1 bit = 0, dyn_rng_ctl[2]: 7 bits = 30
	//
	// This is complex, let's build byte by byte:
	// Bits 0-2: 001 (has_instance=0, excluded=0, has_bands=1)
	// Bits 3-6: 0010 (band_incr=2)
	// Bits 7-10: 0000 (reserved)
	// Bits 11-18: 00001010 (band_top[0]=10)
	// Bits 19-26: 00010100 (band_top[1]=20)
	// Bits 27-34: 00011110 (band_top[2]=30)
	// Bit 35: 0 (has_prog_ref=0)
	// Bits 36-43: 0_0001010 (sgn[0]=0, ctl[0]=10)
	// Bits 44-51: 0_0010100 (sgn[1]=0, ctl[1]=20)
	// Bits 52-59: 0_0011110 (sgn[2]=0, ctl[2]=30)

	// Byte 0: 001_0010_0 = 0x24
	// Byte 1: 000_00001 = 0x01
	// Byte 2: 010_00010 = 0x42
	// Byte 3: 100_00011 = 0x83
	// Byte 4: 110_0_0001 = 0xC1
	// Byte 5: 010_0_0010 = 0x42
	// Byte 6: 100_0_0011 = 0x84
	// Byte 7: 110_xxxxx = 0xC0

	data := []byte{0x24, 0x01, 0x42, 0x83, 0xC1, 0x42, 0x84, 0xC0}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseDynamicRangeInfo(r, drc)

	// 1 base + 1 (bands header) + 3 (band_top) + 3 (dyn_rng) = 8 bytes
	// Wait, FAAD2 counts bytes differently. Let me check the expected value.
	// Looking at syntax.c, n counts "logical bytes" not actual bytes consumed.
	// Actually it counts: 1 (base) + 1 if instance_tag + n from excluded_channels
	// + 1 if bands_data (header) + num_bands (band_top) + 1 if prog_ref
	// + num_bands (dyn_rng)
	// = 1 + 0 + 0 + 1 + 3 + 0 + 3 = 8

	// Actually, looking more carefully:
	// n starts at 1
	// If has_instance_tag: n++
	// If excluded_chns: n += excluded_channels()
	// If has_bands_data: n++ (for header), then n += num_bands (for band_top)
	// If has_prog_ref: n++
	// Then n += num_bands (for dyn_rng)
	//
	// So: n=1, bands_data: n++ (n=2), band_top: n+=3 (n=5), dyn_rng: n+=3 (n=8)
	// Expected: 8

	if bytesRead != 8 {
		t.Errorf("bytesRead = %d, want 8", bytesRead)
	}

	if drc.NumBands != 3 {
		t.Errorf("NumBands = %d, want 3", drc.NumBands)
	}

	if drc.BandTop[0] != 10 || drc.BandTop[1] != 20 || drc.BandTop[2] != 30 {
		t.Errorf("BandTop = %v, want [10, 20, 30]", drc.BandTop[:3])
	}

	if drc.DynRngCtl[0] != 10 || drc.DynRngCtl[1] != 20 || drc.DynRngCtl[2] != 30 {
		t.Errorf("DynRngCtl = %v, want [10, 20, 30]", drc.DynRngCtl[:3])
	}
}
```

**Step 4.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: parseDynamicRangeInfo"

**Step 4.3: Write minimal implementation**

```go
// Add to internal/syntax/fill.go

// parseDynamicRangeInfo parses the dynamic_range_info() element.
// Returns the number of "logical bytes" consumed (for extension_payload counting).
//
// Ported from: dynamic_range_info() in ~/dev/faad2/libfaad/syntax.c:2302-2364
func parseDynamicRangeInfo(r *bits.Reader, drc *DRCInfo) uint8 {
	var n uint8 = 1

	drc.NumBands = 1

	// has_instance_tag (1 bit)
	if r.Get1Bit() == 1 {
		drc.PCEInstanceTag = uint8(r.GetBits(4))
		_ = r.GetBits(4) // drc_tag_reserved_bits
		n++
	}

	// excluded_chns_present (1 bit)
	drc.ExcludedChnsPresent = r.Get1Bit() == 1
	if drc.ExcludedChnsPresent {
		n += parseExcludedChannels(r, drc)
	}

	// has_bands_data (1 bit)
	if r.Get1Bit() == 1 {
		bandIncr := uint8(r.GetBits(4))
		_ = r.GetBits(4) // drc_bands_reserved_bits
		n++
		drc.NumBands += bandIncr

		for i := uint8(0); i < drc.NumBands; i++ {
			drc.BandTop[i] = uint8(r.GetBits(8))
			n++
		}
	}

	// has_prog_ref_level (1 bit)
	if r.Get1Bit() == 1 {
		drc.ProgRefLevel = uint8(r.GetBits(7))
		_ = r.Get1Bit() // prog_ref_level_reserved_bits
		n++
	}

	// Read dynamic range data for each band
	for i := uint8(0); i < drc.NumBands; i++ {
		drc.DynRngSgn[i] = r.Get1Bit()
		drc.DynRngCtl[i] = uint8(r.GetBits(7))
		n++
	}

	return n
}
```

**Step 4.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 4.5: Commit**

```bash
git add internal/syntax/fill.go internal/syntax/fill_test.go
git commit -m "feat(syntax): implement parseDynamicRangeInfo"
```

---

## Task 5: Implement Extension Payload Parser

**Files:**
- Modify: `internal/syntax/fill.go`
- Modify: `internal/syntax/fill_test.go`

**Step 5.1: Write the failing test for parseExtensionPayload**

```go
// Add to internal/syntax/fill_test.go

func TestParseExtensionPayload_DynamicRange(t *testing.T) {
	// Extension type: EXT_DYNAMIC_RANGE (11)
	// Followed by minimal DRC data
	//
	// extension_type: 4 bits = 1011 (11)
	// Then dynamic_range_info:
	// - has_instance_tag: 0
	// - excluded_chns_present: 0
	// - has_bands_data: 0
	// - has_prog_ref_level: 0
	// - dyn_rng_sgn: 0
	// - dyn_rng_ctl: 7 bits = 0x55
	//
	// Binary: 1011 0 0 0 0 0 1010101
	// = 1011_0000_0101_0101
	// Byte 0: 10110000 = 0xB0
	// Byte 1: 01010101 = 0x55

	data := []byte{0xB0, 0x55}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseExtensionPayload(r, drc, 2)

	// dynamic_range_info returns 1, so extension_payload returns 1
	if bytesRead != 1 {
		t.Errorf("bytesRead = %d, want 1", bytesRead)
	}

	if !drc.Present {
		t.Error("drc.Present should be true")
	}

	if drc.DynRngCtl[0] != 0x55 {
		t.Errorf("DynRngCtl[0] = %d, want 85", drc.DynRngCtl[0])
	}
}

func TestParseExtensionPayload_FillData(t *testing.T) {
	// Extension type: EXT_FILL_DATA (1)
	// Followed by fill_nibble (4 bits = 0000) + fill_bytes
	//
	// extension_type: 4 bits = 0001
	// fill_nibble: 4 bits = 0000
	// fill_byte[0]: 8 bits = 0xA5 (should be 0xA5 per spec)
	// fill_byte[1]: 8 bits = 0xA5
	// (count = 3 means 2 fill bytes after the nibble)
	//
	// Binary: 0001 0000 10100101 10100101
	// Byte 0: 00010000 = 0x10
	// Byte 1: 10100101 = 0xA5
	// Byte 2: 10100101 = 0xA5

	data := []byte{0x10, 0xA5, 0xA5}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseExtensionPayload(r, drc, 3)

	// EXT_FILL_DATA returns count
	if bytesRead != 3 {
		t.Errorf("bytesRead = %d, want 3", bytesRead)
	}
}

func TestParseExtensionPayload_Filler(t *testing.T) {
	// Extension type: EXT_FIL (0)
	// Just reads fill_nibble (4 bits) + remaining bytes
	//
	// extension_type: 4 bits = 0000
	// fill_nibble: 4 bits (align=4)
	// other_bits: 8 bits per remaining byte
	//
	// count = 2 means 1 byte of fill after nibble
	//
	// Byte 0: 00000000 = 0x00
	// Byte 1: 11111111 = 0xFF

	data := []byte{0x00, 0xFF}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseExtensionPayload(r, drc, 2)

	if bytesRead != 2 {
		t.Errorf("bytesRead = %d, want 2", bytesRead)
	}
}

func TestParseExtensionPayload_DataElement(t *testing.T) {
	// Extension type: EXT_DATA_ELEMENT (2)
	// data_element_version: 4 bits = 0 (ANC_DATA)
	// dataElementLengthPart: 8 bits = 5 (length)
	// data_element_byte[0-4]: 5 bytes
	//
	// extension_type: 4 bits = 0010
	// data_element_version: 4 bits = 0000
	// dataElementLengthPart: 8 bits = 00000101 (5)
	// data: 5 bytes
	//
	// Byte 0: 00100000 = 0x20
	// Byte 1: 00000101 = 0x05
	// Bytes 2-6: data

	data := []byte{0x20, 0x05, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	bytesRead := parseExtensionPayload(r, drc, 10)

	// dataElementLength=5, loopCounter=1, +1 = 7
	if bytesRead != 7 {
		t.Errorf("bytesRead = %d, want 7", bytesRead)
	}
}
```

**Step 5.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: parseExtensionPayload"

**Step 5.3: Write minimal implementation**

```go
// Add to internal/syntax/fill.go

// parseExtensionPayload parses an extension_payload() element.
// Returns the number of payload bytes consumed.
//
// Ported from: extension_payload() in ~/dev/faad2/libfaad/syntax.c:2240-2299
func parseExtensionPayload(r *bits.Reader, drc *DRCInfo, count uint16) uint16 {
	var align uint8 = 4

	extensionType := ExtensionType(r.GetBits(4))

	switch extensionType {
	case ExtDynamicRange:
		drc.Present = true
		n := parseDynamicRangeInfo(r, drc)
		return uint16(n)

	case ExtFillData:
		// fill_nibble (must be 0000)
		_ = r.GetBits(4)
		// fill_byte (must be 0xA5 "10100101")
		for i := uint16(0); i < count-1; i++ {
			_ = r.GetBits(8)
		}
		return count

	case ExtDataElement:
		dataElementVersion := r.GetBits(4)
		switch dataElementVersion {
		case AncData:
			loopCounter := uint16(0)
			dataElementLength := uint16(0)
			for {
				dataElementLengthPart := uint8(r.GetBits(8))
				dataElementLength += uint16(dataElementLengthPart)
				loopCounter++
				if dataElementLengthPart != 255 {
					break
				}
			}
			// Read data_element_bytes
			for i := uint16(0); i < dataElementLength; i++ {
				_ = r.GetBits(8)
				// Note: FAAD2 returns early here after first byte, which seems like a bug
				// We'll follow the same behavior for compatibility
				return dataElementLength + loopCounter + 1
			}
			// If dataElementLength is 0
			return loopCounter + 1
		default:
			align = 0
		}
		fallthrough

	case ExtFil:
		fallthrough

	default:
		// Read fill_nibble or align bits
		r.GetBits(uint(align))
		// Read remaining bytes
		for i := uint16(0); i < count-1; i++ {
			_ = r.GetBits(8)
		}
		return count
	}
}
```

**Step 5.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 5.5: Commit**

```bash
git add internal/syntax/fill.go internal/syntax/fill_test.go
git commit -m "feat(syntax): implement parseExtensionPayload"
```

---

## Task 6: Implement Fill Element Parser

**Files:**
- Modify: `internal/syntax/fill.go`
- Modify: `internal/syntax/fill_test.go`

**Step 6.1: Write the failing test for ParseFillElement**

```go
// Add to internal/syntax/fill_test.go

func TestParseFillElement_Empty(t *testing.T) {
	// count = 0, no payload
	// count: 4 bits = 0000
	//
	// Byte 0: 0000xxxx = 0x00

	data := []byte{0x00}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	err := ParseFillElement(r, drc)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseFillElement_SmallCount(t *testing.T) {
	// count = 3 (small, no extension)
	// count: 4 bits = 0011
	// Followed by 3 bytes of extension_payload (EXT_FIL with fill data)
	//
	// count: 0011
	// ext_type: 0000 (EXT_FIL)
	// fill_nibble: 0000
	// fill_bytes: 2 bytes
	//
	// Byte 0: 0011_0000 = 0x30
	// Byte 1: 0000_1111 = 0x0F
	// Byte 2: 11111111 = 0xFF

	data := []byte{0x30, 0x0F, 0xFF}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	err := ParseFillElement(r, drc)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseFillElement_ExtendedCount(t *testing.T) {
	// count = 15 (triggers extended count)
	// count: 4 bits = 1111 (15)
	// extra_count: 8 bits = 5 (total = 15 + 5 - 1 = 19 bytes)
	// Followed by 19 bytes of extension_payload
	//
	// count: 1111
	// extra: 00000101 (5)
	// ext_type: 0000 (EXT_FIL)
	// fill_nibble: 0000
	// fill_bytes: 18 bytes
	//
	// Byte 0: 1111_0000 = 0xF0
	// Byte 1: 0101_0000 = 0x50
	// Byte 2: 0000_xxxx = fill starts
	// ... more fill bytes

	data := make([]byte, 25)
	data[0] = 0xF0 // count=15
	data[1] = 0x50 // extra=5, ext_type=0 start
	data[2] = 0x00 // ext_type=0 end, fill_nibble
	// Rest are fill bytes

	r := bits.NewReader(data)
	drc := &DRCInfo{}

	err := ParseFillElement(r, drc)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseFillElement_WithDRC(t *testing.T) {
	// Fill element containing DRC extension
	// count: 4 bits = 3
	// ext_type: 4 bits = 1011 (EXT_DYNAMIC_RANGE = 11)
	// dynamic_range_info (minimal)
	//
	// count: 0011
	// ext_type: 1011
	// has_instance_tag: 0
	// excluded_chns_present: 0
	// has_bands_data: 0
	// has_prog_ref_level: 0
	// dyn_rng_sgn: 0
	// dyn_rng_ctl: 1010101 (0x55)
	//
	// Binary: 0011 1011 0 0 0 0 0 1010101
	// Byte 0: 0011_1011 = 0x3B
	// Byte 1: 0000_0101 = 0x05
	// Byte 2: 0101_xxxx = 0x50

	data := []byte{0x3B, 0x05, 0x50}
	r := bits.NewReader(data)
	drc := &DRCInfo{}

	err := ParseFillElement(r, drc)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !drc.Present {
		t.Error("drc.Present should be true")
	}

	if drc.DynRngCtl[0] != 0x55 {
		t.Errorf("DynRngCtl[0] = %d, want 85", drc.DynRngCtl[0])
	}
}
```

**Step 6.2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "undefined: ParseFillElement"

**Step 6.3: Write minimal implementation**

```go
// Add to internal/syntax/fill.go

// FillConfig holds configuration for Fill Element parsing.
// Currently empty, but structured for future SBR support.
type FillConfig struct {
	// SBRElement specifies which SBR element to associate with.
	// Set to InvalidSBRElement (255) if no SBR association.
	SBRElement uint8
}

// ParseFillElement parses a Fill Element (ID_FIL).
// Fill elements contain extension payloads including DRC and SBR data.
//
// For now, SBR data is skipped. SBR support is implemented in Phase 8.
//
// Ported from: fill_element() in ~/dev/faad2/libfaad/syntax.c:1110-1197
func ParseFillElement(r *bits.Reader, drc *DRCInfo) error {
	return ParseFillElementWithConfig(r, drc, &FillConfig{SBRElement: InvalidSBRElement})
}

// ParseFillElementWithConfig parses a Fill Element with explicit configuration.
// This variant allows specifying SBR element association for future SBR support.
//
// Ported from: fill_element() in ~/dev/faad2/libfaad/syntax.c:1110-1197
func ParseFillElementWithConfig(r *bits.Reader, drc *DRCInfo, cfg *FillConfig) error {
	// count (4 bits)
	count := uint16(r.GetBits(4))

	// If count == 15, read extended count
	if count == 15 {
		count += uint16(r.GetBits(8)) - 1
	}

	if count == 0 {
		return nil
	}

	// Check for SBR extension (Phase 8)
	// For now, just peek and skip SBR data
	bsExtensionType := ExtensionType(r.ShowBits(4))

	if bsExtensionType == ExtSBRData || bsExtensionType == ExtSBRDataCRC {
		// SBR data - skip for now (Phase 8 implementation)
		// Just consume all the count bytes
		for i := uint16(0); i < count; i++ {
			r.GetBits(8)
		}
		return nil
	}

	// Parse extension payloads until count is exhausted
	for count > 0 {
		payloadBytes := parseExtensionPayload(r, drc, count)
		if payloadBytes <= count {
			count -= payloadBytes
		} else {
			count = 0
		}
	}

	return nil
}
```

**Step 6.4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 6.5: Commit**

```bash
git add internal/syntax/fill.go internal/syntax/fill_test.go
git commit -m "feat(syntax): implement ParseFillElement"
```

---

## Task 7: Add Documentation and Final Tests

**Files:**
- Modify: `internal/syntax/fill.go` (add package documentation)
- Modify: `internal/syntax/dse.go` (add package documentation)

**Step 7.1: Add file documentation to fill.go**

Update the file header:

```go
// internal/syntax/fill.go
//
// Fill Element and Extension Payload Parsing
//
// This file implements:
// - ParseFillElement: Parses ID_FIL elements
// - parseExtensionPayload: Dispatches to extension-specific parsers
// - parseDynamicRangeInfo: Parses DRC (Dynamic Range Control) data
// - parseExcludedChannels: Parses excluded channel masks for DRC
//
// Fill elements can contain:
// - EXT_DYNAMIC_RANGE (11): Dynamic Range Control data
// - EXT_FILL_DATA (1): Filler bytes (0xA5 pattern)
// - EXT_DATA_ELEMENT (2): Ancillary data
// - EXT_FIL (0): Generic filler
// - EXT_SBR_DATA (13): SBR data (handled in Phase 8)
// - EXT_SBR_DATA_CRC (14): SBR data with CRC (handled in Phase 8)
//
// Ported from: ~/dev/faad2/libfaad/syntax.c:1110-1197, 2240-2394
package syntax
```

**Step 7.2: Add file documentation to dse.go**

Update the file header:

```go
// internal/syntax/dse.go
//
// Data Stream Element Parsing
//
// This file implements:
// - ParseDataStreamElement: Parses ID_DSE elements
//
// Data Stream Elements carry auxiliary data that is not part of the
// audio bitstream. This data is typically discarded during decoding.
//
// Ported from: ~/dev/faad2/libfaad/syntax.c:1080-1107
package syntax
```

**Step 7.3: Run all tests**

Run: `make check`
Expected: PASS

**Step 7.4: Commit**

```bash
git add internal/syntax/fill.go internal/syntax/dse.go
git commit -m "docs(syntax): add documentation for fill and dse parsers"
```

---

## Summary

This plan implements:

1. **SBR extension type constants** - Added for future Phase 8 SBR support
2. **ParseDataStreamElement** - Parses ID_DSE elements (auxiliary data)
3. **parseExcludedChannels** - Internal DRC helper for channel exclusion masks
4. **parseDynamicRangeInfo** - Parses DRC data into existing DRCInfo struct
5. **parseExtensionPayload** - Extension type dispatcher
6. **ParseFillElement** - Main entry point for ID_FIL elements

**Total new files:** 4 (fill.go, fill_test.go, dse.go, dse_test.go)
**Total new lines:** ~350

**Testing approach:**
- Unit tests with hand-crafted bit patterns matching FAAD2 parsing behavior
- Each function tested in isolation before integration
- TDD cycle followed throughout

---

**Plan complete and saved to `docs/plans/2025-12-28-fill-element-parser.md`. Two execution options:**

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
