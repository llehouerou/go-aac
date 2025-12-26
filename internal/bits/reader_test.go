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

func TestNewReader_LoadsBuffersCorrectly(t *testing.T) {
	// 8 bytes = exactly fills bufa and bufb
	data := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
	r := NewReader(data)

	// bufa should be first 4 bytes in big-endian: 0x12345678
	expectedBufA := uint32(0x12345678)
	if r.bufa != expectedBufA {
		t.Errorf("bufa = 0x%08X, want 0x%08X", r.bufa, expectedBufA)
	}

	// bufb should be next 4 bytes in big-endian: 0x9ABCDEF0
	expectedBufB := uint32(0x9ABCDEF0)
	if r.bufb != expectedBufB {
		t.Errorf("bufb = 0x%08X, want 0x%08X", r.bufb, expectedBufB)
	}
}

func TestNewReader_PartialBufferLoading(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		expectedBufA uint32
		expectedBufB uint32
	}{
		{
			name:         "1 byte",
			data:         []byte{0xAB},
			expectedBufA: 0xAB000000,
			expectedBufB: 0x00000000,
		},
		{
			name:         "2 bytes",
			data:         []byte{0xAB, 0xCD},
			expectedBufA: 0xABCD0000,
			expectedBufB: 0x00000000,
		},
		{
			name:         "3 bytes",
			data:         []byte{0xAB, 0xCD, 0xEF},
			expectedBufA: 0xABCDEF00,
			expectedBufB: 0x00000000,
		},
		{
			name:         "4 bytes",
			data:         []byte{0x12, 0x34, 0x56, 0x78},
			expectedBufA: 0x12345678,
			expectedBufB: 0x00000000,
		},
		{
			name:         "5 bytes",
			data:         []byte{0x12, 0x34, 0x56, 0x78, 0xAB},
			expectedBufA: 0x12345678,
			expectedBufB: 0xAB000000,
		},
		{
			name:         "6 bytes",
			data:         []byte{0x12, 0x34, 0x56, 0x78, 0xAB, 0xCD},
			expectedBufA: 0x12345678,
			expectedBufB: 0xABCD0000,
		},
		{
			name:         "7 bytes",
			data:         []byte{0x12, 0x34, 0x56, 0x78, 0xAB, 0xCD, 0xEF},
			expectedBufA: 0x12345678,
			expectedBufB: 0xABCDEF00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReader(tt.data)
			if r.Error() {
				t.Errorf("NewReader should not error for %d bytes", len(tt.data))
			}
			if r.bufa != tt.expectedBufA {
				t.Errorf("bufa = 0x%08X, want 0x%08X", r.bufa, tt.expectedBufA)
			}
			if r.bufb != tt.expectedBufB {
				t.Errorf("bufb = 0x%08X, want 0x%08X", r.bufb, tt.expectedBufB)
			}
		})
	}
}

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
		{"peek 1 bit (MSB)", 1, 1},       // First bit is 1
		{"peek 4 bits", 4, 0xF},          // 1111
		{"peek 8 bits", 8, 0xFF},         // 11111111
		{"peek 12 bits", 12, 0xFF0},      // 111111110000
		{"peek 16 bits", 16, 0xFF0F},     // First 2 bytes
		{"peek 24 bits", 24, 0xFF0FAB},   // First 3 bytes
		{"peek 32 bits", 32, 0xFF0FABCD}, // First 4 bytes
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

func TestReader_ShowBits_AfterPartialRead(t *testing.T) {
	// Test ShowBits when bitsLeft < 32 (simulating partial consumption)
	// This tests the cross-buffer case without needing GetBits
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xAA, 0xBB, 0xCC, 0xDD}
	r := NewReader(data)

	// Manually simulate having consumed 24 bits (for testing purposes)
	// After consuming 24 bits, bitsLeft would be 8, and bufa would still have
	// the last 8 bits (0xFF) in the low position
	// Simulate: we have 8 bits left in bufa (value 0xFF in low 8 bits)
	r.bitsLeft = 8
	// bufa still contains the original 0xFFFFFFFF, but only bottom 8 bits matter
	// The 8 bits left are the last 8 bits of original bufa = 0xFF

	// Now peek 16 bits should get 8 from bufa + 8 from bufb
	// bufa low 8 bits = 0xFF, bufb high 8 bits = 0xAA
	// Result should be 0xFFAA
	got := r.ShowBits(16)
	expected := uint32(0xFFAA)
	if got != expected {
		t.Errorf("ShowBits(16) after partial = 0x%X, want 0x%X", got, expected)
	}
}

func TestReader_ShowBits_EdgeCases(t *testing.T) {
	// Test exactly bitsLeft boundary
	data := []byte{0x12, 0x34, 0x56, 0x78, 0xAB, 0xCD, 0xEF, 0x00}
	r := NewReader(data)

	// Simulate 16 bits left
	r.bitsLeft = 16

	// Peek exactly 16 bits (boundary case: n == bitsLeft)
	got := r.ShowBits(16)
	// Low 16 bits of bufa (0x12345678) = 0x5678
	expected := uint32(0x5678)
	if got != expected {
		t.Errorf("ShowBits(16) at boundary = 0x%X, want 0x%X", got, expected)
	}

	// Peek 17 bits (just over boundary: n > bitsLeft)
	// Should get 16 bits from bufa (0x5678) + 1 bit from bufb
	// bufb = 0xABCDEF00, top bit = 1
	// Result: 0x5678 << 1 | 1 = 0xACF1
	got = r.ShowBits(17)
	expected = uint32(0xACF1)
	if got != expected {
		t.Errorf("ShowBits(17) crossing boundary = 0x%X, want 0x%X", got, expected)
	}
}

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
