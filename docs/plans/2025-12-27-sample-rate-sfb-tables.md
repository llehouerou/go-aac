# Sample Rate and Scalefactor Band Tables Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement sample rate lookup tables and scalefactor band (SFB) offset tables needed for AAC decoding window grouping.

**Architecture:** Create 4 files in `internal/tables/` package: `sample_rates.go` for sample rate lookup functions, `sfb_long.go` for long window SFB tables, `sfb_short.go` for short window SFB tables, and `sfb.go` for unified lookup functions. All values are copied exactly from FAAD2.

**Tech Stack:** Go, TDD, no external dependencies

---

## Source Analysis

### FAAD2 Sources
- `~/dev/faad2/libfaad/common.c:41-121` - Functions: `get_sr_index()`, `get_sample_rate()`, `max_pred_sfb()`, `max_tns_sfb()`, `can_decode_ot()`
- `~/dev/faad2/libfaad/specrec.c:66-285` - SFB offset tables and lookup arrays

### Tables to Port
| Table | Size | Purpose |
|-------|------|---------|
| `sample_rates[]` | 12 entries | Sample rate lookup by index |
| `pred_sfb_max[]` | 12 entries | Max prediction SFB per sample rate |
| `tns_sbf_max[][]` | 16x4 entries | Max TNS SFB by sample rate and window type |
| `num_swb_1024_window[]` | 12 entries | Number of SWBs for 1024-sample long windows |
| `num_swb_960_window[]` | 12 entries | Number of SWBs for 960-sample long windows |
| `num_swb_128_window[]` | 12 entries | Number of SWBs for 128-sample short windows |
| `swb_offset_1024_*[]` | 7 tables | SWB offsets for long windows (by sample rate group) |
| `swb_offset_128_*[]` | 6 tables | SWB offsets for short windows (by sample rate group) |

### Functions to Port
| Function | Purpose |
|----------|---------|
| `get_sr_index(samplerate)` | Get sample rate index from sample rate value |
| `get_sample_rate(sr_index)` | Get sample rate value from index |
| `max_pred_sfb(sr_index)` | Get max prediction SFB for sample rate |
| `max_tns_sfb(sr_index, object_type, is_short)` | Get max TNS SFB |
| `can_decode_ot(object_type)` | Check if object type is supported |

---

## Task 1: Sample Rate Lookup

**Files:**
- Create: `internal/tables/sample_rates.go`
- Test: `internal/tables/sample_rates_test.go`

### Step 1.1: Write failing test for GetSampleRate

```go
// internal/tables/sample_rates_test.go
package tables

import "testing"

func TestGetSampleRate(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/common.c:59-71
	tests := []struct {
		index    uint8
		expected uint32
	}{
		{0, 96000},
		{1, 88200},
		{2, 64000},
		{3, 48000},
		{4, 44100},
		{5, 32000},
		{6, 24000},
		{7, 22050},
		{8, 16000},
		{9, 12000},
		{10, 11025},
		{11, 8000},
		{12, 0}, // Invalid index
		{15, 0}, // Invalid index
	}

	for _, tt := range tests {
		got := GetSampleRate(tt.index)
		if got != tt.expected {
			t.Errorf("GetSampleRate(%d) = %d, want %d", tt.index, got, tt.expected)
		}
	}
}
```

### Step 1.2: Run test to verify it fails

Run: `make test PKG=./internal/tables`
Expected: FAIL with "undefined: GetSampleRate"

### Step 1.3: Write minimal implementation

```go
// internal/tables/sample_rates.go
package tables

// SampleRates maps sample rate index to actual sample rate in Hz.
// Source: ~/dev/faad2/libfaad/common.c:61-65
var SampleRates = [12]uint32{
	96000, 88200, 64000, 48000, 44100, 32000,
	24000, 22050, 16000, 12000, 11025, 8000,
}

// GetSampleRate returns the sample rate for a given index.
// Returns 0 for invalid indices (>= 12).
// Source: ~/dev/faad2/libfaad/common.c:59-71
func GetSampleRate(srIndex uint8) uint32 {
	if srIndex >= 12 {
		return 0
	}
	return SampleRates[srIndex]
}
```

### Step 1.4: Run test to verify it passes

Run: `make test PKG=./internal/tables`
Expected: PASS

### Step 1.5: Write failing test for GetSRIndex

