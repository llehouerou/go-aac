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
