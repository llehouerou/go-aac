# Huffman Codebook Tables Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Port all 12 Huffman codebook tables from FAAD2 to Go, matching the exact binary representation.

**Architecture:** Each codebook is a separate Go file with lookup tables. We use two decoding methods: 2-step table lookup (for codebooks 1, 2, 4, 6, 8, 10, 11) and binary search (for codebooks 3, 5, 7, 9, SF). All tables are copied exactly from FAAD2 without modification.

**Tech Stack:** Go 1.21+, standard library only, TDD with table-driven tests

---

## Background

### Codebook Methods

| Codebook | Method | Output Type | Root Size | Second Table Size |
|----------|--------|-------------|-----------|-------------------|
| HCB_1 | 2-Step | Quad (4 values) | 32 | 113 |
| HCB_2 | 2-Step | Quad (4 values) | 32 | 85 |
| HCB_3 | Binary | Quad (4 values) | - | 161 nodes |
| HCB_4 | 2-Step | Quad (4 values) | 32 | 184 |
| HCB_5 | Binary | Pair (2 values) | - | 161 nodes |
| HCB_6 | 2-Step | Pair (2 values) | 32 | 125 |
| HCB_7 | Binary | Pair (2 values) | - | 127 nodes |
| HCB_8 | 2-Step | Pair (2 values) | 32 | 83 |
| HCB_9 | Binary | Pair (2 values) | - | 337 nodes |
| HCB_10 | 2-Step | Pair (2 values) | 64 | 209 |
| HCB_11 | 2-Step | Pair (2 values) | 32 | 374 |
| HCB_SF | Binary | Scale Factor | - | 241 nodes |

### Data Structures (from FAAD2)

```c
// 1st step table entry
typedef struct { uint8_t offset; uint8_t extra_bits; } hcb;

// 2nd step table - quadruple output
typedef struct { uint8_t bits; int8_t x, y, v, w; } hcb_2_quad;

// 2nd step table - pair output
typedef struct { uint8_t bits; int8_t x, y; } hcb_2_pair;

// Binary search - quadruple
typedef struct { uint8_t is_leaf; int8_t data[4]; } hcb_bin_quad;

// Binary search - pair
typedef struct { uint8_t is_leaf; int8_t data[2]; } hcb_bin_pair;
```

---

## Task 1: Create Codebook Type Definitions

**Files:**
- Create: `internal/huffman/types.go`
- Test: `internal/huffman/types_test.go`

**Step 1: Write the failing test**

```go
// internal/huffman/types_test.go
package huffman

import (
	"testing"
	"unsafe"
)

func TestHCBStructSize(t *testing.T) {
	// HCB struct should be 2 bytes (offset + extra_bits)
	if size := unsafe.Sizeof(HCB{}); size != 2 {
		t.Errorf("HCB size = %d, want 2", size)
	}
}

func TestHCB2QuadStructSize(t *testing.T) {
	// HCB2Quad struct should be 5 bytes (bits + x + y + v + w)
	if size := unsafe.Sizeof(HCB2Quad{}); size != 5 {
		t.Errorf("HCB2Quad size = %d, want 5", size)
	}
}

func TestHCB2PairStructSize(t *testing.T) {
	// HCB2Pair struct should be 3 bytes (bits + x + y)
	if size := unsafe.Sizeof(HCB2Pair{}); size != 3 {
		t.Errorf("HCB2Pair size = %d, want 3", size)
	}
}

func TestHCBBinQuadStructSize(t *testing.T) {
	// HCBBinQuad struct should be 5 bytes (is_leaf + data[4])
	if size := unsafe.Sizeof(HCBBinQuad{}); size != 5 {
		t.Errorf("HCBBinQuad size = %d, want 5", size)
	}
}

func TestHCBBinPairStructSize(t *testing.T) {
	// HCBBinPair struct should be 3 bytes (is_leaf + data[2])
	if size := unsafe.Sizeof(HCBBinPair{}); size != 3 {
		t.Errorf("HCBBinPair size = %d, want 3", size)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/huffman`
Expected: FAIL with "undefined: HCB"

**Step 3: Write minimal implementation**