```go
// Add to internal/tables/sample_rates_test.go
func TestGetSRIndex(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/common.c:41-56
	// Uses threshold-based matching, not exact lookup
	tests := []struct {
		sampleRate uint32
		expected   uint8
	}{
		{96000, 0},
		{92017, 0},  // Threshold for index 0
		{92016, 1},  // Just below threshold
		{88200, 1},
		{75132, 1},  // Threshold for index 1
		{75131, 2},  // Just below threshold
		{64000, 2},
		{48000, 3},
		{44100, 4},
		{32000, 5},
		{24000, 6},
		{22050, 7},
		{16000, 8},
		{12000, 9},
		{11025, 10},
		{8000, 11},
		{7350, 11},  // Below 8000, still returns 11
		{100, 11},   // Any very low rate returns 11
	}

	for _, tt := range tests {
		got := GetSRIndex(tt.sampleRate)
		if got != tt.expected {
			t.Errorf("GetSRIndex(%d) = %d, want %d", tt.sampleRate, got, tt.expected)
		}
	}
}
```

### Step 1.6: Run test to verify it fails

Run: `make test PKG=./internal/tables`
Expected: FAIL with "undefined: GetSRIndex"

### Step 1.7: Write minimal implementation

```go
// Add to internal/tables/sample_rates.go

// GetSRIndex returns the sample rate index for a given sample rate.
// Uses threshold-based matching as defined in MPEG-4 standard.
// Source: ~/dev/faad2/libfaad/common.c:41-56
func GetSRIndex(sampleRate uint32) uint8 {
	if sampleRate >= 92017 {
		return 0
	}
	if sampleRate >= 75132 {
		return 1
	}
	if sampleRate >= 55426 {
		return 2
	}
	if sampleRate >= 46009 {
		return 3
	}
	if sampleRate >= 37566 {
		return 4
	}
	if sampleRate >= 27713 {
		return 5
	}
	if sampleRate >= 23004 {
		return 6
	}
	if sampleRate >= 18783 {
		return 7
	}
	if sampleRate >= 13856 {
		return 8
	}
	if sampleRate >= 11502 {
		return 9
	}
	if sampleRate >= 9391 {
		return 10
	}
	return 11
}
```

### Step 1.8: Run test to verify it passes

Run: `make test PKG=./internal/tables`
Expected: PASS

### Step 1.9: Commit

```bash
git add internal/tables/sample_rates.go internal/tables/sample_rates_test.go
git commit -m "feat(tables): add sample rate lookup functions

- GetSampleRate(index) returns sample rate for index
- GetSRIndex(sampleRate) returns index using threshold matching

Ported from: ~/dev/faad2/libfaad/common.c:41-71

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Prediction and TNS Limits

**Files:**
- Modify: `internal/tables/sample_rates.go`
- Modify: `internal/tables/sample_rates_test.go`

### Step 2.1: Write failing test for MaxPredSFB

```go
// Add to internal/tables/sample_rates_test.go
func TestMaxPredSFB(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/common.c:73-85
	expected := [12]uint8{33, 33, 38, 40, 40, 40, 41, 41, 37, 37, 37, 34}

	for i := uint8(0); i < 12; i++ {
		got := MaxPredSFB(i)
		if got != expected[i] {
			t.Errorf("MaxPredSFB(%d) = %d, want %d", i, got, expected[i])
		}
	}

	// Test invalid index
	if got := MaxPredSFB(12); got != 0 {
		t.Errorf("MaxPredSFB(12) = %d, want 0", got)
	}
}
```

### Step 2.2: Run test to verify it fails

Run: `make test PKG=./internal/tables`
Expected: FAIL with "undefined: MaxPredSFB"

### Step 2.3: Write minimal implementation

```go
// Add to internal/tables/sample_rates.go

// predSFBMax contains max prediction SFB per sample rate index.
// Source: ~/dev/faad2/libfaad/common.c:75-78
var predSFBMax = [12]uint8{
	33, 33, 38, 40, 40, 40, 41, 41, 37, 37, 37, 34,
}

// MaxPredSFB returns the maximum prediction scalefactor band for a sample rate index.
// Returns 0 for invalid indices.
// Source: ~/dev/faad2/libfaad/common.c:73-85
func MaxPredSFB(srIndex uint8) uint8 {
	if srIndex >= 12 {
		return 0
	}
	return predSFBMax[srIndex]
}
```

### Step 2.4: Run test to verify it passes

Run: `make test PKG=./internal/tables`
Expected: PASS

### Step 2.5: Write failing test for MaxTNSSFB

```go
// Add to internal/tables/sample_rates_test.go
import "github.com/user/go-aac"

