package bits

// Reader reads bits from a byte buffer.
//
// It uses a two-buffer approach for efficient bit reading:
// - bufa holds the current 32 bits being read from
// - bufb pre-loads the next 32 bits for look-ahead
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
// The reader pre-loads the first 64 bits (or as many as available) into
// two 32-bit buffers for efficient reading. Empty or nil buffers set
// the error flag.
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
// Handles partial reads at end of buffer by padding with zeros on the right.
//
// Ported from: getdword() in bits.h:96-100 and getdword_n() in bits.c:38-52
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