```go
// internal/huffman/types.go
package huffman

// HCB is a first-step table entry for 2-step Huffman decoding.
// It maps initial bits to an offset in the second-step table and
// indicates how many additional bits to read.
//
// Ported from: hcb struct in ~/dev/faad2/libfaad/codebook/hcb.h:85-89
type HCB struct {
	Offset    uint8 // Index into second-step table
	ExtraBits uint8 // Number of additional bits to read
}

// HCB2Quad is a second-step table entry for quadruple codebooks.
// Used by codebooks 1, 2, 4 which output 4 spectral coefficients.
//
// Ported from: hcb_2_quad struct in ~/dev/faad2/libfaad/codebook/hcb.h:99-106
type HCB2Quad struct {
	Bits uint8 // Total codeword length
	X    int8  // First coefficient
	Y    int8  // Second coefficient
	V    int8  // Third coefficient
	W    int8  // Fourth coefficient
}

// HCB2Pair is a second-step table entry for pair codebooks.
// Used by codebooks 6, 8, 10, 11 which output 2 spectral coefficients.
//
// Ported from: hcb_2_pair struct in ~/dev/faad2/libfaad/codebook/hcb.h:92-97
type HCB2Pair struct {
	Bits uint8 // Total codeword length
	X    int8  // First coefficient
	Y    int8  // Second coefficient
}

// HCBBinQuad is a binary search tree node for quadruple codebooks.
// Used by codebook 3 which outputs 4 spectral coefficients.
//
// Ported from: hcb_bin_quad struct in ~/dev/faad2/libfaad/codebook/hcb.h:109-113
type HCBBinQuad struct {
	IsLeaf uint8    // 1 if leaf node with data, 0 if internal with branch offsets
	Data   [4]int8  // Leaf: output values; Internal: branch offsets in data[0], data[1]
}

// HCBBinPair is a binary search tree node for pair codebooks.
// Used by codebooks 5, 7, 9 which output 2 spectral coefficients.
//
// Ported from: hcb_bin_pair struct in ~/dev/faad2/libfaad/codebook/hcb.h:115-119
type HCBBinPair struct {
	IsLeaf uint8   // 1 if leaf node with data, 0 if internal with branch offsets
	Data   [2]int8 // Leaf: output values; Internal: branch offsets
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/huffman`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/huffman/types.go internal/huffman/types_test.go
git commit -m "feat(huffman): add codebook type definitions

Port Huffman codebook structures from FAAD2:
- HCB: first-step table entry (offset + extra_bits)
- HCB2Quad: second-step quad entry (4 coefficients)
- HCB2Pair: second-step pair entry (2 coefficients)
- HCBBinQuad: binary search quad node
- HCBBinPair: binary search pair node

All struct sizes match FAAD2 exactly."
```

---

## Task 2: Port Codebook 1 (2-Step Quad)

**Files:**
- Create: `internal/huffman/codebook_1.go`
- Create: `internal/huffman/codebook_1_test.go`

**Step 1: Write the failing test**

```go
// internal/huffman/codebook_1_test.go
package huffman

import "testing"

func TestHCB1_1Size(t *testing.T) {
	// First-step table must have 32 entries (2^5 bits)
	if len(hcb1_1) != 32 {
		t.Errorf("hcb1_1 size = %d, want 32", len(hcb1_1))
	}
}

func TestHCB1_2Size(t *testing.T) {
	// Second-step table must have 113 entries
	if len(hcb1_2) != 113 {
		t.Errorf("hcb1_2 size = %d, want 113", len(hcb1_2))
	}
}

func TestHCB1_1Values(t *testing.T) {
	// Verify key entries from FAAD2
	tests := []struct {
		idx      int
		offset   uint8
		extra    uint8
	}{
		{0, 0, 0},   // 1-bit codeword maps to index 0
		{16, 1, 0},  // 5-bit: 10000 -> offset 1
		{24, 9, 2},  // 7-bit: 11000 -> offset 9, 2 extra bits
		{30, 33, 4}, // 9-bit: 11110 -> offset 33, 4 extra bits
		{31, 49, 6}, // 9/10/11-bit: 11111 -> offset 49, 6 extra bits
	}
	for _, tt := range tests {
		if hcb1_1[tt.idx].Offset != tt.offset || hcb1_1[tt.idx].ExtraBits != tt.extra {
			t.Errorf("hcb1_1[%d] = {%d, %d}, want {%d, %d}",
				tt.idx, hcb1_1[tt.idx].Offset, hcb1_1[tt.idx].ExtraBits,
				tt.offset, tt.extra)
		}
	}
}

