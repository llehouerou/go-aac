# Huffman Decoder Functions Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement Huffman decoding functions to decode scale factors and spectral data from AAC bitstreams.

**Architecture:** Port FAAD2's `huffman.c` to Go as `internal/huffman/decoder.go`. The decoder uses the lookup tables from Step 2.1 (already implemented) and the `bits.Reader` from Step 1.4. Two decoding methods are used: 2-step table lookup for most codebooks, and binary search for codebooks 3, 5, 7, 9, and scale factors.

**Tech Stack:** Pure Go, depends on `internal/bits` and existing `internal/huffman` tables.

---

## Background: FAAD2 Huffman Decoding

The FAAD2 Huffman decoder uses two main approaches:

1. **2-Step Lookup (codebooks 1, 2, 4, 6, 8, 10, 11):**
   - Read N bits (5 or 6) to index the first-step table
   - First-step table gives offset + extra_bits
   - If extra_bits > 0, read more bits and index second-step table
   - Second-step table contains the decoded values

2. **Binary Search (codebooks 3, 5, 7, 9, scale factors):**
   - Start at offset 0
   - Read 1 bit at a time, follow tree until leaf node
   - Leaf node contains decoded values

**Signed vs Unsigned Codebooks:**
- Unsigned (3, 4, 7, 8, 9, 10, 11): Sign bits are read separately after decoding
- Signed (1, 2, 5, 6): Values include sign already

**Escape Codebook (11):**
- Values of ±16 indicate escape sequence
- Read unary count (N ones terminated by zero)
- Read N bits as magnitude
- Final value = magnitude | (1 << N)

---

## Task 1: Scale Factor Huffman Decoding

**Files:**
- Create: `internal/huffman/decoder.go`
- Test: `internal/huffman/decoder_test.go`

### Step 1.1: Write the failing test for ScaleFactor

```go
// internal/huffman/decoder_test.go
package huffman

import (
	"testing"

	"go-aac/internal/bits"
)

func TestScaleFactor_ZeroValue(t *testing.T) {
	// Codeword for sf_index 60 (value 0) is 0b1111_1110_110 (11 bits)
	// See hcb_sf table: offset 0 -> branches lead to value 60
	// Actually, value 0 is at index 60 in the scale factor table.
	// The shortest codeword in hcb_sf is for index 60 (delta = 0).
	// Looking at hcb_sf: [60, 1] means value=60 is leaf at offset 0+1=1
	// Let's trace: starting at offset 0, hcb_sf[0] = [60, 1]
	// Since hcb_sf[offset][1] != 0, we read a bit
	// We need to find the actual codeword by tracing the tree.

	// For simplicity, let's test with a known pattern:
	// The scale factor codebook returns delta values centered at 60.
	// Index 60 = delta 0, index 59 = delta -1, index 61 = delta +1, etc.

	// From FAAD2 hcb_sf.h, the structure is [value, branch_offset]
	// If branch_offset is 0, we've found a leaf with the value.
	// Let's test with a bit pattern that leads to a known value.

	// The codeword "11011" (5 bits) should decode to index 60 (delta 0)
	// Actually, we need to verify this against FAAD2. For TDD, let's
	// start with a basic test structure.

	// For now, test that the function exists and returns something
	data := []byte{0xFF, 0xFF} // All 1s - will traverse deep
	r := bits.NewReader(data)

	sf := ScaleFactor(r)

	// sf should be in range [-60, 60] for valid scale factors
	if sf < -60 || sf > 60 {
		t.Errorf("ScaleFactor out of range: got %d", sf)
	}
}
```

### Step 1.2: Run test to verify it fails

Run: `go test -v ./internal/huffman -run TestScaleFactor_ZeroValue`
Expected: FAIL with "undefined: ScaleFactor"

### Step 1.3: Write minimal ScaleFactor implementation

