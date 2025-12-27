# Bit Reader Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a bitstream reader for parsing AAC encoded data, ported from FAAD2's bits.c/bits.h

**Architecture:** Two-buffer design (bufa + bufb) where bufa holds current 32 bits being consumed and bufb holds the next 32 bits pre-loaded. This allows reading up to 32 bits without buffer reload. Data is loaded in big-endian order (MSB first). The `bits_left` field tracks unread bits remaining in bufa (starts at 32, decrements as bits are consumed).

**Tech Stack:** Pure Go, no external dependencies

---

## FAAD2 Source Reference

- **Source files**: `~/dev/faad2/libfaad/bits.c` (292 lines), `~/dev/faad2/libfaad/bits.h` (422 lines)
- **Key data structure**: `bitfile` struct (bits.h:48-60)
- **Key functions**:
  - `faad_initbits()` - Initialize from byte buffer
  - `faad_showbits()` - Peek bits without consuming (inline, bits.h:102-113)
  - `faad_flushbits()` - Skip bits (inline, bits.h:115-127)
  - `faad_getbits()` - Read and consume bits (bits.h:130-146)
  - `faad_get1bit()` - Optimized single bit read (bits.h:148-167)
  - `faad_byte_align()` - Align to byte boundary (bits.c:111-121)
  - `faad_get_processed_bits()` - Get current bit position (bits.c:106-109)
  - `faad_getbitbuffer()` - Read raw bytes (bits.c:222-245)
  - `faad_flushbits_ex()` - Extended flush when crossing buffer boundary (bits.c:123-144)
  - `faad_resetbits()` - Seek to bit position (bits.c:180-220)

---

## Design Decisions

### 1. Go-Idiomatic API

```go
// FAAD2 C style:
// faad_initbits(&ld, buffer, size)
// bits = faad_getbits(&ld, n)

// Go style:
// r := bits.NewReader(data)
// bits := r.GetBits(n)
```

### 2. Error Handling

FAAD2 uses an `error` field that becomes non-zero on buffer overrun. We'll use the same approach but provide a method to check error state, with option to return errors from methods if preferred.

### 3. No Pointer Arithmetic

FAAD2 uses `tail`, `start` pointers for buffer navigation. We'll use slice indices instead.

### 4. Simplified Memory Model

FAAD2 uses `void*` buffer + byte tracking. Go's slices handle bounds naturally.

---

## Task 1: Create Reader Type and Constructor

**Files:**
- Create: `internal/bits/reader.go`
- Test: `internal/bits/reader_test.go`

**Step 1: Write the failing test for NewReader**

```go
// internal/bits/reader_test.go
package bits

import "testing"

func TestNewReader_BasicInit(t *testing.T) {
	data := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
	r := NewReader(data)

	if r == nil {
		t.Fatal("NewReader returned nil")
	}
	if r.Error() {
		t.Error("NewReader set error flag unexpectedly")
	}
	if r.BitsLeft() != 32 {
		t.Errorf("BitsLeft = %d, want 32", r.BitsLeft())
	}
}

func TestNewReader_EmptyBuffer(t *testing.T) {
	r := NewReader(nil)
	if !r.Error() {
		t.Error("NewReader(nil) should set error flag")
	}

	r = NewReader([]byte{})
	if !r.Error() {
		t.Error("NewReader([]) should set error flag")
	}
}

func TestNewReader_SmallBuffer(t *testing.T) {
	// Less than 4 bytes should still work
	r := NewReader([]byte{0x12, 0x34})
	if r.Error() {
		t.Error("NewReader with 2 bytes should not error")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestNewReader`
Expected: FAIL - Reader type and NewReader not defined

**Step 3: Write minimal implementation**

```go
// internal/bits/reader.go
package bits

// Reader reads bits from a byte buffer.
//
// Ported from: bitfile struct in ~/dev/faad2/libfaad/bits.h:48-60
type Reader struct {
	buffer     []byte // Original buffer
	bufa       uint32 // Current 32-bit buffer (high bits)
	bufb       uint32 // Next 32-bit buffer (look-ahead)
	bitsLeft   uint32 // Bits remaining in bufa (1-32)
	pos        int    // Current byte position in buffer (next to load)
	bufferSize int    // Total buffer size in bytes
	err        bool   // Error flag (buffer overrun)
}

// NewReader creates a Reader from a byte slice.
//
// Ported from: faad_initbits() in ~/dev/faad2/libfaad/bits.c:55-99
func NewReader(data []byte) *Reader {
	r := &Reader{
		buffer:     data,
		bufferSize: len(data),
	}

	if len(data) == 0 {
		r.err = true
		return r
	}

	// Load first 32-bit word into bufa
	r.bufa = r.loadWord(0)
	// Load second 32-bit word into bufb
	r.bufb = r.loadWord(4)
	// Track position (next word to load would be at byte 8)
	r.pos = 8
	r.bitsLeft = 32

	return r
}

// loadWord loads up to 4 bytes from buffer position as big-endian uint32.
// Handles partial reads at end of buffer.
//
// Ported from: getdword() and getdword_n() in bits.h:96-100, bits.c:38-52
func (r *Reader) loadWord(offset int) uint32 {
	if offset >= len(r.buffer) {
		return 0
	}

	remaining := len(r.buffer) - offset
	if remaining >= 4 {
		// Full 4-byte read (big-endian)
		return uint32(r.buffer[offset])<<24 |
			uint32(r.buffer[offset+1])<<16 |
			uint32(r.buffer[offset+2])<<8 |
			uint32(r.buffer[offset+3])
	}

	// Partial read - pad with zeros on the right
	var result uint32
	switch remaining {
	case 3:
		result = uint32(r.buffer[offset])<<24 |
			uint32(r.buffer[offset+1])<<16 |
			uint32(r.buffer[offset+2])<<8
	case 2:
		result = uint32(r.buffer[offset])<<24 |
			uint32(r.buffer[offset+1])<<16
	case 1:
		result = uint32(r.buffer[offset]) << 24
	}
	return result
}

// Error returns true if a buffer overrun occurred.
func (r *Reader) Error() bool {
	return r.err
}

// BitsLeft returns the number of unread bits in the current word.
func (r *Reader) BitsLeft() uint32 {
	return r.bitsLeft
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestNewReader`
Expected: PASS

