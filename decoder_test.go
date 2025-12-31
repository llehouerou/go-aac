// decoder_test.go
package aac

import "testing"

func TestDecoder_New(t *testing.T) {
	dec := NewDecoder()
	if dec == nil {
		t.Fatal("NewDecoder returned nil")
	}

	// Verify default configuration
	if dec.Config().OutputFormat != OutputFormat16Bit {
		t.Errorf("default output format: got %d, want %d", dec.Config().OutputFormat, OutputFormat16Bit)
	}

	// Verify frame length is 1024 (standard AAC)
	if dec.frameLength != 1024 {
		t.Errorf("default frame length: got %d, want 1024", dec.frameLength)
	}
}
