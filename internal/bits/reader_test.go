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