func TestMaxTNSSFB(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/common.c:87-121
	// Table columns: [Main/LC long, Main/LC short, SSR long, SSR short]
	tests := []struct {
		srIndex    uint8
		objectType aac.ObjectType
		isShort    bool
		expected   uint8
	}{
		// 96000 Hz
		{0, aac.ObjectTypeLC, false, 31},
		{0, aac.ObjectTypeLC, true, 9},
		{0, aac.ObjectTypeSSR, false, 28},
		{0, aac.ObjectTypeSSR, true, 7},
		// 48000 Hz
		{3, aac.ObjectTypeLC, false, 40},
		{3, aac.ObjectTypeLC, true, 14},
		{3, aac.ObjectTypeSSR, false, 26},
		{3, aac.ObjectTypeSSR, true, 6},
		// 44100 Hz
		{4, aac.ObjectTypeLC, false, 42},
		{4, aac.ObjectTypeLC, true, 14},
		// 8000 Hz
		{11, aac.ObjectTypeLC, false, 39},
		{11, aac.ObjectTypeLC, true, 14},
		// Invalid index returns 0
		{15, aac.ObjectTypeLC, false, 0},
	}

	for _, tt := range tests {
		got := MaxTNSSFB(tt.srIndex, tt.objectType, tt.isShort)
		if got != tt.expected {
			t.Errorf("MaxTNSSFB(%d, %d, %v) = %d, want %d",
				tt.srIndex, tt.objectType, tt.isShort, got, tt.expected)
		}
	}
}
```

### Step 2.6: Run test to verify it fails

Run: `make test PKG=./internal/tables`
Expected: FAIL with "undefined: MaxTNSSFB"

### Step 2.7: Write minimal implementation

```go
// Add to internal/tables/sample_rates.go
import "github.com/user/go-aac"

// tnsSFBMax contains max TNS SFB values.
// Columns: [Main/LC long, Main/LC short, SSR long, SSR short]
// Source: ~/dev/faad2/libfaad/common.c:96-114
var tnsSFBMax = [16][4]uint8{
	{31, 9, 28, 7},  // 96000
	{31, 9, 28, 7},  // 88200
	{34, 10, 27, 7}, // 64000
	{40, 14, 26, 6}, // 48000
	{42, 14, 26, 6}, // 44100
	{51, 14, 26, 6}, // 32000
	{46, 14, 29, 7}, // 24000
	{46, 14, 29, 7}, // 22050
	{42, 14, 23, 8}, // 16000
	{42, 14, 23, 8}, // 12000
	{42, 14, 23, 8}, // 11025
	{39, 14, 19, 7}, // 8000
	{39, 14, 19, 7}, // 7350
	{0, 0, 0, 0},
	{0, 0, 0, 0},
	{0, 0, 0, 0},
}

// MaxTNSSFB returns the maximum TNS scalefactor band.
// Source: ~/dev/faad2/libfaad/common.c:87-121
func MaxTNSSFB(srIndex uint8, objectType aac.ObjectType, isShort bool) uint8 {
	if srIndex >= 16 {
		return 0
	}

	i := 0
	if isShort {
		i = 1
	}
	if objectType == aac.ObjectTypeSSR {
		i += 2
	}

	return tnsSFBMax[srIndex][i]
}
```

### Step 2.8: Run test to verify it passes

Run: `make test PKG=./internal/tables`
Expected: PASS

### Step 2.9: Write failing test for CanDecodeOT

```go
// Add to internal/tables/sample_rates_test.go
func TestCanDecodeOT(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/common.c:124-172
	// Note: We support LC, MAIN, LTP. SSR is not supported.
	tests := []struct {
		objectType aac.ObjectType
		canDecode  bool
	}{
		{aac.ObjectTypeLC, true},
		{aac.ObjectTypeMain, true},
		{aac.ObjectTypeLTP, true},
		{aac.ObjectTypeSSR, false},  // Not supported
		{aac.ObjectTypeHEAAC, false}, // SBR handled separately
		{aac.ObjectTypeERLC, true},
		{aac.ObjectTypeERLTP, true},
		{aac.ObjectTypeLD, true},
		{aac.ObjectTypeDRMERLC, true},
		{100, false}, // Unknown type
	}

	for _, tt := range tests {
		got := CanDecodeOT(tt.objectType)
		if got != tt.canDecode {
			t.Errorf("CanDecodeOT(%d) = %v, want %v", tt.objectType, got, tt.canDecode)
		}
	}
}
```

### Step 2.10: Run test to verify it fails

Run: `make test PKG=./internal/tables`
Expected: FAIL with "undefined: CanDecodeOT"

### Step 2.11: Write minimal implementation

```go
// Add to internal/tables/sample_rates.go