**Step 5: Commit**

```bash
cd /home/laurent/dev/go-aac
git add internal/bits/reader.go internal/bits/reader_test.go
git commit -m "feat(bits): add Reader type and NewReader constructor

Ported from FAAD2 bits.c/bits.h. Implements two-buffer design
with big-endian loading for efficient bit reading."
```

---

## Task 2: Implement ShowBits (Peek Without Consuming)

**Files:**
- Modify: `internal/bits/reader.go`
- Modify: `internal/bits/reader_test.go`

**Step 1: Write the failing test**

```go
func TestReader_ShowBits(t *testing.T) {
	// Test pattern: 0xFF 0x0F 0xAB 0xCD 0x12 0x34 0x56 0x78
	// Binary: 11111111 00001111 10101011 11001101 00010010 00110100 01010110 01111000
	data := []byte{0xFF, 0x0F, 0xAB, 0xCD, 0x12, 0x34, 0x56, 0x78}
	r := NewReader(data)

	tests := []struct {
		name     string
		n        uint
		expected uint32
	}{
		{"peek 0 bits", 0, 0},
		{"peek 1 bit (MSB)", 1, 1},           // First bit is 1
		{"peek 4 bits", 4, 0xF},              // 1111
		{"peek 8 bits", 8, 0xFF},             // 11111111
		{"peek 12 bits", 12, 0xFF0},          // 111111110000
		{"peek 16 bits", 16, 0xFF0F},         // First 2 bytes
		{"peek 24 bits", 24, 0xFF0FAB},       // First 3 bytes
		{"peek 32 bits", 32, 0xFF0FABCD},     // First 4 bytes
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := r.ShowBits(tc.n)
			if got != tc.expected {
				t.Errorf("ShowBits(%d) = 0x%X, want 0x%X", tc.n, got, tc.expected)
			}
		})
	}

	// Verify ShowBits doesn't consume bits (multiple calls return same value)
	first := r.ShowBits(16)
	second := r.ShowBits(16)
	if first != second {
		t.Errorf("ShowBits not idempotent: first=0x%X, second=0x%X", first, second)
	}
}

func TestReader_ShowBits_CrossBuffer(t *testing.T) {
	// Need to flush some bits first, then peek across bufa/bufb boundary
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xAA, 0xBB, 0xCC, 0xDD}
	r := NewReader(data)

	// Flush 24 bits, leaving 8 bits in bufa (0xFF)
	// Then peek 16 bits should get 0xFFAA (8 from bufa + 8 from bufb)
	_ = r.GetBits(24) // consume 24 bits

	got := r.ShowBits(16)
	expected := uint32(0xFFAA)
	if got != expected {
		t.Errorf("ShowBits(16) after flush = 0x%X, want 0x%X", got, expected)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_ShowBits`
Expected: FAIL - ShowBits method not defined

**Step 3: Write minimal implementation**

Add to `internal/bits/reader.go`:

```go
// ShowBits returns the next n bits without consuming them.
// n must be 0-32.
//
// Ported from: faad_showbits() in ~/dev/faad2/libfaad/bits.h:102-113
func (r *Reader) ShowBits(n uint) uint32 {
	if n == 0 {
		return 0
	}

	if n <= uint(r.bitsLeft) {
		// All bits available in bufa
		// Shift bufa left to align MSB, then right to get n bits
		return (r.bufa << (32 - r.bitsLeft)) >> (32 - n)
	}

	// Need bits from both bufa and bufb
	bitsFromBufb := n - uint(r.bitsLeft)
	// Get remaining bits from bufa (mask and shift left)
	// Then get needed bits from bufb (shift right)
	return ((r.bufa & ((1 << r.bitsLeft) - 1)) << bitsFromBufb) |
		(r.bufb >> (32 - bitsFromBufb))
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_ShowBits`
Expected: PASS (may need GetBits stub, implement in next task)

**Step 5: Commit**

```bash
git add internal/bits/reader.go internal/bits/reader_test.go
git commit -m "feat(bits): add ShowBits method for peeking bits

Implements bit peeking without consumption, handling both
single-buffer and cross-buffer cases."
```