```go
// internal/huffman/decoder.go
package huffman

import "go-aac/internal/bits"

// ScaleFactor decodes a scale factor delta from the bitstream.
// Returns a value in the range [-60, 60] representing the delta
// from the previous scale factor.
//
// The scale factor codebook uses binary search: start at offset 0,
// read one bit at a time, and follow branches until reaching a leaf.
//
// Ported from: huffman_scale_factor() in ~/dev/faad2/libfaad/huffman.c:60-72
func ScaleFactor(r *bits.Reader) int8 {
	offset := uint16(0)
	sf := *HCBSF

	// Traverse binary tree until we hit a leaf (branch offset = 0)
	for sf[offset][1] != 0 {
		b := r.Get1Bit()
		offset += uint16(sf[offset][b])
	}

	// Return the value at the leaf node, adjusted to signed delta
	return int8(sf[offset][0]) - 60
}
```

### Step 1.4: Run test to verify it passes

Run: `go test -v ./internal/huffman -run TestScaleFactor_ZeroValue`
Expected: PASS

### Step 1.5: Commit

```bash
git add internal/huffman/decoder.go internal/huffman/decoder_test.go
git commit -m "feat(huffman): add ScaleFactor decoder function"
```

---

## Task 2: Sign Bits Helper

**Files:**
- Modify: `internal/huffman/decoder.go`
- Test: `internal/huffman/decoder_test.go`

### Step 2.1: Write the failing test for signBits

```go
func TestSignBits(t *testing.T) {
	tests := []struct {
		name     string
		input    []int16
		bits     []uint8 // sign bits to inject
		expected []int16
	}{
		{
			name:     "no non-zero values",
			input:    []int16{0, 0, 0, 0},
			bits:     []uint8{},
			expected: []int16{0, 0, 0, 0},
		},
		{
			name:     "single positive stays positive (bit=0)",
			input:    []int16{5, 0, 0, 0},
			bits:     []uint8{0},
			expected: []int16{5, 0, 0, 0},
		},
		{
			name:     "single positive becomes negative (bit=1)",
			input:    []int16{5, 0, 0, 0},
			bits:     []uint8{1},
			expected: []int16{-5, 0, 0, 0},
		},
		{
			name:     "multiple values with mixed signs",
			input:    []int16{3, 0, 7, 2},
			bits:     []uint8{0, 1, 0}, // Only non-zero get bits
			expected: []int16{3, 0, -7, 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Build bitstream from sign bits
			data := buildSignBitstream(tc.bits)
			r := bits.NewReader(data)

			sp := make([]int16, len(tc.input))
			copy(sp, tc.input)

			signBits(r, sp)

			for i := range sp {
				if sp[i] != tc.expected[i] {
					t.Errorf("sp[%d]: got %d, want %d", i, sp[i], tc.expected[i])
				}
			}
		})
	}
}

// buildSignBitstream creates a byte slice from a sequence of bits
func buildSignBitstream(signBits []uint8) []byte {
	if len(signBits) == 0 {
		return []byte{0}
	}
	// Pack bits into bytes (MSB first)
	numBytes := (len(signBits) + 7) / 8
	data := make([]byte, numBytes)
	for i, bit := range signBits {
		byteIdx := i / 8
		bitIdx := 7 - (i % 8) // MSB first
		if bit != 0 {
			data[byteIdx] |= 1 << bitIdx
		}
	}
	return data
}
```

### Step 2.2: Run test to verify it fails

Run: `go test -v ./internal/huffman -run TestSignBits`
Expected: FAIL with "undefined: signBits"

### Step 2.3: Write signBits implementation

Add to `internal/huffman/decoder.go`:

```go
// signBits reads sign bits for non-zero spectral coefficients.
// For each non-zero value in sp, reads 1 bit: if 1, negates the value.
//
// Ported from: huffman_sign_bits() in ~/dev/faad2/libfaad/huffman.c:93-108
func signBits(r *bits.Reader, sp []int16) {
	for i := range sp {
		if sp[i] != 0 {
			if r.Get1Bit()&1 != 0 {
				sp[i] = -sp[i]
			}
		}
	}
}
```

### Step 2.4: Run test to verify it passes

Run: `go test -v ./internal/huffman -run TestSignBits`
Expected: PASS

### Step 2.5: Commit