// CanDecodeOT returns true if the object type can be decoded.
// Source: ~/dev/faad2/libfaad/common.c:124-172
func CanDecodeOT(objectType aac.ObjectType) bool {
	switch objectType {
	case aac.ObjectTypeLC:
		return true
	case aac.ObjectTypeMain:
		return true
	case aac.ObjectTypeLTP:
		return true
	case aac.ObjectTypeSSR:
		return false // SSR not supported
	case aac.ObjectTypeERLC:
		return true
	case aac.ObjectTypeERLTP:
		return true
	case aac.ObjectTypeLD:
		return true
	case aac.ObjectTypeDRMERLC:
		return true
	default:
		return false
	}
}
```

### Step 2.12: Run test to verify it passes

Run: `make test PKG=./internal/tables`
Expected: PASS

### Step 2.13: Commit

```bash
git add internal/tables/sample_rates.go internal/tables/sample_rates_test.go
git commit -m "feat(tables): add prediction and TNS limit functions

- MaxPredSFB(srIndex) returns max prediction SFB
- MaxTNSSFB(srIndex, objectType, isShort) returns max TNS SFB
- CanDecodeOT(objectType) checks if object type is decodable

Ported from: ~/dev/faad2/libfaad/common.c:73-172

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Long Window SFB Tables

**Files:**
- Create: `internal/tables/sfb_long.go`
- Create: `internal/tables/sfb_long_test.go`

### Step 3.1: Write failing test for SFB count tables

```go
// internal/tables/sfb_long_test.go
package tables

import "testing"

func TestNumSWB1024Window(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/specrec.c:81-84
	expected := [12]uint8{41, 41, 47, 49, 49, 51, 47, 47, 43, 43, 43, 40}

	for i := 0; i < 12; i++ {
		if NumSWB1024Window[i] != expected[i] {
			t.Errorf("NumSWB1024Window[%d] = %d, want %d", i, NumSWB1024Window[i], expected[i])
		}
	}
}

func TestNumSWB960Window(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/specrec.c:76-79
	expected := [12]uint8{40, 40, 45, 49, 49, 49, 46, 46, 42, 42, 42, 40}

	for i := 0; i < 12; i++ {
		if NumSWB960Window[i] != expected[i] {
			t.Errorf("NumSWB960Window[%d] = %d, want %d", i, NumSWB960Window[i], expected[i])
		}
	}
}
```

### Step 3.2: Run test to verify it fails

Run: `make test PKG=./internal/tables`
Expected: FAIL with "undefined: NumSWB1024Window"

### Step 3.3: Write minimal implementation for counts

```go
// internal/tables/sfb_long.go
package tables

// NumSWB1024Window contains the number of scale factor window bands
// for 1024-sample long windows at each sample rate index.
// Source: ~/dev/faad2/libfaad/specrec.c:81-84
var NumSWB1024Window = [12]uint8{
	41, 41, 47, 49, 49, 51, 47, 47, 43, 43, 43, 40,
}

// NumSWB960Window contains the number of scale factor window bands
// for 960-sample long windows at each sample rate index.
// Source: ~/dev/faad2/libfaad/specrec.c:76-79
var NumSWB960Window = [12]uint8{
	40, 40, 45, 49, 49, 49, 46, 46, 42, 42, 42, 40,
}
```

### Step 3.4: Run test to verify it passes

Run: `make test PKG=./internal/tables`
Expected: PASS

### Step 3.5: Write failing test for SFB offset tables