---

## Task 3: Implement FlushBits and GetBits

**Files:**
- Modify: `internal/bits/reader.go`
- Modify: `internal/bits/reader_test.go`

**Step 1: Write the failing tests**

```go
func TestReader_FlushBits(t *testing.T) {
	data := []byte{0xFF, 0x0F, 0xAB, 0xCD, 0x12, 0x34, 0x56, 0x78}
	r := NewReader(data)

	// Initial state: bitsLeft = 32
	if r.BitsLeft() != 32 {
		t.Fatalf("Initial BitsLeft = %d, want 32", r.BitsLeft())
	}

	// Flush 8 bits
	r.FlushBits(8)
	if r.BitsLeft() != 24 {
		t.Errorf("After flush 8: BitsLeft = %d, want 24", r.BitsLeft())
	}

	// Now ShowBits should return 0x0FABCD (next 24 bits)
	got := r.ShowBits(24)
	if got != 0x0FABCD {
		t.Errorf("After flush 8: ShowBits(24) = 0x%X, want 0x0FABCD", got)
	}
}

func TestReader_FlushBits_CrossBuffer(t *testing.T) {
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xAA, 0xBB, 0xCC, 0xDD, 0x11, 0x22, 0x33, 0x44}
	r := NewReader(data)

	// Flush 40 bits (crosses from bufa into bufb, triggers reload)
	r.FlushBits(40)

	// Now bufa should have bufb's value shifted, bufb reloaded
	// After 40 bits, we should see 0xBBCCDD11
	got := r.ShowBits(32)
	expected := uint32(0xBBCCDD11)
	if got != expected {
		t.Errorf("After flush 40: ShowBits(32) = 0x%X, want 0x%X", got, expected)
	}
}

func TestReader_GetBits(t *testing.T) {
	data := []byte{0xFF, 0x0F, 0xAB, 0xCD}
	r := NewReader(data)

	// Get 8 bits
	got := r.GetBits(8)
	if got != 0xFF {
		t.Errorf("GetBits(8) = 0x%X, want 0xFF", got)
	}

	// Get next 8 bits
	got = r.GetBits(8)
	if got != 0x0F {
		t.Errorf("GetBits(8) = 0x%X, want 0x0F", got)
	}

	// Get next 16 bits
	got = r.GetBits(16)
	if got != 0xABCD {
		t.Errorf("GetBits(16) = 0x%X, want 0xABCD", got)
	}
}

func TestReader_GetBits_Zero(t *testing.T) {
	data := []byte{0xFF, 0x0F}
	r := NewReader(data)

	got := r.GetBits(0)
	if got != 0 {
		t.Errorf("GetBits(0) = %d, want 0", got)
	}

	// Verify no bits consumed
	got = r.GetBits(8)
	if got != 0xFF {
		t.Errorf("After GetBits(0), GetBits(8) = 0x%X, want 0xFF", got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run "TestReader_FlushBits|TestReader_GetBits"`
Expected: FAIL - FlushBits, GetBits not defined

**Step 3: Write minimal implementation**

Add to `internal/bits/reader.go`:

```go
// FlushBits discards n bits from the stream.
//
// Ported from: faad_flushbits() in ~/dev/faad2/libfaad/bits.h:115-127
// and faad_flushbits_ex() in ~/dev/faad2/libfaad/bits.c:123-144
func (r *Reader) FlushBits(n uint) {
	if r.err {
		return
	}

	if n < uint(r.bitsLeft) {
		r.bitsLeft -= uint32(n)
		return
	}

	// Need to reload buffer
	r.flushBitsEx(n)
}

// flushBitsEx handles flushing when we need to reload from buffer.
//
// Ported from: faad_flushbits_ex() in ~/dev/faad2/libfaad/bits.c:123-144
func (r *Reader) flushBitsEx(n uint) {
	// Move bufb to bufa
	r.bufa = r.bufb
	// Load next word into bufb
	r.bufb = r.loadWord(r.pos)
	r.pos += 4

	// Adjust bits left: we gained 32 bits from new bufa, consumed n
	r.bitsLeft += 32 - uint32(n)
}

// GetBits reads and returns n bits from the stream.
// n must be 0-32.
//
// Ported from: faad_getbits() in ~/dev/faad2/libfaad/bits.h:130-146
func (r *Reader) GetBits(n uint) uint32 {
	if n == 0 {
		return 0
	}

	ret := r.ShowBits(n)
	r.FlushBits(n)
	return ret
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run "TestReader_FlushBits|TestReader_GetBits"`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/bits/reader.go internal/bits/reader_test.go
git commit -m "feat(bits): add FlushBits and GetBits methods

Implements bit consumption with proper buffer reload when
crossing 32-bit boundaries."
```

---

## Task 4: Implement Get1Bit (Optimized Single Bit)

**Files:**
- Modify: `internal/bits/reader.go`
- Modify: `internal/bits/reader_test.go`

**Step 1: Write the failing test**

```go
func TestReader_Get1Bit(t *testing.T) {
	// 0xA5 = 10100101 binary
	data := []byte{0xA5, 0x00, 0x00, 0x00}
	r := NewReader(data)

	expected := []uint8{1, 0, 1, 0, 0, 1, 0, 1}
	for i, want := range expected {
		got := r.Get1Bit()
		if got != want {
			t.Errorf("Get1Bit() #%d = %d, want %d", i, got, want)
		}
	}
}

