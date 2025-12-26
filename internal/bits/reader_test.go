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
