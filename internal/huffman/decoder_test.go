// Package huffman implements AAC Huffman decoding.
package huffman

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestScaleFactor_ZeroValue(t *testing.T) {
	// The scale factor codebook returns delta values centered at 60.
	// Index 60 = delta 0, index 59 = delta -1, index 61 = delta +1, etc.
	// For TDD, test that the function exists and returns something in valid range.

	data := []byte{0xFF, 0xFF} // All 1s - will traverse deep
	r := bits.NewReader(data)

	sf := ScaleFactor(r)

	// sf should be in range [-60, 60] for valid scale factors
	if sf < -60 || sf > 60 {
		t.Errorf("ScaleFactor out of range: got %d", sf)
	}
}

func TestScaleFactor_KnownCodewords(t *testing.T) {
	// Test specific known codewords against expected values.
	// These are traced through the hcb_sf table manually.
	//
	// Reference: ~/dev/faad2/libfaad/codebook/hcb_sf.h

	tests := []struct {
		name     string
		data     []byte
		expected int8
	}{
		{
			// Codeword: "0" (1 bit)
			// Path: 0 -> +1 -> 1 (leaf, value=60)
			// Delta: 60 - 60 = 0
			name:     "delta_0_shortest_codeword",
			data:     []byte{0x00}, // 0b00000000
			expected: 0,
		},
		{
			// Codeword: "100" (3 bits)
			// Path: 0 ->+2-> 2 ->+1-> 3 ->+2-> 5 (leaf, value=59)
			// Delta: 59 - 60 = -1
			name:     "delta_minus1",
			data:     []byte{0x80}, // 0b10000000
			expected: -1,
		},
		{
			// Codeword: "1010" (4 bits)
			// Path: 0 ->+2-> 2 ->+1-> 3 ->+3-> 6 ->+3-> 9 (leaf, value=61)
			// Delta: 61 - 60 = +1
			name:     "delta_plus1",
			data:     []byte{0xA0}, // 0b10100000
			expected: 1,
		},
		{
			// Codeword: "1011" (4 bits)
			// Path: 0 ->+2-> 2 ->+1-> 3 ->+3-> 6 ->+4-> 10 (leaf, value=58)
			// Delta: 58 - 60 = -2
			name:     "delta_minus2",
			data:     []byte{0xB0}, // 0b10110000
			expected: -2,
		},
		{
			// Codeword: "1100" (4 bits)
			// Path: 0 ->+2-> 2 ->+2-> 4 ->+3-> 7 ->+4-> 11 (leaf, value=62)
			// Delta: 62 - 60 = +2
			name:     "delta_plus2",
			data:     []byte{0xC0}, // 0b11000000
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bits.NewReader(tt.data)
			got := ScaleFactor(r)
			if got != tt.expected {
				t.Errorf("ScaleFactor() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestScaleFactor_ConsecutiveDecoding(t *testing.T) {
	// Test decoding multiple scale factors from the same bitstream.
	// This verifies the reader advances correctly.

	// Encode: delta 0 (bit "0"), delta -1 (bits "100"), delta 1 (bits "1010")
	// Total: 0|100|1010 = 0b0100_1010 = 0x4A
	data := []byte{0x4A, 0x00}
	r := bits.NewReader(data)

	// First: delta 0 (codeword "0")
	sf1 := ScaleFactor(r)
	if sf1 != 0 {
		t.Errorf("First ScaleFactor() = %d, want 0", sf1)
	}

	// Second: delta -1 (codeword "100")
	sf2 := ScaleFactor(r)
	if sf2 != -1 {
		t.Errorf("Second ScaleFactor() = %d, want -1", sf2)
	}

	// Third: delta +1 (codeword "1010")
	sf3 := ScaleFactor(r)
	if sf3 != 1 {
		t.Errorf("Third ScaleFactor() = %d, want 1", sf3)
	}
}