```go
// Add to internal/tables/sfb_long_test.go
func TestSWBOffset1024_96(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/specrec.c:91-96
	expected := []uint16{
		0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56,
		64, 72, 80, 88, 96, 108, 120, 132, 144, 156, 172, 188, 212, 240,
		276, 320, 384, 448, 512, 576, 640, 704, 768, 832, 896, 960, 1024,
	}

	if len(SWBOffset1024_96) != len(expected) {
		t.Fatalf("SWBOffset1024_96 length = %d, want %d", len(SWBOffset1024_96), len(expected))
	}

	for i, v := range expected {
		if SWBOffset1024_96[i] != v {
			t.Errorf("SWBOffset1024_96[%d] = %d, want %d", i, SWBOffset1024_96[i], v)
		}
	}
}

func TestSWBOffset1024_48(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/specrec.c:116-122
	expected := []uint16{
		0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 48, 56, 64, 72,
		80, 88, 96, 108, 120, 132, 144, 160, 176, 196, 216, 240, 264, 292,
		320, 352, 384, 416, 448, 480, 512, 544, 576, 608, 640, 672, 704, 736,
		768, 800, 832, 864, 896, 928, 1024,
	}

	if len(SWBOffset1024_48) != len(expected) {
		t.Fatalf("SWBOffset1024_48 length = %d, want %d", len(SWBOffset1024_48), len(expected))
	}

	for i, v := range expected {
		if SWBOffset1024_48[i] != v {
			t.Errorf("SWBOffset1024_48[%d] = %d, want %d", i, SWBOffset1024_48[i], v)
		}
	}
}
```

### Step 3.6: Run test to verify it fails

Run: `make test PKG=./internal/tables`
Expected: FAIL with "undefined: SWBOffset1024_96"

### Step 3.7: Write all SFB offset tables for long windows

```go
// Add to internal/tables/sfb_long.go

// SWBOffset1024_96 contains SFB offsets for 96kHz/88.2kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:91-96
var SWBOffset1024_96 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56,
	64, 72, 80, 88, 96, 108, 120, 132, 144, 156, 172, 188, 212, 240,
	276, 320, 384, 448, 512, 576, 640, 704, 768, 832, 896, 960, 1024,
}

// SWBOffset1024_64 contains SFB offsets for 64kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:103-109
var SWBOffset1024_64 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56,
	64, 72, 80, 88, 100, 112, 124, 140, 156, 172, 192, 216, 240, 268,
	304, 344, 384, 424, 464, 504, 544, 584, 624, 664, 704, 744, 784, 824,
	864, 904, 944, 984, 1024,
}

// SWBOffset1024_48 contains SFB offsets for 48kHz/44.1kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:116-122
var SWBOffset1024_48 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 48, 56, 64, 72,
	80, 88, 96, 108, 120, 132, 144, 160, 176, 196, 216, 240, 264, 292,
	320, 352, 384, 416, 448, 480, 512, 544, 576, 608, 640, 672, 704, 736,
	768, 800, 832, 864, 896, 928, 1024,
}

// SWBOffset1024_32 contains SFB offsets for 32kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:145-151
var SWBOffset1024_32 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 48, 56, 64, 72,
	80, 88, 96, 108, 120, 132, 144, 160, 176, 196, 216, 240, 264, 292,
	320, 352, 384, 416, 448, 480, 512, 544, 576, 608, 640, 672, 704, 736,
	768, 800, 832, 864, 896, 928, 960, 992, 1024,
}

// SWBOffset1024_24 contains SFB offsets for 24kHz/22.05kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:169-175
var SWBOffset1024_24 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 52, 60, 68,
	76, 84, 92, 100, 108, 116, 124, 136, 148, 160, 172, 188, 204, 220,
	240, 260, 284, 308, 336, 364, 396, 432, 468, 508, 552, 600, 652, 704,
	768, 832, 896, 960, 1024,
}

// SWBOffset1024_16 contains SFB offsets for 16kHz/12kHz/11.025kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:197-202
var SWBOffset1024_16 = []uint16{
	0, 8, 16, 24, 32, 40, 48, 56, 64, 72, 80, 88, 100, 112, 124,
	136, 148, 160, 172, 184, 196, 212, 228, 244, 260, 280, 300, 320, 344,
	368, 396, 424, 456, 492, 532, 572, 616, 664, 716, 772, 832, 896, 960, 1024,
}

// SWBOffset1024_8 contains SFB offsets for 8kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:209-214
var SWBOffset1024_8 = []uint16{
	0, 12, 24, 36, 48, 60, 72, 84, 96, 108, 120, 132, 144, 156, 172,
	188, 204, 220, 236, 252, 268, 288, 308, 328, 348, 372, 396, 420, 448,
	476, 508, 544, 580, 620, 664, 712, 764, 820, 880, 944, 1024,
}

// SWBOffset1024Window maps sample rate index to the appropriate SFB offset table.
// Source: ~/dev/faad2/libfaad/specrec.c:221-235
var SWBOffset1024Window = [12][]uint16{
	SWBOffset1024_96, // 96000
	SWBOffset1024_96, // 88200
	SWBOffset1024_64, // 64000
	SWBOffset1024_48, // 48000
	SWBOffset1024_48, // 44100
	SWBOffset1024_32, // 32000
	SWBOffset1024_24, // 24000
	SWBOffset1024_24, // 22050
	SWBOffset1024_16, // 16000
	SWBOffset1024_16, // 12000
	SWBOffset1024_16, // 11025
	SWBOffset1024_8,  // 8000
}
```

