package aac

import "testing"

func TestDecoder_Decode_NilDecoder(t *testing.T) {
	var d *Decoder
	_, _, err := d.Decode([]byte{0xFF, 0xF1, 0x50, 0x80})
	if err != ErrNilDecoder {
		t.Errorf("expected ErrNilDecoder, got %v", err)
	}
}

func TestDecoder_Decode_NilBuffer(t *testing.T) {
	d := NewDecoder()
	_, _, err := d.Decode(nil)
	if err != ErrNilBuffer {
		t.Errorf("expected ErrNilBuffer, got %v", err)
	}
}

func TestDecoder_Decode_EmptyBuffer(t *testing.T) {
	d := NewDecoder()
	_, _, err := d.Decode([]byte{})
	if err != ErrBufferTooSmall {
		t.Errorf("expected ErrBufferTooSmall, got %v", err)
	}
}

func TestDecoder_Decode_ID3Tag(t *testing.T) {
	d := NewDecoder()
	// Initialize with valid ADTS header first
	adtsHeader := []byte{0xFF, 0xF1, 0x50, 0x80, 0x00, 0x1F, 0xFC}
	_, err := d.Init(adtsHeader)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create ID3v1 tag (128 bytes starting with "TAG")
	id3Tag := make([]byte, 128)
	copy(id3Tag, []byte("TAG"))

	// Decode should return nil samples and consume 128 bytes
	samples, info, err := d.Decode(id3Tag)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if samples != nil {
		t.Error("expected nil samples for ID3 tag")
	}
	if info == nil {
		t.Fatal("expected non-nil FrameInfo")
	}
	if info.BytesConsumed != 128 {
		t.Errorf("BytesConsumed: got %d, want 128", info.BytesConsumed)
	}
}

func TestDecoder_Decode_ADTSHeaderParsed(t *testing.T) {
	d := NewDecoder()
	// Initialize with ADTS stream
	adtsHeader := []byte{0xFF, 0xF1, 0x50, 0x80, 0x00, 0x1F, 0xFC}
	_, err := d.Init(adtsHeader)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create minimal ADTS frame with header.
	// Frame length = 7 bytes (header only, no payload).
	// ADTS fixed header (28 bits after syncword):
	//   0xFF, 0xF1 = syncword (12b) + id=0 (1b) + layer=0 (2b) + protection_absent=1 (1b)
	//   0x50 = profile=1(LC) (2b) + sf_index=4(44100) (4b) + private=0 (1b) + chan_config high bit=0 (1b)
	//   0x80 = chan_config low bits=10 (2b) + original=0 + home=0 + copyright_id_bit=0 + copyright_id_start=0
	// ADTS variable header (28 bits):
	//   frame_length (13 bits) = 7 (0x007)
	//   adts_buffer_fullness (11 bits) = 0x7FF (VBR)
	//   number_of_raw_data_blocks (2 bits) = 0
	//
	// Encoding frame_length=7:
	//   Bits 54-42 = frame_length = 0x007 = 0000000000111
	// Let's calculate the bytes:
	//   Byte 4 (bits 32-39): last 2 bits of chan_config (10), orig(0), home(0), copy_id(0), copy_start(0), frame_len[12:11]=00
	//   Byte 5 (bits 40-47): frame_len[10:3] = 00000001
	//   Byte 6 (bits 48-55): frame_len[2:0]=110, buffer_fullness[10:6]=11111
	//   Byte 7 (bits 56-63): buffer_fullness[5:0]=111111, num_blocks=00
	frame := []byte{
		0xFF, 0xF1, // syncword + id=0, layer=0, protection_absent=1
		0x50, // profile=1(LC), sf_index=4(44100), private=0, chan_config[2]=0
		0x80, // chan_config[1:0]=10, original=0, home=0, copyright_id_bit=0, copyright_id_start=0, frame_length[12:11]=00
		0x01, // frame_length[10:3] = 00000001 (for frame_length=7)
		0xDF, // frame_length[2:0]=110, buffer_fullness[10:6]=11111
		0xFC, // buffer_fullness[5:0]=111111, num_blocks=00
	}

	// This should parse ADTS header without error
	// (decoding will fail due to no payload, but header parsing should work)
	_, info, _ := d.Decode(frame)
	if info == nil {
		t.Fatal("expected non-nil FrameInfo")
	}
	if info.HeaderType != HeaderTypeADTS {
		t.Errorf("HeaderType: got %d, want %d (ADTS)", info.HeaderType, HeaderTypeADTS)
	}
}

func TestDecoder_Decode_ADIFHeaderType(t *testing.T) {
	d := NewDecoder()
	// Manually set ADIF mode (since ADIF init is not fully implemented)
	d.adifHeaderPresent = true

	// Create minimal buffer (not a real ADIF stream, just to test header type)
	buffer := []byte{0x00, 0x00, 0x00, 0x00}

	_, info, _ := d.Decode(buffer)
	if info == nil {
		t.Fatal("expected non-nil FrameInfo")
	}
	if info.HeaderType != HeaderTypeADIF {
		t.Errorf("HeaderType: got %d, want %d (ADIF)", info.HeaderType, HeaderTypeADIF)
	}
}

func TestDecoder_Decode_RawHeaderType(t *testing.T) {
	d := NewDecoder()
	// No header present = raw AAC

	// Create minimal buffer
	buffer := []byte{0x00, 0x00, 0x00, 0x00}

	_, info, _ := d.Decode(buffer)
	if info == nil {
		t.Fatal("expected non-nil FrameInfo")
	}
	if info.HeaderType != HeaderTypeRAW {
		t.Errorf("HeaderType: got %d, want %d (RAW)", info.HeaderType, HeaderTypeRAW)
	}
}
