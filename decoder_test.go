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

func TestDecoder_SetConfiguration(t *testing.T) {
	dec := NewDecoder()

	cfg := Config{
		DefObjectType: ObjectTypeHEAAC,
		DefSampleRate: 48000,
		OutputFormat:  OutputFormatFloat,
		DownMatrix:    true,
	}

	dec.SetConfiguration(cfg)

	got := dec.Config()
	if got.DefObjectType != cfg.DefObjectType {
		t.Errorf("DefObjectType: got %d, want %d", got.DefObjectType, cfg.DefObjectType)
	}
	if got.DefSampleRate != cfg.DefSampleRate {
		t.Errorf("DefSampleRate: got %d, want %d", got.DefSampleRate, cfg.DefSampleRate)
	}
	if got.OutputFormat != cfg.OutputFormat {
		t.Errorf("OutputFormat: got %d, want %d", got.OutputFormat, cfg.OutputFormat)
	}
	if got.DownMatrix != cfg.DownMatrix {
		t.Errorf("DownMatrix: got %v, want %v", got.DownMatrix, cfg.DownMatrix)
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

func TestDecoder_allocateChannelBuffers(t *testing.T) {
	dec := NewDecoder()

	// Allocate for 2 channels
	err := dec.allocateChannelBuffers(2)
	if err != nil {
		t.Fatalf("allocateChannelBuffers failed: %v", err)
	}

	// Check timeOut buffers
	for ch := 0; ch < 2; ch++ {
		if dec.timeOut[ch] == nil {
			t.Errorf("timeOut[%d] not allocated", ch)
		}
		if len(dec.timeOut[ch]) != int(dec.frameLength) {
			t.Errorf("timeOut[%d] length: got %d, want %d", ch, len(dec.timeOut[ch]), dec.frameLength)
		}
	}

	// Check fbIntermed buffers
	for ch := 0; ch < 2; ch++ {
		if dec.fbIntermed[ch] == nil {
			t.Errorf("fbIntermed[%d] not allocated", ch)
		}
		if len(dec.fbIntermed[ch]) != int(dec.frameLength) {
			t.Errorf("fbIntermed[%d] length: got %d, want %d", ch, len(dec.fbIntermed[ch]), dec.frameLength)
		}
	}
}

func TestDecoder_allocateChannelBuffers_Idempotent(t *testing.T) {
	dec := NewDecoder()

	// Allocate for 2 channels
	err := dec.allocateChannelBuffers(2)
	if err != nil {
		t.Fatalf("first allocateChannelBuffers failed: %v", err)
	}

	// Store pointers to original buffers
	origTimeOut0 := dec.timeOut[0]
	origTimeOut1 := dec.timeOut[1]
	origFbIntermed0 := dec.fbIntermed[0]
	origFbIntermed1 := dec.fbIntermed[1]

	// Call again - should be idempotent (no-op)
	err = dec.allocateChannelBuffers(2)
	if err != nil {
		t.Fatalf("second allocateChannelBuffers failed: %v", err)
	}

	// Verify buffers are the same (not reallocated)
	if &dec.timeOut[0][0] != &origTimeOut0[0] {
		t.Error("timeOut[0] was reallocated, should be idempotent")
	}
	if &dec.timeOut[1][0] != &origTimeOut1[0] {
		t.Error("timeOut[1] was reallocated, should be idempotent")
	}
	if &dec.fbIntermed[0][0] != &origFbIntermed0[0] {
		t.Error("fbIntermed[0] was reallocated, should be idempotent")
	}
	if &dec.fbIntermed[1][0] != &origFbIntermed1[0] {
		t.Error("fbIntermed[1] was reallocated, should be idempotent")
	}
}

func TestDecoder_allocateChannelBuffers_TooManyChannels(t *testing.T) {
	dec := NewDecoder()

	// Try to allocate more than maxChannels
	err := dec.allocateChannelBuffers(maxChannels + 1)
	if err != ErrInvalidNumChannels {
		t.Errorf("expected ErrInvalidNumChannels, got %v", err)
	}
}

func TestDecoder_allocateChannelBuffers_ZeroChannels(t *testing.T) {
	dec := NewDecoder()

	// Zero channels should be valid (no-op)
	err := dec.allocateChannelBuffers(0)
	if err != nil {
		t.Errorf("allocateChannelBuffers(0) failed: %v", err)
	}
}

func TestDecoder_allocateLTPBuffers(t *testing.T) {
	dec := NewDecoder()

	// Allocate LTP for 2 channels
	dec.allocateLTPBuffers(2)

	// LTP buffers are 2*frameLength for overlap storage
	expectedLen := 2 * int(dec.frameLength)

	for ch := 0; ch < 2; ch++ {
		if dec.ltPredStat[ch] == nil {
			t.Errorf("ltPredStat[%d] not allocated", ch)
		}
		if len(dec.ltPredStat[ch]) != expectedLen {
			t.Errorf("ltPredStat[%d] length: got %d, want %d", ch, len(dec.ltPredStat[ch]), expectedLen)
		}
	}
}

func TestDecoder_Close(t *testing.T) {
	dec := NewDecoder()

	// Allocate some buffers
	_ = dec.allocateChannelBuffers(2)
	dec.allocateLTPBuffers(2)

	// Close should not panic
	dec.Close()

	// Verify buffers are nil'd (helps GC)
	for ch := 0; ch < 2; ch++ {
		if dec.timeOut[ch] != nil {
			t.Errorf("timeOut[%d] not cleared after Close", ch)
		}
	}
}