### Step 3.8: Run test to verify it passes

Run: `make test PKG=./internal/tables`
Expected: PASS

### Step 3.9: Commit

```bash
git add internal/tables/sfb_long.go internal/tables/sfb_long_test.go
git commit -m "feat(tables): add long window SFB offset tables

- NumSWB1024Window and NumSWB960Window for SFB counts
- SWBOffset1024_* tables for all sample rate groups
- SWBOffset1024Window lookup array by sample rate index

Ported from: ~/dev/faad2/libfaad/specrec.c:66-235

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Short Window SFB Tables

**Files:**
- Create: `internal/tables/sfb_short.go`
- Create: `internal/tables/sfb_short_test.go`

### Step 4.1: Write failing test for short window SFB count

```go
// internal/tables/sfb_short_test.go
package tables

import "testing"

func TestNumSWB128Window(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/specrec.c:86-89
	expected := [12]uint8{12, 12, 12, 14, 14, 14, 15, 15, 15, 15, 15, 15}

	for i := 0; i < 12; i++ {
		if NumSWB128Window[i] != expected[i] {
			t.Errorf("NumSWB128Window[%d] = %d, want %d", i, NumSWB128Window[i], expected[i])
		}
	}
}
```

### Step 4.2: Run test to verify it fails

Run: `make test PKG=./internal/tables`
Expected: FAIL with "undefined: NumSWB128Window"

### Step 4.3: Write short window count table

```go
// internal/tables/sfb_short.go
package tables

// NumSWB128Window contains the number of scale factor window bands
// for 128-sample short windows at each sample rate index.
// Source: ~/dev/faad2/libfaad/specrec.c:86-89
var NumSWB128Window = [12]uint8{
	12, 12, 12, 14, 14, 14, 15, 15, 15, 15, 15, 15,
}
```

### Step 4.4: Run test to verify it passes

Run: `make test PKG=./internal/tables`
Expected: PASS

### Step 4.5: Write failing test for short window SFB offset tables

```go
// Add to internal/tables/sfb_short_test.go
func TestSWBOffset128_48(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/specrec.c:140-143
	expected := []uint16{0, 4, 8, 12, 16, 20, 28, 36, 44, 56, 68, 80, 96, 112, 128}

	if len(SWBOffset128_48) != len(expected) {
		t.Fatalf("SWBOffset128_48 length = %d, want %d", len(SWBOffset128_48), len(expected))
	}

	for i, v := range expected {
		if SWBOffset128_48[i] != v {
			t.Errorf("SWBOffset128_48[%d] = %d, want %d", i, SWBOffset128_48[i], v)
		}
	}
}

func TestSWBOffset128Window(t *testing.T) {
	// Verify the lookup array is properly configured
	// Source: ~/dev/faad2/libfaad/specrec.c:271-285

	// 48kHz should use SWBOffset128_48
	if len(SWBOffset128Window[3]) != len(SWBOffset128_48) {
		t.Errorf("SWBOffset128Window[3] should point to SWBOffset128_48")
	}

	// 96kHz should use SWBOffset128_96
	if len(SWBOffset128Window[0]) != len(SWBOffset128_96) {
		t.Errorf("SWBOffset128Window[0] should point to SWBOffset128_96")
	}
}
```

### Step 4.6: Run test to verify it fails

Run: `make test PKG=./internal/tables`
Expected: FAIL with "undefined: SWBOffset128_48"

### Step 4.7: Write all short window SFB offset tables

```go
// Add to internal/tables/sfb_short.go