func TestHCB1_2Values(t *testing.T) {
	// Verify key entries from FAAD2
	tests := []struct {
		idx  int
		bits uint8
		x, y, v, w int8
	}{
		{0, 1, 0, 0, 0, 0},    // 1-bit codeword: all zeros
		{1, 5, 1, 0, 0, 0},    // 5-bit: (1,0,0,0)
		{9, 7, 1, -1, 0, 0},   // 7-bit: (1,-1,0,0)
		{112, 11, 1, 1, 1, -1}, // 11-bit: last entry
	}
	for _, tt := range tests {
		e := hcb1_2[tt.idx]
		if e.Bits != tt.bits || e.X != tt.x || e.Y != tt.y || e.V != tt.v || e.W != tt.w {
			t.Errorf("hcb1_2[%d] = {%d, %d, %d, %d, %d}, want {%d, %d, %d, %d, %d}",
				tt.idx, e.Bits, e.X, e.Y, e.V, e.W,
				tt.bits, tt.x, tt.y, tt.v, tt.w)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/huffman`
Expected: FAIL with "undefined: hcb1_1"

**Step 3: Write minimal implementation**

Copy the tables exactly from `~/dev/faad2/libfaad/codebook/hcb_1.h`:

```go
// internal/huffman/codebook_1.go
package huffman

// hcb1_1 is the first-step lookup table for codebook 1.
// 5 bits are read initially (32 entries).
//
// Ported from: hcb1_1[32] in ~/dev/faad2/libfaad/codebook/hcb_1.h:39-83
var hcb1_1 = [32]HCB{
	// 1-bit codeword (0 -> all zeros)
	{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0},
	{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0},
	// 5-bit codewords (10000 - 10111)
	{1, 0}, {2, 0}, {3, 0}, {4, 0}, {5, 0}, {6, 0}, {7, 0}, {8, 0},
	// 7-bit codewords (11000 - 11101)
	{9, 2}, {13, 2}, {17, 2}, {21, 2}, {25, 2}, {29, 2},
	// 9-bit codewords (11110)
	{33, 4},
	// 9/10/11-bit codewords (11111)
	{49, 6},
}

// hcb1_2 is the second-step lookup table for codebook 1.
// Contains output values (x, y, v, w) for each codeword.
//
// Ported from: hcb1_2[113] in ~/dev/faad2/libfaad/codebook/hcb_1.h:89-191
var hcb1_2 = [113]HCB2Quad{
	// 1-bit codeword
	{1, 0, 0, 0, 0},
	// 5-bit codewords
	{5, 1, 0, 0, 0}, {5, -1, 0, 0, 0}, {5, 0, 0, 0, -1}, {5, 0, 1, 0, 0},
	{5, 0, 0, 0, 1}, {5, 0, 0, -1, 0}, {5, 0, 0, 1, 0}, {5, 0, -1, 0, 0},
	// 7-bit codewords (11000xx)
	{7, 1, -1, 0, 0}, {7, -1, 1, 0, 0}, {7, 0, 0, -1, 1}, {7, 0, 1, -1, 0},
	// 7-bit codewords (11001xx)
	{7, 0, -1, 1, 0}, {7, 0, 0, 1, -1}, {7, 1, 1, 0, 0}, {7, 0, 0, -1, -1},
	// 7-bit codewords (11010xx)
	{7, -1, -1, 0, 0}, {7, 0, -1, -1, 0}, {7, 1, 0, -1, 0}, {7, 0, 1, 0, -1},
	// 7-bit codewords (11011xx)
	{7, -1, 0, 1, 0}, {7, 0, 0, 1, 1}, {7, 1, 0, 1, 0}, {7, 0, -1, 0, 1},
	// 7-bit codewords (11100xx)
	{7, 0, 1, 1, 0}, {7, 0, 1, 0, 1}, {7, -1, 0, -1, 0}, {7, 1, 0, 0, 1},
	// 7-bit codewords (11101xx)
	{7, -1, 0, 0, -1}, {7, 1, 0, 0, -1}, {7, -1, 0, 0, 1}, {7, 0, -1, 0, -1},
	// 9-bit codewords (11110xxxx)
	{9, 1, 1, -1, 0}, {9, -1, 1, -1, 0}, {9, 1, -1, 1, 0}, {9, 0, 1, 1, -1},
	{9, 0, 1, -1, 1}, {9, 0, -1, 1, 1}, {9, 0, -1, 1, -1}, {9, 1, -1, -1, 0},
	{9, 1, 0, -1, 1}, {9, 0, 1, -1, -1}, {9, -1, 1, 1, 0}, {9, -1, 0, 1, -1},
	{9, -1, -1, 1, 0}, {9, 0, -1, -1, 1}, {9, 1, -1, 0, 1}, {9, 1, -1, 0, -1},
	// 9/10/11-bit codewords (11111xxxxxx)
	// 9-bit: 4 entries each
	{9, -1, 1, 0, -1}, {9, -1, 1, 0, -1}, {9, -1, 1, 0, -1}, {9, -1, 1, 0, -1},
	{9, -1, -1, -1, 0}, {9, -1, -1, -1, 0}, {9, -1, -1, -1, 0}, {9, -1, -1, -1, 0},
	{9, 0, -1, -1, -1}, {9, 0, -1, -1, -1}, {9, 0, -1, -1, -1}, {9, 0, -1, -1, -1},
	{9, 0, 1, 1, 1}, {9, 0, 1, 1, 1}, {9, 0, 1, 1, 1}, {9, 0, 1, 1, 1},
	{9, 1, 0, 1, -1}, {9, 1, 0, 1, -1}, {9, 1, 0, 1, -1}, {9, 1, 0, 1, -1},
	{9, 1, 1, 0, 1}, {9, 1, 1, 0, 1}, {9, 1, 1, 0, 1}, {9, 1, 1, 0, 1},
	{9, -1, 1, 0, 1}, {9, -1, 1, 0, 1}, {9, -1, 1, 0, 1}, {9, -1, 1, 0, 1},
	{9, 1, 1, 1, 0}, {9, 1, 1, 1, 0}, {9, 1, 1, 1, 0}, {9, 1, 1, 1, 0},
	// 10-bit: 2 entries each
	{10, -1, -1, 0, 1}, {10, -1, -1, 0, 1},
	{10, -1, 0, -1, -1}, {10, -1, 0, -1, -1},
	{10, 1, 1, 0, -1}, {10, 1, 1, 0, -1},
	{10, 1, 0, -1, -1}, {10, 1, 0, -1, -1},
	{10, -1, 0, -1, 1}, {10, -1, 0, -1, 1},
	{10, -1, -1, 0, -1}, {10, -1, -1, 0, -1},
	{10, -1, 0, 1, 1}, {10, -1, 0, 1, 1},
	{10, 1, 0, 1, 1}, {10, 1, 0, 1, 1},
	// 11-bit: 1 entry each
	{11, 1, -1, 1, -1},
	{11, -1, 1, -1, 1},
	{11, -1, 1, 1, -1},
	{11, 1, -1, -1, 1},
	{11, 1, 1, 1, 1},
	{11, -1, -1, 1, 1},
	{11, 1, 1, -1, -1},
	{11, -1, -1, 1, -1},
	{11, -1, -1, -1, -1},
	{11, 1, 1, -1, 1},
	{11, 1, -1, 1, 1},
	{11, -1, 1, 1, 1},
	{11, -1, 1, -1, -1},
	{11, -1, -1, -1, 1},
	{11, 1, -1, -1, -1},
	{11, 1, 1, 1, -1},
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/huffman`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/huffman/codebook_1.go internal/huffman/codebook_1_test.go
git commit -m "feat(huffman): port codebook 1 (2-step quad)

Port HCB_1 tables from FAAD2:
- hcb1_1: 32-entry first-step table (5 bits)
- hcb1_2: 113-entry second-step table (quad output)

Tables copied exactly from ~/dev/faad2/libfaad/codebook/hcb_1.h"
```

---

## Task 3: Port Codebook 2 (2-Step Quad)

**Files:**
- Create: `internal/huffman/codebook_2.go`
- Create: `internal/huffman/codebook_2_test.go`

**Step 1: Write the failing test**

```go
// internal/huffman/codebook_2_test.go
package huffman

import "testing"

func TestHCB2_1Size(t *testing.T) {
	if len(hcb2_1) != 32 {
		t.Errorf("hcb2_1 size = %d, want 32", len(hcb2_1))
	}
}

func TestHCB2_2Size(t *testing.T) {
	if len(hcb2_2) != 85 {
		t.Errorf("hcb2_2 size = %d, want 85", len(hcb2_2))
	}
}

func TestHCB2_1Values(t *testing.T) {
	tests := []struct {
		idx    int
		offset uint8
		extra  uint8
	}{
		{0, 0, 0},   // 3-bit: 000 -> index 0
		{8, 1, 0},   // 5-bit: 01000 -> offset 1
		{16, 9, 2},  // 7-bit: 10000 -> offset 9, 2 extra
		{31, 53, 5}, // last entry
	}
	for _, tt := range tests {
		if hcb2_1[tt.idx].Offset != tt.offset || hcb2_1[tt.idx].ExtraBits != tt.extra {
			t.Errorf("hcb2_1[%d] = {%d, %d}, want {%d, %d}",
				tt.idx, hcb2_1[tt.idx].Offset, hcb2_1[tt.idx].ExtraBits,
				tt.offset, tt.extra)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/huffman`
Expected: FAIL with "undefined: hcb2_1"

**Step 3: Write minimal implementation**

Copy tables from `~/dev/faad2/libfaad/codebook/hcb_2.h`. (Full table content to be copied during implementation.)

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/huffman`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/huffman/codebook_2.go internal/huffman/codebook_2_test.go
git commit -m "feat(huffman): port codebook 2 (2-step quad)

Port HCB_2 tables from FAAD2:
- hcb2_1: 32-entry first-step table
- hcb2_2: 85-entry second-step table"
```

---

## Task 4: Port Codebook 3 (Binary Quad)

**Files:**
- Create: `internal/huffman/codebook_3.go`
- Create: `internal/huffman/codebook_3_test.go`

**Step 1: Write the failing test**

```go
// internal/huffman/codebook_3_test.go
package huffman

import "testing"

func TestHCB3Size(t *testing.T) {
	if len(hcb3) != 161 {
		t.Errorf("hcb3 size = %d, want 161", len(hcb3))
	}
}

func TestHCB3Values(t *testing.T) {
	tests := []struct {
		idx    int
		isLeaf uint8
		data   [4]int8
	}{
		{0, 0, [4]int8{1, 2, 0, 0}},       // Root: internal node
		{1, 1, [4]int8{0, 0, 0, 0}},       // Leaf: (0,0,0,0)
		{9, 1, [4]int8{1, 0, 0, 0}},       // Leaf: (1,0,0,0)
		{160, 1, [4]int8{2, 0, 2, 2}},     // Last entry
	}
	for _, tt := range tests {
		e := hcb3[tt.idx]
		if e.IsLeaf != tt.isLeaf || e.Data != tt.data {
			t.Errorf("hcb3[%d] = {%d, %v}, want {%d, %v}",
				tt.idx, e.IsLeaf, e.Data, tt.isLeaf, tt.data)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/huffman`
Expected: FAIL with "undefined: hcb3"

**Step 3: Write minimal implementation**

Copy binary search table from `~/dev/faad2/libfaad/codebook/hcb_3.h`.

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/huffman`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/huffman/codebook_3.go internal/huffman/codebook_3_test.go
git commit -m "feat(huffman): port codebook 3 (binary quad)

Port HCB_3 binary search table from FAAD2:
- hcb3: 161-node binary tree for quad output"
```

---

## Task 5: Port Codebook 4 (2-Step Quad)

**Files:**
- Create: `internal/huffman/codebook_4.go`
- Create: `internal/huffman/codebook_4_test.go`

**Step 1-5:** Same pattern as Task 2, with:
- hcb4_1: 32 entries
- hcb4_2: 184 entries

**Commit message:**
```
feat(huffman): port codebook 4 (2-step quad with signs)
```

---

## Task 6: Port Codebook 5 (Binary Pair)

**Files:**
- Create: `internal/huffman/codebook_5.go`
- Create: `internal/huffman/codebook_5_test.go`

**Step 1: Write the failing test**

```go
// internal/huffman/codebook_5_test.go
package huffman

import "testing"

func TestHCB5Size(t *testing.T) {
	if len(hcb5) != 161 {
		t.Errorf("hcb5 size = %d, want 161", len(hcb5))
	}
}

func TestHCB5Values(t *testing.T) {
	tests := []struct {
		idx    int
		isLeaf uint8
		data   [2]int8
	}{
		{0, 0, [2]int8{1, 2}},    // Root: internal
		{1, 1, [2]int8{0, 0}},    // Leaf: (0,0)
	}
	for _, tt := range tests {
		e := hcb5[tt.idx]
		if e.IsLeaf != tt.isLeaf || e.Data != tt.data {
			t.Errorf("hcb5[%d] = {%d, %v}, want {%d, %v}",
				tt.idx, e.IsLeaf, e.Data, tt.isLeaf, tt.data)
		}
	}
}
```

**Step 2-5:** Follow standard pattern.

**Commit message:**
```
feat(huffman): port codebook 5 (binary pair)
```

---

## Task 7: Port Codebook 6 (2-Step Pair)

**Files:**
- Create: `internal/huffman/codebook_6.go`
- Create: `internal/huffman/codebook_6_test.go`

Pattern: 2-step with HCB2Pair output
- hcb6_1: 32 entries
- hcb6_2: 125 entries

**Commit message:**
```
feat(huffman): port codebook 6 (2-step pair)
```

---

## Task 8: Port Codebook 7 (Binary Pair)

**Files:**
- Create: `internal/huffman/codebook_7.go`
- Create: `internal/huffman/codebook_7_test.go`

Pattern: Binary search with HCBBinPair
- hcb7: 127 entries

**Commit message:**
```
feat(huffman): port codebook 7 (binary pair)
```

---

## Task 9: Port Codebook 8 (2-Step Pair)

**Files:**
- Create: `internal/huffman/codebook_8.go`
- Create: `internal/huffman/codebook_8_test.go`

Pattern: 2-step with HCB2Pair
- hcb8_1: 32 entries
- hcb8_2: 83 entries

**Commit message:**
```
feat(huffman): port codebook 8 (2-step pair)
```

---

## Task 10: Port Codebook 9 (Binary Pair)

**Files:**
- Create: `internal/huffman/codebook_9.go`
- Create: `internal/huffman/codebook_9_test.go`

Pattern: Binary search with HCBBinPair
- hcb9: 337 entries

**Commit message:**
```
feat(huffman): port codebook 9 (binary pair)
```

---

## Task 11: Port Codebook 10 (2-Step Pair)

**Files:**
- Create: `internal/huffman/codebook_10.go`
- Create: `internal/huffman/codebook_10_test.go`

Pattern: 2-step with HCB2Pair
- hcb10_1: **64 entries** (6 bits first step, not 5!)
- hcb10_2: 209 entries

**Step 1: Write the failing test**

```go
// internal/huffman/codebook_10_test.go
package huffman

import "testing"

func TestHCB10_1Size(t *testing.T) {
	// HCB_10 uses 6-bit first step (64 entries)
	if len(hcb10_1) != 64 {
		t.Errorf("hcb10_1 size = %d, want 64", len(hcb10_1))
	}
}

func TestHCB10_2Size(t *testing.T) {
	if len(hcb10_2) != 209 {
		t.Errorf("hcb10_2 size = %d, want 209", len(hcb10_2))
	}
}
```

**Commit message:**
```
feat(huffman): port codebook 10 (2-step pair, 6-bit first step)
```

---

## Task 12: Port Codebook 11 (2-Step Pair with Escape)

**Files:**
- Create: `internal/huffman/codebook_11.go`
- Create: `internal/huffman/codebook_11_test.go`

Pattern: 2-step with HCB2Pair (escape values are ±16)
- hcb11_1: 32 entries
- hcb11_2: 374 entries

**Commit message:**
```
feat(huffman): port codebook 11 (2-step pair with escape)
```

---

## Task 13: Port Scale Factor Codebook

**Files:**
- Create: `internal/huffman/codebook_sf.go`
- Create: `internal/huffman/codebook_sf_test.go`

**Step 1: Write the failing test**

```go
// internal/huffman/codebook_sf_test.go
package huffman

import "testing"

func TestHCBSFSize(t *testing.T) {
	if len(hcbSF) != 241 {
		t.Errorf("hcbSF size = %d, want 241", len(hcbSF))
	}
}

func TestHCBSFValues(t *testing.T) {
	// Scale factor codebook uses different format: [offset/value, is_leaf]
	tests := []struct {
		idx  int
		val0 uint8
		val1 uint8
	}{
		{0, 1, 2},    // Root: offsets 1, 2
		{1, 60, 0},   // Leaf: value 60
		{171, 0, 0},  // Leaf: value 0
	}
	for _, tt := range tests {
		if hcbSF[tt.idx][0] != tt.val0 || hcbSF[tt.idx][1] != tt.val1 {
			t.Errorf("hcbSF[%d] = {%d, %d}, want {%d, %d}",
				tt.idx, hcbSF[tt.idx][0], hcbSF[tt.idx][1],
				tt.val0, tt.val1)
		}
	}
}
```

**Step 3: Write minimal implementation**

```go
// internal/huffman/codebook_sf.go
package huffman

// hcbSF is the binary search table for scale factor decoding.
// Format: [241][2]uint8 where:
// - For internal nodes: [offset_if_0, offset_if_1]
// - For leaf nodes: [value, 0]
//
// Ported from: hcb_sf[241][2] in ~/dev/faad2/libfaad/codebook/hcb_sf.h:34-276
var hcbSF = [241][2]uint8{
	// ... copy exact values from hcb_sf.h
}
```

**Commit message:**
```
feat(huffman): port scale factor codebook

Port HCB_SF binary search table from FAAD2:
- hcbSF: 241-entry table for scale factor decoding
- Uses different format: [2]uint8 instead of struct
```

---

## Task 14: Create Codebook Lookup Tables

**Files:**
- Create: `internal/huffman/tables.go`
- Create: `internal/huffman/tables_test.go`

These tables map codebook index to the appropriate table for use by the decoder.

**Step 1: Write the failing test**

```go
// internal/huffman/tables_test.go
package huffman

import "testing"

func TestHCBFirstStepTable(t *testing.T) {
	// Verify first-step tables are properly indexed
	if HCBFirstStep[1] == nil {
		t.Error("HCBFirstStep[1] should not be nil")
	}
	if HCBFirstStep[3] != nil {
		t.Error("HCBFirstStep[3] should be nil (binary codebook)")
	}
}

func TestHCB2QuadTable(t *testing.T) {
	if HCB2Quad[1] == nil {
		t.Error("HCB2Quad[1] should not be nil")
	}
	if HCB2Quad[6] != nil {
		t.Error("HCB2Quad[6] should be nil (pair codebook)")
	}
}

func TestHCB2PairTable(t *testing.T) {
	if HCB2Pair[6] == nil {
		t.Error("HCB2Pair[6] should not be nil")
	}
	if HCB2Pair[1] != nil {
		t.Error("HCB2Pair[1] should be nil (quad codebook)")
	}
}

func TestHCBBinQuadTable(t *testing.T) {
	if HCBBinQuad[3] == nil {
		t.Error("HCBBinQuad[3] should not be nil")
	}
}

func TestHCBBinPairTable(t *testing.T) {
	if HCBBinPair[5] == nil {
		t.Error("HCBBinPair[5] should not be nil")
	}
}
```

**Step 3: Write minimal implementation**

```go
// internal/huffman/tables.go
package huffman

// HCBFirstStep maps codebook index to first-step lookup table.
// Only 2-step codebooks have entries; binary codebooks are nil.
var HCBFirstStep = [12][]HCB{
	nil,      // 0: ZERO_HCB
	hcb1_1[:], // 1
	hcb2_1[:], // 2
	nil,      // 3: binary
	hcb4_1[:], // 4
	nil,      // 5: binary
	hcb6_1[:], // 6
	nil,      // 7: binary
	hcb8_1[:], // 8
	nil,      // 9: binary
	hcb10_1[:], // 10
	hcb11_1[:], // 11
}

// HCBFirstStepBits maps codebook to number of first-step bits.
var HCBFirstStepBits = [12]uint{
	0, 5, 5, 0, 5, 0, 5, 0, 5, 0, 6, 5,
}

// HCB2Quad maps codebook index to second-step quad table.
var HCB2Quad = [12][]HCB2Quad{
	nil, hcb1_2[:], hcb2_2[:], nil, hcb4_2[:], nil, nil, nil, nil, nil, nil, nil,
}

// HCB2Pair maps codebook index to second-step pair table.
var HCB2Pair = [12][]HCB2Pair{
	nil, nil, nil, nil, nil, nil, hcb6_2[:], nil, hcb8_2[:], nil, hcb10_2[:], hcb11_2[:],
}

// HCBBinQuad maps codebook index to binary quad table.
var HCBBinQuad = [12][]HCBBinQuad{
	nil, nil, nil, hcb3[:], nil, nil, nil, nil, nil, nil, nil, nil,
}

// HCBBinPair maps codebook index to binary pair table.
var HCBBinPair = [12][]HCBBinPair{
	nil, nil, nil, nil, nil, hcb5[:], nil, hcb7[:], nil, hcb9[:], nil, nil,
}

// UnsignedCB indicates whether a codebook uses unsigned values.
// Ported from: unsigned_cb in ~/dev/faad2/libfaad/huffman.c
var UnsignedCB = [12]bool{
	false, false, false, true, true, false, false, true, true, true, true, true,
}
```

**Commit message:**
```
feat(huffman): add codebook lookup tables

Add lookup tables to map codebook index to data:
- HCBFirstStep: 2-step first-step tables
- HCBFirstStepBits: bits for first step
- HCB2Quad/HCB2Pair: 2-step second-step tables
- HCBBinQuad/HCBBinPair: binary search tables
- UnsignedCB: unsigned codebook flag
```

---

## Task 15: Add Comprehensive FAAD2 Validation Test

**Files:**
- Create: `internal/huffman/validate_test.go`

**Step 1: Write validation test**

```go
// internal/huffman/validate_test.go
package huffman

import (
	"testing"
)

// TestAllCodebookSizes validates that all codebook tables match FAAD2 sizes.
func TestAllCodebookSizes(t *testing.T) {
	sizes := []struct {
		name string
		got  int
		want int
	}{
		{"hcb1_1", len(hcb1_1), 32},
		{"hcb1_2", len(hcb1_2), 113},
		{"hcb2_1", len(hcb2_1), 32},
		{"hcb2_2", len(hcb2_2), 85},
		{"hcb3", len(hcb3), 161},
		{"hcb4_1", len(hcb4_1), 32},
		{"hcb4_2", len(hcb4_2), 184},
		{"hcb5", len(hcb5), 161},
		{"hcb6_1", len(hcb6_1), 32},
		{"hcb6_2", len(hcb6_2), 125},
		{"hcb7", len(hcb7), 127},
		{"hcb8_1", len(hcb8_1), 32},
		{"hcb8_2", len(hcb8_2), 83},
		{"hcb9", len(hcb9), 337},
		{"hcb10_1", len(hcb10_1), 64},
		{"hcb10_2", len(hcb10_2), 209},
		{"hcb11_1", len(hcb11_1), 32},
		{"hcb11_2", len(hcb11_2), 374},
		{"hcbSF", len(hcbSF), 241},
	}

	for _, tt := range sizes {
		if tt.got != tt.want {
			t.Errorf("%s size = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}

// TestCodebookConsistency verifies internal consistency of lookup tables.
func TestCodebookConsistency(t *testing.T) {
	// Verify all 2-step codebooks have matching first-step entries
	for cb := 1; cb <= 11; cb++ {
		if HCBFirstStep[cb] != nil {
			if HCBFirstStepBits[cb] == 0 {
				t.Errorf("Codebook %d has first-step table but zero bits", cb)
			}
			expectedLen := 1 << HCBFirstStepBits[cb]
			if len(HCBFirstStep[cb]) != expectedLen {
				t.Errorf("Codebook %d first-step has %d entries, expected %d",
					cb, len(HCBFirstStep[cb]), expectedLen)
			}
		}
	}
}
```

**Commit message:**
```
test(huffman): add comprehensive codebook validation

Add validation tests to ensure all codebook tables:
- Have correct sizes matching FAAD2
- Have consistent first-step/bits relationships
- Pass internal consistency checks
```

---

## Task 16: Final Cleanup and Documentation

**Files:**
- Update: `internal/huffman/constants.go` (add any missing constants)

**Step 1: Verify all tests pass**

Run: `make check`
Expected: All tests pass, linting clean

**Step 2: Update package documentation**

Add to `internal/huffman/constants.go`:

```go
// LastCBIndex is the highest valid spectral codebook index.
const LastCBIndex = 11
```

**Commit message:**
```
docs(huffman): finalize codebook implementation

- Add LastCBIndex constant
- Ensure all codebooks are documented
- All 12 codebook tables ported from FAAD2
```

---

## Summary

| Task | File | Description |
|------|------|-------------|
| 1 | types.go | Codebook type definitions |
| 2 | codebook_1.go | HCB_1 (2-step quad) |
| 3 | codebook_2.go | HCB_2 (2-step quad) |
| 4 | codebook_3.go | HCB_3 (binary quad) |
| 5 | codebook_4.go | HCB_4 (2-step quad) |
| 6 | codebook_5.go | HCB_5 (binary pair) |
| 7 | codebook_6.go | HCB_6 (2-step pair) |
| 8 | codebook_7.go | HCB_7 (binary pair) |
| 9 | codebook_8.go | HCB_8 (2-step pair) |
| 10 | codebook_9.go | HCB_9 (binary pair) |
| 11 | codebook_10.go | HCB_10 (2-step pair, 6-bit) |
| 12 | codebook_11.go | HCB_11 (2-step pair + escape) |
| 13 | codebook_sf.go | Scale factor codebook |
| 14 | tables.go | Codebook lookup tables |
| 15 | validate_test.go | FAAD2 validation tests |
| 16 | cleanup | Final documentation |

**Total: 16 tasks, ~15 files, ~2500 lines of code**