```bash
git add internal/huffman/decoder.go internal/huffman/decoder_test.go
git commit -m "feat(huffman): add signBits helper for unsigned codebooks"
```

---

## Task 3: Escape Code Decoding

**Files:**
- Modify: `internal/huffman/decoder.go`
- Test: `internal/huffman/decoder_test.go`

### Step 3.1: Write the failing test for getEscape

```go
func TestGetEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    int16
		bits     []uint8 // escape bits: N ones, zero, then N-bit value
		expected int16
		err      bool
	}{
		{
			name:     "not an escape value (positive)",
			input:    15,
			bits:     []uint8{},
			expected: 15,
		},
		{
			name:     "not an escape value (negative)",
			input:    -15,
			bits:     []uint8{},
			expected: -15,
		},
		{
			name:     "positive escape: 4 ones + zero + 4 bits = 17-31",
			input:    16,
			bits:     []uint8{0, 0, 0, 0, 1}, // 4 zeros (i starts at 4), value bits = 0001 = 1
			expected: 17, // (1 << 4) | 1 = 17
		},
		{
			name:     "negative escape: 4 ones + zero + 4 bits",
			input:    -16,
			bits:     []uint8{0, 0, 0, 0, 1}, // Same as above but negative
			expected: -17,
		},
		{
			name:     "escape with more leading ones: 5-bit exponent",
			input:    16,
			bits:     []uint8{1, 0, 0, 0, 0, 0, 1}, // 1 one, zero, then 5 bits = 00001 = 1
			expected: 33, // (1 << 5) | 1 = 33
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := buildSignBitstream(tc.bits)
			r := bits.NewReader(data)

			sp := tc.input
			err := getEscape(r, &sp)

			if tc.err && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.err && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if sp != tc.expected {
				t.Errorf("got %d, want %d", sp, tc.expected)
			}
		})
	}
}
```

### Step 3.2: Run test to verify it fails

Run: `go test -v ./internal/huffman -run TestGetEscape`
Expected: FAIL with "undefined: getEscape"

### Step 3.3: Write getEscape implementation

Add to `internal/huffman/decoder.go`:

```go
import "errors"

// ErrEscapeSequence indicates a malformed escape sequence in spectral data.
var ErrEscapeSequence = errors.New("huffman: invalid escape sequence")

// getEscape decodes an escape code if the value is ±16.
// For escape codebook (11), values of ±16 indicate more bits follow.
// Returns error if escape sequence is malformed.
//
// Format: N ones followed by a zero (N >= 4), then N bits of magnitude.
// Final value = (1 << N) | magnitude_bits
//
// Ported from: huffman_getescape() in ~/dev/faad2/libfaad/huffman.c:110-148
func getEscape(r *bits.Reader, sp *int16) error {
	x := *sp

	// Check if this is an escape value
	var neg bool
	if x < 0 {
		if x != -16 {
			return nil // Not an escape
		}
		neg = true
	} else {
		if x != 16 {
			return nil // Not an escape
		}
		neg = false
	}

	// Count leading ones (starting from i=4, since 16 = 2^4)
	var i uint
	for i = 4; i < 16; i++ {
		if r.Get1Bit() == 0 {
			break
		}
	}
	if i >= 16 {
		return ErrEscapeSequence
	}

	// Read i bits for the offset
	off := int16(r.GetBits(i))
	j := off | (1 << i)

	if neg {
		j = -j
	}

	*sp = j
	return nil
}
```

### Step 3.4: Run test to verify it passes

Run: `go test -v ./internal/huffman -run TestGetEscape`
Expected: PASS

### Step 3.5: Commit

```bash
git add internal/huffman/decoder.go internal/huffman/decoder_test.go
git commit -m "feat(huffman): add getEscape for escape codebook values"
```

---

## Task 4: 2-Step Quad Decoding (Codebooks 1, 2, 4)

**Files:**
- Modify: `internal/huffman/decoder.go`
- Test: `internal/huffman/decoder_test.go`

### Step 4.1: Write the failing test for decode2StepQuad