// SWBOffset128_96 contains SFB offsets for 96kHz/88.2kHz at 128 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:98-101
var SWBOffset128_96 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 92, 128,
}

// SWBOffset128_64 contains SFB offsets for 64kHz at 128 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:111-114
var SWBOffset128_64 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 92, 128,
}

// SWBOffset128_48 contains SFB offsets for 48kHz/44.1kHz/32kHz at 128 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:140-143
var SWBOffset128_48 = []uint16{
	0, 4, 8, 12, 16, 20, 28, 36, 44, 56, 68, 80, 96, 112, 128,
}

// SWBOffset128_24 contains SFB offsets for 24kHz/22.05kHz at 128 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:192-195
var SWBOffset128_24 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 36, 44, 52, 64, 76, 92, 108, 128,
}

// SWBOffset128_16 contains SFB offsets for 16kHz/12kHz/11.025kHz at 128 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:204-207
var SWBOffset128_16 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 40, 48, 60, 72, 88, 108, 128,
}

// SWBOffset128_8 contains SFB offsets for 8kHz at 128 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:216-219
var SWBOffset128_8 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 36, 44, 52, 60, 72, 88, 108, 128,
}

// SWBOffset128Window maps sample rate index to the appropriate short window SFB offset table.
// Source: ~/dev/faad2/libfaad/specrec.c:271-285
var SWBOffset128Window = [12][]uint16{
	SWBOffset128_96, // 96000
	SWBOffset128_96, // 88200
	SWBOffset128_64, // 64000
	SWBOffset128_48, // 48000
	SWBOffset128_48, // 44100
	SWBOffset128_48, // 32000
	SWBOffset128_24, // 24000
	SWBOffset128_24, // 22050
	SWBOffset128_16, // 16000
	SWBOffset128_16, // 12000
	SWBOffset128_16, // 11025
	SWBOffset128_8,  // 8000
}
```

### Step 4.8: Run test to verify it passes

Run: `make test PKG=./internal/tables`
Expected: PASS

### Step 4.9: Commit

```bash
git add internal/tables/sfb_short.go internal/tables/sfb_short_test.go
git commit -m "feat(tables): add short window SFB offset tables

- NumSWB128Window for SFB counts
- SWBOffset128_* tables for all sample rate groups
- SWBOffset128Window lookup array by sample rate index

Ported from: ~/dev/faad2/libfaad/specrec.c:86-285

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 5: SFB Lookup Interface

**Files:**
- Create: `internal/tables/sfb.go`
- Create: `internal/tables/sfb_test.go`

### Step 5.1: Write failing test for GetSWBOffset

```go
// internal/tables/sfb_test.go
package tables

import "testing"

func TestGetSWBOffset(t *testing.T) {
	tests := []struct {
		srIndex     uint8
		frameLength uint16
		isShort     bool
		wantLen     int
	}{
		{3, 1024, false, 50}, // 48kHz long
		{3, 1024, true, 15},  // 48kHz short
		{0, 1024, false, 42}, // 96kHz long
		{0, 1024, true, 13},  // 96kHz short
		{11, 1024, false, 41}, // 8kHz long
		{11, 1024, true, 16},  // 8kHz short
	}

	for _, tt := range tests {
		offsets, err := GetSWBOffset(tt.srIndex, tt.frameLength, tt.isShort)
		if err != nil {
			t.Errorf("GetSWBOffset(%d, %d, %v) error: %v", tt.srIndex, tt.frameLength, tt.isShort, err)
			continue
		}
		if len(offsets) != tt.wantLen {
			t.Errorf("GetSWBOffset(%d, %d, %v) len = %d, want %d",
				tt.srIndex, tt.frameLength, tt.isShort, len(offsets), tt.wantLen)
		}
	}
}

func TestGetSWBOffsetInvalidIndex(t *testing.T) {
	_, err := GetSWBOffset(12, 1024, false)
	if err == nil {
		t.Error("GetSWBOffset(12, 1024, false) should return error")
	}
}

func TestGetNumSWB(t *testing.T) {
	tests := []struct {
		srIndex     uint8
		frameLength uint16
		isShort     bool
		want        uint8
	}{
		{3, 1024, false, 49}, // 48kHz long
		{3, 1024, true, 14},  // 48kHz short
		{0, 1024, false, 41}, // 96kHz long
		{4, 960, false, 49},  // 44.1kHz 960-sample
	}

	for _, tt := range tests {
		got, err := GetNumSWB(tt.srIndex, tt.frameLength, tt.isShort)
		if err != nil {
			t.Errorf("GetNumSWB(%d, %d, %v) error: %v", tt.srIndex, tt.frameLength, tt.isShort, err)
			continue
		}
		if got != tt.want {
			t.Errorf("GetNumSWB(%d, %d, %v) = %d, want %d",
				tt.srIndex, tt.frameLength, tt.isShort, got, tt.want)
		}
	}
}
```

