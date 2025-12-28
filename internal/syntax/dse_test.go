// Package syntax implements AAC bitstream syntax parsing.
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
	//
	// Bit layout (MSB first, bits numbered from 0):
	// bits 0-3: tag = 5 = 0101
	// bit 4: align = 0
	// bits 5-12: count = 3 = 00000011
	// bits 13-20: data[0] = 0xAA = 10101010
	// bits 21-28: data[1] = 0xBB = 10111011
	// bits 29-36: data[2] = 0xCC = 11001100
	//
	// Byte layout:
	// Byte 0 (bits 0-7):  0101_0_000 = 0x28 (tag=5, align=0, count_high_3=000)
	// Byte 1 (bits 8-15): 00011_101 = 0x1D (count_low_5=00011, data0_high_3=101)
	// Byte 2 (bits 16-23): 01010_101 = 0x55 (data0_low_5=01010, data1_high_3=101)
	// Byte 3 (bits 24-31): 11011_110 = 0xDE (data1_low_5=11011, data2_high_3=110)
	// Byte 4 (bits 32-39): 01100_xxx = 0x60 (data2_low_5=01100, padding)

	data := []byte{0x28, 0x1D, 0x55, 0xDE, 0x60}
	r := bits.NewReader(data)

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
	// Then 265 bytes of data (just zeros for simplicity)
	//
	// Bit layout:
	// bits 0-3: tag = 0 = 0000
	// bit 4: align = 0
	// bits 5-12: count = 255 = 11111111
	// bits 13-20: extra = 10 = 00001010
	// bits 21+: 265 bytes of data
	//
	// Byte layout:
	// Byte 0 (bits 0-7):  0000_0_111 = 0x07 (tag=0, align=0, count_high_3=111)
	// Byte 1 (bits 8-15): 11111_000 = 0xF8 (count_low_5=11111, extra_high_3=000)
	// Byte 2 (bits 16-23): 01010_000 = 0x50 (extra_low_5=01010, data_starts)

	data := make([]byte, 300)
	data[0] = 0x07 // tag=0, align=0, count[0:2]=111
	data[1] = 0xF8 // count[3:7]=11111, extra[0:2]=000
	data[2] = 0x50 // extra[3:7]=01010, data[0][0:2]=000
	// Remaining bytes can be zero (data content doesn't matter, just skipped)

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
	// [byte_align to next boundary - skip 3 bits to reach bit 16]
	// data_stream_byte[0-1]: 2 bytes = 0xAA, 0xBB
	//
	// Bit layout:
	// bits 0-3: tag = 3 = 0011
	// bit 4: align = 1
	// bits 5-12: count = 2 = 00000010
	// bits 13-15: padding (3 bits to reach byte boundary at bit 16)
	// bits 16-23: data[0] = 0xAA
	// bits 24-31: data[1] = 0xBB
	//
	// Byte layout:
	// Byte 0 (bits 0-7):  0011_1_000 = 0x38 (tag=3, align=1, count_high_3=000)
	// Byte 1 (bits 8-15): 00010_xxx = 0x10 (count_low_5=00010, padding=000)
	// Byte 2 (bits 16-23): 10101010 = 0xAA (data[0])
	// Byte 3 (bits 24-31): 10111011 = 0xBB (data[1])

	data := []byte{0x38, 0x10, 0xAA, 0xBB}
	r := bits.NewReader(data)

	bytesRead := ParseDataStreamElement(r)

	if bytesRead != 2 {
		t.Errorf("bytesRead = %d, want 2", bytesRead)
	}
}

func TestParseDataStreamElement_ZeroBytes(t *testing.T) {
	// Test with zero data bytes
	// element_instance_tag: 4 bits = 0
	// data_byte_align_flag: 1 bit = 0
	// count: 8 bits = 0
	//
	// Bit layout:
	// bits 0-3: tag = 0 = 0000
	// bit 4: align = 0
	// bits 5-12: count = 0 = 00000000
	//
	// Byte layout:
	// Byte 0 (bits 0-7):  0000_0_000 = 0x00
	// Byte 1 (bits 8-12): 00000_xxx = 0x00

	data := []byte{0x00, 0x00}
	r := bits.NewReader(data)

	bytesRead := ParseDataStreamElement(r)

	if bytesRead != 0 {
		t.Errorf("bytesRead = %d, want 0", bytesRead)
	}
}
