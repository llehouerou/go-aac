// Package huffman implements AAC Huffman decoding.
package huffman

import (
	"fmt"
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

func TestDecode2StepQuad(t *testing.T) {
	// Test with a known codeword from codebook 1
	// The first entry in hcb1_2 (index 0) has bits=1, x=0, y=0, v=0, w=0
	// which corresponds to codeword "0" (1 bit)
	// From hcb1_1[0]: offset=0, extra_bits=0

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

func TestDecode2StepQuad_LongerCodeword(t *testing.T) {
	// Test a codeword that requires extra bits (7-bit codeword)
	// For codebook 1:
	// First 5 bits = 11000 (binary = 24) -> hcb1_1[24] = {offset: 9, extra_bits: 2}
	// Next 2 bits = 00 -> final offset = 9 + 0 = 9
	// hcb1_2[9] = {7, 1, -1, 0, 0} -> 7 bit codeword, values (1, -1, 0, 0)

	// Bits: 1100000x xxxxxxxx
	// 11000 = first 5 bits, 00 = extra bits for offset
	data := []byte{0xC0, 0x00} // 0b11000000 0b00000000
	r := bits.NewReader(data)

	var sp [4]int16
	err := decode2StepQuad(1, r, sp[:])

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := [4]int16{1, -1, 0, 0}
	if sp != expected {
		t.Errorf("got %v, want %v", sp, expected)
	}
}

func TestDecode2StepPair(t *testing.T) {
	// Test with a known codeword from codebook 6
	// The first entry in hcb6_2 (index 0) has bits=4, x=0, y=0
	// which corresponds to codeword "0000" (4 bits)
	// From hcb6_1[0]: offset=0, extra_bits=0

	// So bit pattern 00000xxx (first 5 bits = 0) should give (0,0)
	data := []byte{0x00, 0x00}
	r := bits.NewReader(data)

	var sp [2]int16
	err := decode2StepPair(6, r, sp[:])

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := [2]int16{0, 0}
	if sp != expected {
		t.Errorf("got %v, want %v", sp, expected)
	}
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

func TestDecode2StepPair_LongerCodeword(t *testing.T) {
	// Test a codeword that requires extra bits (6-bit codeword)
	// For codebook 6:
	// First 5 bits = 10010 (binary = 18) -> hcb6_1[18] = {9, 1} (offset: 9, extra_bits: 1)
	// Next 1 bit = 0 -> final offset = 9 + 0 = 9
	// hcb6_2[9] = {6, 2, -1} -> 6-bit codeword, values (2, -1)

	// Bits: 10010 0xx xxxxx
	data := []byte{0x90, 0x00} // 0b10010000 0b00000000
	r := bits.NewReader(data)

	var sp [2]int16
	err := decode2StepPair(6, r, sp[:])

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := [2]int16{2, -1}
	if sp != expected {
		t.Errorf("got %v, want %v", sp, expected)
	}
}

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
			expected: 17,                     // (1 << 4) | 1 = 17
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
			expected: 33,                           // (1 << 5) | 1 = 33
		},
		{
			name:  "malformed escape: too many leading ones",
			input: 16,
			// 12 ones would make i=16 (starting at 4), which is an error
			bits:     []uint8{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			expected: 16, // unchanged on error
			err:      true,
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