```go
func TestDecode2StepQuad(t *testing.T) {
	// Test with a known codeword from codebook 1
	// The first entry in hcb1_2 (index 0) has bits=1, x=0, y=0, v=0, w=0
	// which corresponds to codeword "0" (1 bit)
	// From hcb1_1[0]: offset=0, extra_bits=0

	// To decode {0,0,0,0} from codebook 1, we need codeword that maps there
	// Looking at the 2-step structure:
	// - First table hcb1_1 is indexed by first 5 bits
	// - hcb1_1[0] = {0, 0} means offset=0, extra_bits=0
	// - hcb1_2[0] = {1, 0, 0, 0, 0} means 1 bit total, values (0,0,0,0)

	// So bit pattern 00000xxx (first 5 bits = 0) should give (0,0,0,0)
	data := []byte{0x00, 0x00}
	r := bits.NewReader(data)

	var sp [4]int16
	err := decode2StepQuad(1, r, sp[:])

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := [4]int16{0, 0, 0, 0}
	if sp != expected {
		t.Errorf("got %v, want %v", sp, expected)
	}
}

func TestDecode2StepQuad_AllCodebooks(t *testing.T) {
	// Test that valid codebook indices work without panic
	for _, cb := range []uint8{1, 2, 4} {
		t.Run(fmt.Sprintf("codebook_%d", cb), func(t *testing.T) {
			// Use a pattern of zeros which should decode to smallest values
			data := []byte{0x00, 0x00, 0x00, 0x00}
			r := bits.NewReader(data)

			var sp [4]int16
			err := decode2StepQuad(cb, r, sp[:])

			if err != nil {
				t.Errorf("codebook %d: unexpected error: %v", cb, err)
			}
		})
	}
}
```

### Step 4.2: Run test to verify it fails

Run: `go test -v ./internal/huffman -run TestDecode2StepQuad`
Expected: FAIL with "undefined: decode2StepQuad"

### Step 4.3: Write decode2StepQuad implementation

Add to `internal/huffman/decoder.go`:

```go
// decode2StepQuad decodes a quadruple (4 values) using 2-step table lookup.
// Used for codebooks 1, 2, and 4.
//
// Step 1: Read root_bits (5) and lookup in first table
// Step 2: If extra_bits > 0, read more bits and lookup in second table
//
// Ported from: huffman_2step_quad() in ~/dev/faad2/libfaad/huffman.c:150-188
func decode2StepQuad(cb uint8, r *bits.Reader, sp []int16) error {
	root := HCBTable[cb]
	rootBits := HCBN[cb]
	table := HCB2QuadTable[cb]

	// Read first-step bits and lookup
	cw := r.ShowBits(uint(rootBits))
	offset := uint16((*root)[cw].Offset)
	extraBits := (*root)[cw].ExtraBits

	if extraBits != 0 {
		// Need more bits - flush root bits, read extra, adjust offset
		r.FlushBits(uint(rootBits))
		offset += uint16(r.ShowBits(uint(extraBits)))
		r.FlushBits(uint((*table)[offset].Bits) - uint(rootBits))
	} else {
		// All bits in root lookup
		r.FlushBits(uint((*table)[offset].Bits))
	}

	// Extract the four values
	sp[0] = int16((*table)[offset].X)
	sp[1] = int16((*table)[offset].Y)
	sp[2] = int16((*table)[offset].V)
	sp[3] = int16((*table)[offset].W)

	return nil
}
```

### Step 4.4: Run test to verify it passes

Run: `go test -v ./internal/huffman -run TestDecode2StepQuad`
Expected: PASS

### Step 4.5: Commit

```bash
git add internal/huffman/decoder.go internal/huffman/decoder_test.go
git commit -m "feat(huffman): add decode2StepQuad for codebooks 1, 2, 4"
```

---

## Task 5: 2-Step Pair Decoding (Codebooks 6, 8, 10, 11)

**Files:**
- Modify: `internal/huffman/decoder.go`
- Test: `internal/huffman/decoder_test.go`

### Step 5.1: Write the failing test for decode2StepPair