func TestReader_Get1Bit_CrossBuffer(t *testing.T) {
	// Read 31 bits, then read single bits across boundary
	data := []byte{0xFF, 0xFF, 0xFF, 0xFE, 0x80, 0x00, 0x00, 0x00}
	r := NewReader(data)

	// Skip 31 bits
	_ = r.GetBits(31)

	// Next bit (bit 32) should be 0 (from 0xFE = 11111110)
	got := r.Get1Bit()
	if got != 0 {
		t.Errorf("Get1Bit() at bit 32 = %d, want 0", got)
	}

	// Next bit (bit 33) should be 1 (from 0x80 = 10000000)
	got = r.Get1Bit()
	if got != 1 {
		t.Errorf("Get1Bit() at bit 33 = %d, want 1", got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_Get1Bit`
Expected: FAIL - Get1Bit not defined

**Step 3: Write minimal implementation**

Add to `internal/bits/reader.go`:

```go
// Get1Bit reads and returns a single bit from the stream.
// Optimized path for single-bit reads.
//
// Ported from: faad_get1bit() in ~/dev/faad2/libfaad/bits.h:148-167
func (r *Reader) Get1Bit() uint8 {
	if r.bitsLeft > 0 {
		r.bitsLeft--
		return uint8((r.bufa >> r.bitsLeft) & 1)
	}

	// bitsLeft == 0, need to reload
	return uint8(r.GetBits(1))
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_Get1Bit`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/bits/reader.go internal/bits/reader_test.go
git commit -m "feat(bits): add Get1Bit method for single-bit reads

Optimized path that avoids ShowBits/FlushBits overhead for
the common case of reading individual bits."
```

---

## Task 5: Implement ByteAlign

**Files:**
- Modify: `internal/bits/reader.go`
- Modify: `internal/bits/reader_test.go`

**Step 1: Write the failing test**

```go
func TestReader_ByteAlign(t *testing.T) {
	data := []byte{0xFF, 0xAB, 0xCD, 0xEF}
	r := NewReader(data)

	// Already aligned - should skip 0 bits
	skipped := r.ByteAlign()
	if skipped != 0 {
		t.Errorf("ByteAlign() when aligned = %d, want 0", skipped)
	}

	// Read 3 bits, then align - should skip 5 bits
	_ = r.GetBits(3)
	skipped = r.ByteAlign()
	if skipped != 5 {
		t.Errorf("ByteAlign() after 3 bits = %d, want 5", skipped)
	}

	// Now we should be at byte 1 (0xAB)
	got := r.GetBits(8)
	if got != 0xAB {
		t.Errorf("After align: GetBits(8) = 0x%X, want 0xAB", got)
	}
}

func TestReader_ByteAlign_AfterFullByte(t *testing.T) {
	data := []byte{0xFF, 0xAB}
	r := NewReader(data)

	// Read 8 bits (full byte)
	_ = r.GetBits(8)

	// Already aligned
	skipped := r.ByteAlign()
	if skipped != 0 {
		t.Errorf("ByteAlign() after 8 bits = %d, want 0", skipped)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_ByteAlign`
Expected: FAIL - ByteAlign not defined

**Step 3: Write minimal implementation**

Add to `internal/bits/reader.go`:

```go
// ByteAlign aligns the bit position to the next byte boundary.
// Returns the number of bits skipped (0-7).
//
// Ported from: faad_byte_align() in ~/dev/faad2/libfaad/bits.c:111-121
func (r *Reader) ByteAlign() uint8 {
	// Calculate how many bits we've consumed in the current byte
	// bitsLeft counts down from 32, so (32 - bitsLeft) is bits consumed
	// We want remainder when divided by 8
	consumed := (32 - r.bitsLeft) & 7

	if consumed == 0 {
		return 0
	}

	skip := 8 - consumed
	r.FlushBits(uint(skip))
	return uint8(skip)
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_ByteAlign`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/bits/reader.go internal/bits/reader_test.go
git commit -m "feat(bits): add ByteAlign method for byte boundary alignment

Aligns read position to next byte boundary, returning count
of bits skipped."
```

---

## Task 6: Implement GetProcessedBits (Bit Position)

**Files:**
- Modify: `internal/bits/reader.go`
- Modify: `internal/bits/reader_test.go`

**Step 1: Write the failing test**

```go
func TestReader_GetProcessedBits(t *testing.T) {
	data := []byte{0xFF, 0x0F, 0xAB, 0xCD, 0x12, 0x34, 0x56, 0x78}
	r := NewReader(data)

	// Initially at bit 0
	if got := r.GetProcessedBits(); got != 0 {
		t.Errorf("Initial position = %d, want 0", got)
	}

	// Read 12 bits
	_ = r.GetBits(12)
	if got := r.GetProcessedBits(); got != 12 {
		t.Errorf("After 12 bits: position = %d, want 12", got)
	}

	// Read another 8 bits
	_ = r.GetBits(8)
	if got := r.GetProcessedBits(); got != 20 {
		t.Errorf("After 20 bits: position = %d, want 20", got)
	}
}

func TestReader_GetProcessedBits_CrossBuffer(t *testing.T) {
	data := []byte{
		0xFF, 0xFF, 0xFF, 0xFF, // 32 bits
		0xAA, 0xBB, 0xCC, 0xDD, // 32 bits
		0x11, 0x22, 0x33, 0x44, // 32 bits
	}
	r := NewReader(data)

	// Read 40 bits (crosses buffer boundary)
	_ = r.GetBits(32)
	_ = r.GetBits(8)

	if got := r.GetProcessedBits(); got != 40 {
		t.Errorf("After 40 bits: position = %d, want 40", got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_GetProcessedBits`
Expected: FAIL - GetProcessedBits not defined

**Step 3: Write minimal implementation**

Add to `internal/bits/reader.go`:

```go
// GetProcessedBits returns the number of bits read from the stream.
//
// Ported from: faad_get_processed_bits() in ~/dev/faad2/libfaad/bits.c:106-109
func (r *Reader) GetProcessedBits() uint32 {
	// pos is the byte offset of the next word to load
	// We've loaded (pos/4) words, but bufa and bufb are pre-loaded
	// So we've consumed (pos - 8) bytes worth = (pos - 8) * 8 bits
	// Plus (32 - bitsLeft) bits from the current bufa
	//
	// Formula: 8 * ((pos - 8) + 4) - bitsLeft = 8 * (pos - 4) - bitsLeft
	return uint32((r.pos-4)*8) - r.bitsLeft
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_GetProcessedBits`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/bits/reader.go internal/bits/reader_test.go
git commit -m "feat(bits): add GetProcessedBits for tracking read position

Returns total bits consumed from stream, used for debugging
and ensuring correct parsing."
```

---

## Task 7: Implement GetBitBuffer (Read Raw Bytes)

**Files:**
- Modify: `internal/bits/reader.go`
- Modify: `internal/bits/reader_test.go`

**Step 1: Write the failing test**

```go
func TestReader_GetBitBuffer(t *testing.T) {
	data := []byte{0xFF, 0x0F, 0xAB, 0xCD, 0x12, 0x34}
	r := NewReader(data)

	// Skip 4 bits, then read 16 bits as bytes
	_ = r.GetBits(4)
	buf := r.GetBitBuffer(16)

	// After skipping 4 bits (0xF from 0xFF), next 16 bits are:
	// 0xF (remaining) + 0x0F = 0xF0, 0xFA from 0x0FAB
	// Actually: 1111 0000 1111 1010 = 0xF0, 0xFA
	// No wait, let's trace: 0xFF0FABCD = 11111111 00001111 10101011 11001101
	// Skip 4: 1111 00001111 10101011 11001101
	// Next 16: 00001111 10101011 = 0x0F 0xAB
	expected := []byte{0xF0, 0xFA}
	if len(buf) != 2 {
		t.Fatalf("GetBitBuffer(16) len = %d, want 2", len(buf))
	}
	if buf[0] != expected[0] || buf[1] != expected[1] {
		t.Errorf("GetBitBuffer(16) = [0x%02X, 0x%02X], want [0x%02X, 0x%02X]",
			buf[0], buf[1], expected[0], expected[1])
	}
}

func TestReader_GetBitBuffer_Aligned(t *testing.T) {
	data := []byte{0xFF, 0x0F, 0xAB, 0xCD}
	r := NewReader(data)

	// Read 2 bytes (16 bits) when aligned
	buf := r.GetBitBuffer(16)
	if len(buf) != 2 {
		t.Fatalf("len = %d, want 2", len(buf))
	}
	if buf[0] != 0xFF || buf[1] != 0x0F {
		t.Errorf("got [0x%02X, 0x%02X], want [0xFF, 0x0F]", buf[0], buf[1])
	}
}

func TestReader_GetBitBuffer_WithRemainder(t *testing.T) {
	data := []byte{0xF0, 0x00, 0x00, 0x00}
	r := NewReader(data)

	// Read 4 bits - should return 1 byte with high nibble = 0xF, low nibble = 0
	buf := r.GetBitBuffer(4)
	if len(buf) != 1 {
		t.Fatalf("len = %d, want 1", len(buf))
	}
	// 4 bits 1111 shifted left by 4 = 11110000 = 0xF0
	if buf[0] != 0xF0 {
		t.Errorf("got 0x%02X, want 0xF0", buf[0])
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_GetBitBuffer`
Expected: FAIL - GetBitBuffer not defined

**Step 3: Write minimal implementation**

Add to `internal/bits/reader.go`:

```go
// GetBitBuffer reads 'bits' bits and returns them as a byte slice.
// Partial final byte is left-aligned (MSB) with zero padding.
//
// Ported from: faad_getbitbuffer() in ~/dev/faad2/libfaad/bits.c:222-245
func (r *Reader) GetBitBuffer(bits uint) []byte {
	numBytes := (bits + 7) / 8
	remainder := bits & 7

	buffer := make([]byte, numBytes)

	for i := uint(0); i < bits/8; i++ {
		buffer[i] = byte(r.GetBits(8))
	}

	if remainder > 0 {
		// Read remaining bits and left-align in the last byte
		temp := r.GetBits(remainder) << (8 - remainder)
		buffer[numBytes-1] = byte(temp)
	}

	return buffer
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_GetBitBuffer`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/bits/reader.go internal/bits/reader_test.go
git commit -m "feat(bits): add GetBitBuffer for reading raw byte chunks

Reads arbitrary bit counts as byte slices, with partial bytes
left-aligned (used for SBR data extraction)."
```

---

## Task 8: Implement ResetBits (Seek to Position)

**Files:**
- Modify: `internal/bits/reader.go`
- Modify: `internal/bits/reader_test.go`

**Step 1: Write the failing test**

```go
func TestReader_ResetBits(t *testing.T) {
	data := []byte{0xFF, 0x0F, 0xAB, 0xCD, 0x12, 0x34, 0x56, 0x78}
	r := NewReader(data)

	// Read 24 bits
	first24 := r.GetBits(24)
	if first24 != 0xFF0FAB {
		t.Fatalf("First read: got 0x%X, want 0xFF0FAB", first24)
	}

	// Reset to beginning
	r.ResetBits(0)
	if r.GetProcessedBits() != 0 {
		t.Errorf("After reset(0): position = %d, want 0", r.GetProcessedBits())
	}

	// Should read same data again
	again := r.GetBits(24)
	if again != first24 {
		t.Errorf("After reset: got 0x%X, want 0x%X", again, first24)
	}
}

func TestReader_ResetBits_ToMiddle(t *testing.T) {
	data := []byte{0xFF, 0x0F, 0xAB, 0xCD, 0x12, 0x34, 0x56, 0x78}
	r := NewReader(data)

	// Reset to bit 16
	r.ResetBits(16)

	// Should read from byte 2 (0xAB)
	got := r.GetBits(16)
	expected := uint32(0xABCD)
	if got != expected {
		t.Errorf("After reset(16): got 0x%X, want 0x%X", got, expected)
	}
}

func TestReader_ResetBits_NonByteAligned(t *testing.T) {
	data := []byte{0xFF, 0x0F, 0xAB, 0xCD, 0x12, 0x34, 0x56, 0x78}
	r := NewReader(data)

	// Reset to bit 12 (1.5 bytes)
	r.ResetBits(12)

	// First 12 bits are 0xFF0 = 111111110000, so remaining starts at 0x0FAB...
	// At bit 12, next 8 bits should be 0x0F (bits 12-19)
	// Actually: bit 12 onwards is: 0 1111 1010 1011... = 0xFAB...
	// Wait, let me trace carefully:
	// 0xFF0FABCD in binary:
	// 11111111 00001111 10101011 11001101
	// Bit 0-11:  111111110000
	// Bit 12-19: 11111010 = 0xFA
	got := r.GetBits(8)
	expected := uint32(0xFA)
	if got != expected {
		t.Errorf("After reset(12): got 0x%X, want 0x%X", got, expected)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_ResetBits`
Expected: FAIL - ResetBits not defined

**Step 3: Write minimal implementation**

Add to `internal/bits/reader.go`:

```go
// ResetBits seeks to a specific bit position in the stream.
//
// Ported from: faad_resetbits() in ~/dev/faad2/libfaad/bits.c:180-220
func (r *Reader) ResetBits(bits uint32) {
	wordIndex := int(bits / 32)
	remainder := bits & 31

	byteOffset := wordIndex * 4
	if byteOffset > r.bufferSize {
		r.err = true
		return
	}

	// Load bufa from word at wordIndex
	r.bufa = r.loadWord(byteOffset)
	// Load bufb from next word
	r.bufb = r.loadWord(byteOffset + 4)
	// Set position for next load
	r.pos = byteOffset + 8

	r.bitsLeft = 32 - remainder
	r.err = false
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_ResetBits`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/bits/reader.go internal/bits/reader_test.go
git commit -m "feat(bits): add ResetBits for seeking to bit position

Enables seeking to arbitrary bit positions, used for
error recovery and re-parsing."
```

---

## Task 9: Implement RemainingBits and Buffer Overrun Detection

**Files:**
- Modify: `internal/bits/reader.go`
- Modify: `internal/bits/reader_test.go`

**Step 1: Write the failing test**

```go
func TestReader_RemainingBits(t *testing.T) {
	data := []byte{0xFF, 0x0F, 0xAB, 0xCD}
	r := NewReader(data)

	// Initial: 32 bits available
	if got := r.RemainingBits(); got != 32 {
		t.Errorf("Initial remaining = %d, want 32", got)
	}

	// Read 12 bits
	_ = r.GetBits(12)
	if got := r.RemainingBits(); got != 20 {
		t.Errorf("After 12 bits: remaining = %d, want 20", got)
	}
}

func TestReader_BufferOverrun(t *testing.T) {
	data := []byte{0xFF, 0x0F} // Only 16 bits
	r := NewReader(data)

	// Read all 16 bits - should not error
	_ = r.GetBits(16)
	if r.Error() {
		t.Error("Should not error after reading available bits")
	}

	// Try to read more - implementation may or may not error
	// but should not panic
	_ = r.GetBits(8) // Reading past end

	// Reading well past end
	_ = r.GetBits(32)
	// Should have set error or returned zeros gracefully
}

func TestReader_BitsAvailable(t *testing.T) {
	data := []byte{0xFF, 0x0F, 0xAB, 0xCD} // 32 bits
	r := NewReader(data)

	if !r.BitsAvailable(32) {
		t.Error("Should have 32 bits available initially")
	}

	if r.BitsAvailable(33) {
		t.Error("Should not have 33 bits available")
	}

	_ = r.GetBits(16)

	if !r.BitsAvailable(16) {
		t.Error("Should have 16 bits available after reading 16")
	}

	if r.BitsAvailable(17) {
		t.Error("Should not have 17 bits available after reading 16")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run "TestReader_RemainingBits|TestReader_BufferOverrun|TestReader_BitsAvailable"`
Expected: FAIL - Methods not defined

**Step 3: Write minimal implementation**

Add to `internal/bits/reader.go`:

```go
// RemainingBits returns the number of unread bits in the buffer.
func (r *Reader) RemainingBits() uint32 {
	totalBits := uint32(r.bufferSize * 8)
	consumed := r.GetProcessedBits()
	if consumed >= totalBits {
		return 0
	}
	return totalBits - consumed
}

// BitsAvailable returns true if at least n bits remain unread.
func (r *Reader) BitsAvailable(n uint32) bool {
	return r.RemainingBits() >= n
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run "TestReader_RemainingBits|TestReader_BufferOverrun|TestReader_BitsAvailable"`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/bits/reader.go internal/bits/reader_test.go
git commit -m "feat(bits): add RemainingBits and BitsAvailable helpers

Utility methods for checking available data, useful for
error handling and end-of-stream detection."
```

---

## Task 10: ADTS Header Parsing Validation Test

**Files:**
- Create: `internal/bits/reader_adts_test.go`

This task validates the bit reader against real ADTS data using the FAAD2 debug tool.

**Step 1: Generate test data**

Run (manual preparation):
```bash
# Generate test AAC file
cd /home/laurent/dev/go-aac
ffmpeg -f lavfi -i "sine=frequency=1000:duration=0.5" -c:a aac -b:a 128k testdata/sine1k.aac 2>/dev/null

# Generate FAAD2 reference (if debug tool exists)
cd /home/laurent/dev/go-aac/scripts
./check_faad2 ../testdata/sine1k.aac 2>/dev/null || echo "Note: FAAD2 debug tool not built yet"
```

**Step 2: Write the validation test**

```go
// internal/bits/reader_adts_test.go
package bits

import (
	"os"
	"testing"
)

// TestReader_ADTSHeaderParsing validates bit reading against known ADTS header structure.
// ADTS header is 56 bits (7 bytes) without CRC, 72 bits (9 bytes) with CRC.
//
// Reference: ISO/IEC 13818-7 section 6.2.1
func TestReader_ADTSHeaderParsing(t *testing.T) {
	// Read test file if available
	data, err := os.ReadFile("../../testdata/sine1k.aac")
	if err != nil {
		t.Skipf("Test file not available: %v", err)
	}

	r := NewReader(data)

	// Parse ADTS header fields per spec
	// https://wiki.multimedia.cx/index.php/ADTS
	syncword := r.GetBits(12)
	if syncword != 0xFFF {
		t.Fatalf("Invalid syncword: 0x%X, expected 0xFFF", syncword)
	}

	id := r.Get1Bit()                  // 0 = MPEG-4, 1 = MPEG-2
	layer := r.GetBits(2)              // Always 0
	protectionAbsent := r.Get1Bit()    // 1 = no CRC
	profile := r.GetBits(2)            // 0=Main, 1=LC, 2=SSR, 3=reserved
	sfIndex := r.GetBits(4)            // Sample rate index
	privateBit := r.Get1Bit()          // Ignored
	channelConfig := r.GetBits(3)      // Channel configuration
	original := r.Get1Bit()            // Ignored
	home := r.Get1Bit()                // Ignored
	copyrightIDBit := r.Get1Bit()      // Ignored
	copyrightIDStart := r.Get1Bit()    // Ignored
	frameLength := r.GetBits(13)       // Frame length including header
	bufferFullness := r.GetBits(11)    // Buffer fullness
	numRawBlocks := r.GetBits(2)       // Number of raw data blocks - 1

	// Validate parsed values
	t.Logf("ADTS Header parsed:")
	t.Logf("  ID (MPEG version): %d", id)
	t.Logf("  Layer: %d (should be 0)", layer)
	t.Logf("  Protection absent: %d", protectionAbsent)
	t.Logf("  Profile: %d (1=LC)", profile)
	t.Logf("  Sample rate index: %d", sfIndex)
	t.Logf("  Private bit: %d", privateBit)
	t.Logf("  Channel config: %d", channelConfig)
	t.Logf("  Original: %d", original)
	t.Logf("  Home: %d", home)
	t.Logf("  Copyright ID bit: %d", copyrightIDBit)
	t.Logf("  Copyright ID start: %d", copyrightIDStart)
	t.Logf("  Frame length: %d bytes", frameLength)
	t.Logf("  Buffer fullness: %d", bufferFullness)
	t.Logf("  Num raw blocks: %d", numRawBlocks)

	// Basic validity checks
	if layer != 0 {
		t.Errorf("Layer should be 0, got %d", layer)
	}

	if profile > 3 {
		t.Errorf("Profile out of range: %d", profile)
	}

	if sfIndex > 12 {
		t.Errorf("Sample rate index out of range: %d", sfIndex)
	}

	if channelConfig > 7 {
		t.Errorf("Channel config out of range: %d", channelConfig)
	}

	// Frame length should be reasonable (header + some data)
	if frameLength < 7 || frameLength > 8192 {
		t.Errorf("Frame length suspicious: %d", frameLength)
	}

	// Check we consumed exactly 56 bits
	consumed := r.GetProcessedBits()
	if consumed != 56 {
		t.Errorf("Consumed %d bits, expected 56 for ADTS header", consumed)
	}
}

// TestReader_ADTSMultipleFrames parses multiple ADTS frames to validate
// buffer handling across frame boundaries.
func TestReader_ADTSMultipleFrames(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sine1k.aac")
	if err != nil {
		t.Skipf("Test file not available: %v", err)
	}

	offset := 0
	frameCount := 0
	maxFrames := 5

	for offset < len(data) && frameCount < maxFrames {
		if len(data)-offset < 7 {
			break // Not enough data for header
		}

		r := NewReader(data[offset:])

		// Check syncword
		syncword := r.GetBits(12)
		if syncword != 0xFFF {
			t.Fatalf("Frame %d: Invalid syncword at offset %d: 0x%X",
				frameCount, offset, syncword)
		}

		// Skip to frame length field
		_ = r.GetBits(3)  // id, layer, protection_absent
		_ = r.GetBits(12) // profile, sf_index, private, channel_config
		_ = r.GetBits(4)  // original, home, copyright bits
		frameLength := r.GetBits(13)

		t.Logf("Frame %d at offset %d: length=%d bytes", frameCount, offset, frameLength)

		if frameLength < 7 {
			t.Fatalf("Frame %d: Invalid frame length %d", frameCount, frameLength)
		}

		offset += int(frameLength)
		frameCount++
	}

	t.Logf("Successfully parsed %d ADTS frames", frameCount)
}
```

**Step 3: Run the test**

Run: `cd /home/laurent/dev/go-aac && go test ./internal/bits/... -v -run TestReader_ADTS`
Expected: PASS (or SKIP if test file not available)

**Step 4: Commit**

```bash
git add internal/bits/reader_adts_test.go
git commit -m "test(bits): add ADTS header parsing validation

Validates bit reader against real ADTS frames, ensuring
correct parsing of AAC transport stream headers."
```

---

## Task 11: Run Full Test Suite and Final Cleanup

**Files:**
- Verify all tests pass

**Step 1: Run all tests**

Run: `cd /home/laurent/dev/go-aac && make check`
Expected: All checks pass (fmt, lint, test)

**Step 2: Generate test file if needed**

```bash
cd /home/laurent/dev/go-aac
mkdir -p testdata
ffmpeg -f lavfi -i "sine=frequency=1000:duration=0.5" -c:a aac -b:a 128k testdata/sine1k.aac 2>/dev/null || echo "FFmpeg not available"
```

**Step 3: Final commit with summary**

```bash
git add -A
git status
# If any uncommitted changes:
git commit -m "feat(bits): complete bit reader implementation

Phase 1 Step 1.4 complete. Implements:
- Reader type with two-buffer design (bufa/bufb)
- NewReader constructor with big-endian loading
- ShowBits (peek without consume)
- FlushBits/GetBits (consume bits)
- Get1Bit (optimized single-bit read)
- ByteAlign (align to byte boundary)
- GetProcessedBits (track position)
- GetBitBuffer (extract raw bytes)
- ResetBits (seek to position)
- RemainingBits/BitsAvailable (bounds checking)

All ported from FAAD2 bits.c/bits.h with source references."
```

---

## Summary

| Task | Description | Est. Time |
|------|-------------|-----------|
| 1 | Reader type + NewReader constructor | 5 min |
| 2 | ShowBits (peek) | 5 min |
| 3 | FlushBits + GetBits | 5 min |
| 4 | Get1Bit (optimized) | 3 min |
| 5 | ByteAlign | 3 min |
| 6 | GetProcessedBits | 3 min |
| 7 | GetBitBuffer | 5 min |
| 8 | ResetBits (seek) | 5 min |
| 9 | RemainingBits + BitsAvailable | 3 min |
| 10 | ADTS validation test | 5 min |
| 11 | Final test suite + cleanup | 3 min |

**Total: ~45 minutes**

---

## Verification Against FAAD2

After implementation, compare behavior against FAAD2 debug output:

```bash
# Build FAAD2 debug tool
cd ~/dev/go-aac/scripts && make

# Generate reference data
./check_faad2 ../testdata/sine1k.aac

# Reference ADTS header will be in /tmp/faad2_ref_sine1k/frame_0001_adts.bin
# Compare parsed values match
```
