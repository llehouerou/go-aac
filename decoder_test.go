// decoder_test.go
package aac

import "testing"

func TestDecoder_New(t *testing.T) {
	dec := NewDecoder()
	if dec == nil {
		t.Fatal("NewDecoder returned nil")
	}

	// Verify default configuration
	cfg := dec.Config()
	if cfg.OutputFormat != OutputFormat16Bit {
		t.Errorf("default output format: got %d, want %d", cfg.OutputFormat, OutputFormat16Bit)
	}
	if cfg.DefObjectType != ObjectTypeMain {
		t.Errorf("default object type: got %d, want %d (MAIN)", cfg.DefObjectType, ObjectTypeMain)
	}
	if cfg.DefSampleRate != 44100 {
		t.Errorf("default sample rate: got %d, want 44100", cfg.DefSampleRate)
	}

	// Verify frame length is 1024 (standard AAC)
	if dec.frameLength != 1024 {
		t.Errorf("default frame length: got %d, want 1024", dec.frameLength)
	}

	// Verify RNG seeds match FAAD2 (decoder.c:151-153)
	if dec.rngState1 != 0x2bb431ea {
		t.Errorf("rngState1: got 0x%x, want 0x2bb431ea", dec.rngState1)
	}
	if dec.rngState2 != 0x206155b7 {
		t.Errorf("rngState2: got 0x%x, want 0x206155b7", dec.rngState2)
	}
}

func TestDecoder_Constants_MatchFAAD2(t *testing.T) {
	// Verify constants match FAAD2's structs.h:43-44 definitions
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"maxChannels", maxChannels, 64},
		{"maxSyntaxElements", maxSyntaxElements, 48},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s: got %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
}