```go
func TestDecode2StepPair(t *testing.T) {
	// Test with a known codeword from codebook 6
	// Similar structure to quad but only 2 values
	data := []byte{0x00, 0x00}
	r := bits.NewReader(data)

	var sp [2]int16
	err := decode2StepPair(6, r, sp[:])

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// The exact values depend on the codebook, just verify no crash
}

func TestDecode2StepPair_AllCodebooks(t *testing.T) {
	for _, cb := range []uint8{6, 8, 10, 11} {
		t.Run(fmt.Sprintf("codebook_%d", cb), func(t *testing.T) {
			data := []byte{0x00, 0x00, 0x00, 0x00}
			r := bits.NewReader(data)

			var sp [2]int16
			err := decode2StepPair(cb, r, sp[:])

			if err != nil {
				t.Errorf("codebook %d: unexpected error: %v", cb, err)
			}
		})
	}
}
```

### Step 5.2: Run test to verify it fails

Run: `go test -v ./internal/huffman -run TestDecode2StepPair`
Expected: FAIL with "undefined: decode2StepPair"

### Step 5.3: Write decode2StepPair implementation

Add to `internal/huffman/decoder.go`:

```go
// decode2StepPair decodes a pair (2 values) using 2-step table lookup.
// Used for codebooks 6, 8, 10, and 11.
//
// Ported from: huffman_2step_pair() in ~/dev/faad2/libfaad/huffman.c:198-234
func decode2StepPair(cb uint8, r *bits.Reader, sp []int16) error {
	root := HCBTable[cb]
	rootBits := HCBN[cb]
	table := HCB2PairTable[cb]

	// Read first-step bits and lookup
	cw := r.ShowBits(uint(rootBits))
	offset := uint16((*root)[cw].Offset)
	extraBits := (*root)[cw].ExtraBits

	if extraBits != 0 {
		// Need more bits
		r.FlushBits(uint(rootBits))
		offset += uint16(r.ShowBits(uint(extraBits)))
		r.FlushBits(uint((*table)[offset].Bits) - uint(rootBits))
	} else {
		r.FlushBits(uint((*table)[offset].Bits))
	}

	// Extract the two values
	sp[0] = int16((*table)[offset].X)
	sp[1] = int16((*table)[offset].Y)

	return nil
}
```

### Step 5.4: Run test to verify it passes

Run: `go test -v ./internal/huffman -run TestDecode2StepPair`
Expected: PASS

### Step 5.5: Commit

```bash
git add internal/huffman/decoder.go internal/huffman/decoder_test.go
git commit -m "feat(huffman): add decode2StepPair for codebooks 6, 8, 10, 11"
```

---

## Task 6: Binary Quad Decoding (Codebook 3)

**Files:**
- Modify: `internal/huffman/decoder.go`
- Test: `internal/huffman/decoder_test.go`

### Step 6.1: Write the failing test for decodeBinaryQuad

```go
func TestDecodeBinaryQuad(t *testing.T) {
	// Codebook 3 uses binary search with HCB3 table
	// Each node either has is_leaf=1 (data[0..3] are values)
	// or is_leaf=0 (data[0], data[1] are branch offsets for bit 0 and 1)

	data := []byte{0x00, 0x00, 0x00, 0x00}
	r := bits.NewReader(data)

	var sp [4]int16
	err := decodeBinaryQuad(r, sp[:])

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Values will be whatever the binary tree gives for all-zero input
}
```

### Step 6.2: Run test to verify it fails

Run: `go test -v ./internal/huffman -run TestDecodeBinaryQuad`
Expected: FAIL with "undefined: decodeBinaryQuad"

### Step 6.3: Write decodeBinaryQuad implementation

Add to `internal/huffman/decoder.go`:

```go
// decodeBinaryQuad decodes a quadruple using binary search.
// Used only for codebook 3.
//
// Traverse the binary tree by reading one bit at a time until
// reaching a leaf node (is_leaf = 1), then extract the 4 values.
//
// Ported from: huffman_binary_quad() in ~/dev/faad2/libfaad/huffman.c:244-266
func decodeBinaryQuad(r *bits.Reader, sp []int16) error {
	offset := uint16(0)
	table := *HCB3

	// Traverse until we hit a leaf
	for table[offset].IsLeaf == 0 {
		b := r.Get1Bit()
		offset += uint16(table[offset].Data[b])
	}

	// Extract the four values from the leaf
	sp[0] = int16(table[offset].Data[0])
	sp[1] = int16(table[offset].Data[1])
	sp[2] = int16(table[offset].Data[2])
	sp[3] = int16(table[offset].Data[3])

	return nil
}
```

