// internal/syntax/dse.go
//
// # Data Stream Element Parsing
//
// This file implements:
// - ParseDataStreamElement: Parses ID_DSE elements
//
// Data Stream Elements carry auxiliary data that is not part of the
// audio bitstream. This data is typically discarded during decoding.
//
// Ported from: ~/dev/faad2/libfaad/syntax.c:1080-1107
package syntax

import "github.com/llehouerou/go-aac/internal/bits"

// ParseDataStreamElement parses a Data Stream Element (DSE).
// DSE carries auxiliary data that is not part of the audio bitstream.
// The data is simply skipped after reading.
//
// Returns the number of data bytes in the element.
//
// Ported from: data_stream_element() in ~/dev/faad2/libfaad/syntax.c:1080-1107
func ParseDataStreamElement(r *bits.Reader) uint16 {
	// element_instance_tag (4 bits) - discarded
	_ = r.GetBits(LenTag)

	// data_byte_align_flag (1 bit)
	byteAligned := r.Get1Bit() == 1

	// count (8 bits)
	count := uint16(r.GetBits(8))

	// If count == 255, read extended count
	if count == 255 {
		count += uint16(r.GetBits(8))
	}

	// Byte align if requested
	if byteAligned {
		r.ByteAlign()
	}

	// Skip data_stream_bytes
	for i := uint16(0); i < count; i++ {
		r.GetBits(LenByte)
	}

	return count
}
