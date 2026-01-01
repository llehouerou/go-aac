// decode.go
package aac

// Decode decodes one AAC frame and returns PCM samples.
//
// Parameters:
//   - buffer: Input AAC frame data
//
// Returns:
//   - samples: Interleaved PCM samples (int16 for 16-bit format)
//   - info: Frame information (channels, sample rate, bytes consumed, etc.)
//   - err: Error if decoding fails
//
// The decoder must be initialized with Init() or Init2() before calling Decode().
// Each call to Decode() processes exactly one frame. For ADTS streams, the ADTS
// header is parsed automatically. For raw AAC, the caller must provide frame
// boundaries.
//
// Note: The first frame returns zero samples due to the overlap-add delay.
// This matches FAAD2 behavior (decoder.c:1204-1206).
//
// Ported from: aac_frame_decode() in ~/dev/faad2/libfaad/decoder.c:848-1255
func (d *Decoder) Decode(buffer []byte) (interface{}, *FrameInfo, error) {
	// Safety checks
	// Ported from: decoder.c:872-876
	if d == nil {
		return nil, nil, ErrNilDecoder
	}
	if buffer == nil {
		return nil, nil, ErrNilBuffer
	}
	if len(buffer) == 0 {
		return nil, nil, ErrBufferTooSmall
	}

	// Initialize FrameInfo
	info := &FrameInfo{}

	// Check for ID3v1 tag (128 bytes starting with "TAG")
	// Ported from: decoder.c:901-910
	if len(buffer) >= 128 && buffer[0] == 'T' && buffer[1] == 'A' && buffer[2] == 'G' {
		info.BytesConsumed = 128
		// No error, but no output either
		return nil, info, nil
	}

	// TODO: Continue with bitstream parsing
	return nil, info, nil
}
