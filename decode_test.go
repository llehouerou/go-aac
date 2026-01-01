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