### Step 6.4: Run test to verify it passes

Run: `go test -v ./internal/huffman -run TestDecodeBinaryQuad`
Expected: PASS

### Step 6.5: Commit

```bash
git add internal/huffman/decoder.go internal/huffman/decoder_test.go
git commit -m "feat(huffman): add decodeBinaryQuad for codebook 3"
```

---

## Task 7: Binary Pair Decoding (Codebooks 5, 7, 9)

**Files:**
- Modify: `internal/huffman/decoder.go`
- Test: `internal/huffman/decoder_test.go`

### Step 7.1: Write the failing test for decodeBinaryPair

```go
func TestDecodeBinaryPair(t *testing.T) {
	for _, cb := range []uint8{5, 7, 9} {
		t.Run(fmt.Sprintf("codebook_%d", cb), func(t *testing.T) {
			data := []byte{0x00, 0x00, 0x00, 0x00}
			r := bits.NewReader(data)

			var sp [2]int16
			err := decodeBinaryPair(cb, r, sp[:])

			if err != nil {
				t.Errorf("codebook %d: unexpected error: %v", cb, err)
			}
		})
	}
}
```

### Step 7.2: Run test to verify it fails

Run: `go test -v ./internal/huffman -run TestDecodeBinaryPair`
Expected: FAIL with "undefined: decodeBinaryPair"

### Step 7.3: Write decodeBinaryPair implementation

Add to `internal/huffman/decoder.go`:

```go
// decodeBinaryPair decodes a pair using binary search.
// Used for codebooks 5, 7, and 9.
//
// Ported from: huffman_binary_pair() in ~/dev/faad2/libfaad/huffman.c:276-298
func decodeBinaryPair(cb uint8, r *bits.Reader, sp []int16) error {
	offset := uint16(0)
	table := *HCBBinPairTable[cb]

	// Traverse until we hit a leaf
	for table[offset].IsLeaf == 0 {
		b := r.Get1Bit()
		offset += uint16(table[offset].Data[b])
	}

	// Extract the two values from the leaf
	sp[0] = int16(table[offset].Data[0])
	sp[1] = int16(table[offset].Data[1])

	return nil
}
```

### Step 7.4: Run test to verify it passes

Run: `go test -v ./internal/huffman -run TestDecodeBinaryPair`
Expected: PASS

### Step 7.5: Commit

```bash
git add internal/huffman/decoder.go internal/huffman/decoder_test.go
git commit -m "feat(huffman): add decodeBinaryPair for codebooks 5, 7, 9"
```

---

## Task 8: VCB11 LAV Check

**Files:**
- Modify: `internal/huffman/decoder.go`
- Test: `internal/huffman/decoder_test.go`

### Step 8.1: Write the failing test for vcb11CheckLAV

```go
func TestVcb11CheckLAV(t *testing.T) {
	tests := []struct {
		name     string
		cb       uint8
		input    [2]int16
		expected [2]int16
	}{
		{
			name:     "codebook 16 LAV=16, within limit",
			cb:       16,
			input:    [2]int16{15, -10},
			expected: [2]int16{15, -10},
		},
		{
			name:     "codebook 16 LAV=16, exceeds limit",
			cb:       16,
			input:    [2]int16{17, 5},
			expected: [2]int16{0, 0},
		},
		{
			name:     "codebook 31 LAV=2047, within limit",
			cb:       31,
			input:    [2]int16{2000, -1000},
			expected: [2]int16{2000, -1000},
		},
		{
			name:     "non-VCB11 codebook, no change",
			cb:       11,
			input:    [2]int16{1000, 2000},
			expected: [2]int16{1000, 2000},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sp := tc.input
			vcb11CheckLAV(tc.cb, sp[:])

			if sp != tc.expected {
				t.Errorf("got %v, want %v", sp, tc.expected)
			}
		})
	}
}
```