### Step 5.2: Run test to verify it fails

Run: `make test PKG=./internal/tables`
Expected: FAIL with "undefined: GetSWBOffset"

### Step 5.3: Write SFB lookup functions

```go
// internal/tables/sfb.go
package tables

import "errors"

// ErrInvalidSRIndex indicates an invalid sample rate index.
var ErrInvalidSRIndex = errors.New("tables: invalid sample rate index")

// GetSWBOffset returns the SFB offset table for the given parameters.
// Source: ~/dev/faad2/libfaad/specrec.c:221-285
func GetSWBOffset(srIndex uint8, frameLength uint16, isShort bool) ([]uint16, error) {
	if srIndex >= 12 {
		return nil, ErrInvalidSRIndex
	}

	if isShort {
		return SWBOffset128Window[srIndex], nil
	}

	// Long window - only 1024 supported for now (960 for AAC LD would need additional tables)
	return SWBOffset1024Window[srIndex], nil
}

// GetNumSWB returns the number of scale factor window bands.
// Source: ~/dev/faad2/libfaad/specrec.c:66-89
func GetNumSWB(srIndex uint8, frameLength uint16, isShort bool) (uint8, error) {
	if srIndex >= 12 {
		return 0, ErrInvalidSRIndex
	}

	if isShort {
		return NumSWB128Window[srIndex], nil
	}

	if frameLength == 960 {
		return NumSWB960Window[srIndex], nil
	}

	return NumSWB1024Window[srIndex], nil
}
```

### Step 5.4: Run test to verify it passes

Run: `make test PKG=./internal/tables`
Expected: PASS

### Step 5.5: Commit

```bash
git add internal/tables/sfb.go internal/tables/sfb_test.go
git commit -m "feat(tables): add SFB lookup interface functions

- GetSWBOffset(srIndex, frameLength, isShort) returns offset table
- GetNumSWB(srIndex, frameLength, isShort) returns SFB count
- Error handling for invalid sample rate indices

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Run Full Test Suite

**Files:** None (verification only)

### Step 6.1: Run make check

Run: `make check`
Expected: All checks pass (fmt, lint, test)

### Step 6.2: Review coverage

Run: `make coverage PKG=./internal/tables`
Expected: Good coverage of all new functions

### Step 6.3: Final commit if needed

If any adjustments were made during verification:
```bash
git add -u
git commit -m "chore(tables): cleanup and polish

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Summary

**Files Created:**
- `internal/tables/sample_rates.go` - Sample rate lookup and limit functions
- `internal/tables/sample_rates_test.go`
- `internal/tables/sfb_long.go` - Long window SFB offset tables
- `internal/tables/sfb_long_test.go`
- `internal/tables/sfb_short.go` - Short window SFB offset tables
- `internal/tables/sfb_short_test.go`
- `internal/tables/sfb.go` - Unified SFB lookup interface
- `internal/tables/sfb_test.go`

**Functions Implemented:**
- `GetSampleRate(srIndex)` - Sample rate from index
- `GetSRIndex(sampleRate)` - Index from sample rate (threshold-based)
- `MaxPredSFB(srIndex)` - Max prediction SFB
- `MaxTNSSFB(srIndex, objectType, isShort)` - Max TNS SFB
- `CanDecodeOT(objectType)` - Object type support check
- `GetSWBOffset(srIndex, frameLength, isShort)` - SFB offset table lookup
- `GetNumSWB(srIndex, frameLength, isShort)` - SFB count lookup

**Tables Ported:**
- Sample rate table (12 entries)
- Prediction SFB max table (12 entries)
- TNS SFB max table (16x4 entries)
- NumSWB tables (1024, 960, 128 windows)
- SWB offset tables (7 long + 6 short variants)