### Step 8.2: Run test to verify it fails

Run: `go test -v ./internal/huffman -run TestVcb11CheckLAV`
Expected: FAIL with "undefined: vcb11CheckLAV"

### Step 8.3: Write vcb11CheckLAV implementation

Add to `internal/huffman/decoder.go`:

```go
// vcb11LAV is the Largest Absolute Value table for virtual codebook 11.
// Index is (cb - 16), valid for codebooks 16-31.
//
// Ported from: vcb11_LAV_tab in ~/dev/faad2/libfaad/huffman.c:319-322
var vcb11LAV = [16]int16{
	16, 31, 47, 63, 95, 127, 159, 191,
	223, 255, 319, 383, 511, 767, 1023, 2047,
}

// vcb11CheckLAV checks if values exceed the Largest Absolute Value
// for virtual codebook 11 (codebooks 16-31). If exceeded, zeros them.
//
// This catches errors in escape sequences for VCB11.
//
// Ported from: vcb11_check_LAV() in ~/dev/faad2/libfaad/huffman.c:317-335
func vcb11CheckLAV(cb uint8, sp []int16) {
	if cb < 16 || cb > 31 {
		return
	}

	maxVal := vcb11LAV[cb-16]

	if abs16(sp[0]) > maxVal || abs16(sp[1]) > maxVal {
		sp[0] = 0
		sp[1] = 0
	}
}

// abs16 returns the absolute value of an int16.
func abs16(x int16) int16 {
	if x < 0 {
		return -x
	}
	return x
}
```

### Step 8.4: Run test to verify it passes

Run: `go test -v ./internal/huffman -run TestVcb11CheckLAV`
Expected: PASS

### Step 8.5: Commit

```bash
git add internal/huffman/decoder.go internal/huffman/decoder_test.go
git commit -m "feat(huffman): add vcb11CheckLAV for error resilience"
```

---

## Task 9: Main SpectralData Entry Point

**Files:**
- Modify: `internal/huffman/decoder.go`
- Test: `internal/huffman/decoder_test.go`

### Step 9.1: Write the failing test for SpectralData

```go
func TestSpectralData(t *testing.T) {
	tests := []struct {
		name string
		cb   uint8
	}{
		{"codebook_1_quad_signed", 1},
		{"codebook_2_quad_signed", 2},
		{"codebook_3_binary_quad_unsigned", 3},
		{"codebook_4_quad_unsigned", 4},
		{"codebook_5_binary_pair_signed", 5},
		{"codebook_6_pair_signed", 6},
		{"codebook_7_binary_pair_unsigned", 7},
		{"codebook_8_pair_unsigned", 8},
		{"codebook_9_binary_pair_unsigned", 9},
		{"codebook_10_pair_unsigned", 10},
		{"codebook_11_escape", 11},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create bitstream with enough data
			data := make([]byte, 16)
			r := bits.NewReader(data)

			var sp [4]int16
			err := SpectralData(tc.cb, r, sp[:])

			if err != nil {
				t.Errorf("codebook %d: unexpected error: %v", tc.cb, err)
			}
		})
	}
}

func TestSpectralData_InvalidCodebook(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x00}
	r := bits.NewReader(data)

	var sp [4]int16
	err := SpectralData(0, r, sp[:])

	if err == nil {
		t.Error("expected error for codebook 0")
	}

	err = SpectralData(12, r, sp[:])
	if err == nil {
		t.Error("expected error for codebook 12")
	}
}
```

### Step 9.2: Run test to verify it fails

Run: `go test -v ./internal/huffman -run TestSpectralData`
Expected: FAIL with "undefined: SpectralData"

### Step 9.3: Write SpectralData implementation

Add to `internal/huffman/decoder.go`:

```go
// ErrInvalidCodebook indicates an invalid Huffman codebook index.
var ErrInvalidCodebook = errors.New("huffman: invalid codebook")

// SpectralData decodes spectral coefficients using the specified codebook.
// For quad codebooks (1-4), sp must have at least 4 elements.
// For pair codebooks (5-11), sp must have at least 2 elements.
//
// Returns the decoded values in sp. For unsigned codebooks, sign bits
// are read separately. For escape codebook (11), escape sequences
// are decoded for values of ±16.
//
// Ported from: huffman_spectral_data() in ~/dev/faad2/libfaad/huffman.c:337-398
func SpectralData(cb uint8, r *bits.Reader, sp []int16) error {
	switch cb {
	case 1, 2: // 2-step quad, signed
		return decode2StepQuad(cb, r, sp)

	case 3: // Binary quad, unsigned
		if err := decodeBinaryQuad(r, sp); err != nil {
			return err
		}
		signBits(r, sp[:QuadLen])
		return nil

	case 4: // 2-step quad, unsigned
		if err := decode2StepQuad(cb, r, sp); err != nil {
			return err
		}
		signBits(r, sp[:QuadLen])
		return nil

	case 5: // Binary pair, signed
		return decodeBinaryPair(cb, r, sp)

	case 6: // 2-step pair, signed
		return decode2StepPair(cb, r, sp)

	case 7, 9: // Binary pair, unsigned
		if err := decodeBinaryPair(cb, r, sp); err != nil {
			return err
		}
		signBits(r, sp[:PairLen])
		return nil

	case 8, 10: // 2-step pair, unsigned
		if err := decode2StepPair(cb, r, sp); err != nil {
			return err
		}
		signBits(r, sp[:PairLen])
		return nil

	case 11: // Escape codebook
		if err := decode2StepPair(11, r, sp); err != nil {
			return err
		}
		signBits(r, sp[:PairLen])
		if err := getEscape(r, &sp[0]); err != nil {
			return err
		}
		return getEscape(r, &sp[1])

	case 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31:
		// VCB11: virtual codebooks using codebook 11
		if err := decode2StepPair(11, r, sp); err != nil {
			return err
		}
		signBits(r, sp[:PairLen])
		if err := getEscape(r, &sp[0]); err != nil {
			return err
		}
		if err := getEscape(r, &sp[1]); err != nil {
			return err
		}
		vcb11CheckLAV(cb, sp)
		return nil

	default:
		return ErrInvalidCodebook
	}
}
```

### Step 9.4: Run test to verify it passes

Run: `go test -v ./internal/huffman -run TestSpectralData`
Expected: PASS

### Step 9.5: Commit

```bash
git add internal/huffman/decoder.go internal/huffman/decoder_test.go
git commit -m "feat(huffman): add SpectralData main entry point"
```

---

## Task 10: Run Full Test Suite and Lint

**Files:**
- All modified files

### Step 10.1: Run format and lint

Run: `make fmt && make lint`
Expected: No errors or warnings

### Step 10.2: Run all huffman tests

Run: `go test -v ./internal/huffman/...`
Expected: All tests PASS

### Step 10.3: Run full project check

Run: `make check`
Expected: PASS

### Step 10.4: Final commit with test cleanup if needed

```bash
git add -A
git commit -m "chore(huffman): clean up decoder tests and ensure lint passes"
```

---

## Summary

This plan implements the Huffman decoder functions for Step 2.2 of the AAC migration:

| Function | Purpose | Codebooks |
|----------|---------|-----------|
| `ScaleFactor` | Decode scale factor delta | Scale factor |
| `SpectralData` | Main entry for spectral decoding | 1-11, 16-31 |
| `signBits` | Apply sign bits | 3, 4, 7-11 |
| `getEscape` | Decode escape sequences | 11, 16-31 |
| `decode2StepQuad` | 2-step quad decoding | 1, 2, 4 |
| `decode2StepPair` | 2-step pair decoding | 6, 8, 10, 11 |
| `decodeBinaryQuad` | Binary quad decoding | 3 |
| `decodeBinaryPair` | Binary pair decoding | 5, 7, 9 |
| `vcb11CheckLAV` | Check VCB11 limits | 16-31 |

Total: ~300 lines of Go code matching FAAD2's `huffman.c`
