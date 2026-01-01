package aac

import "testing"

// TestErrorMessages verifies error messages match FAAD2 exactly.
// Source: ~/dev/faad2/libfaad/error.c:34-69
func TestErrorMessages(t *testing.T) {
	// Expected messages from FAAD2 error.c
	expected := []string{
		"No error",
		"Gain control not yet implemented",
		"Pulse coding not allowed in short blocks",
		"Invalid huffman codebook",
		"Scalefactor out of range",
		"Unable to find ADTS syncword",
		"Channel coupling not yet implemented",
		"Channel configuration not allowed in error resilient frame",
		"Bit error in error resilient scalefactor decoding",
		"Error decoding huffman scalefactor (bitstream error)",
		"Error decoding huffman codeword (bitstream error)",
		"Non existent huffman codebook number found",
		"Invalid number of channels",
		"Maximum number of bitstream elements exceeded",
		"Input data buffer too small",
		"Array index out of range",
		"Maximum number of scalefactor bands exceeded",
		"Quantised value out of range",
		"LTP lag out of range",
		"Invalid SBR parameter decoded",
		"SBR called without being initialised",
		"Unexpected channel configuration change",
		"Error in program_config_element",
		"First SBR frame is not the same as first AAC frame",
		"Unexpected fill element with SBR data",
		"Not all elements were provided with SBR data",
		"LTP decoding not available",
		"Output data buffer too small",
		"CRC error in DRM data",
		"PNS not allowed in DRM data stream",
		"No standard extension payload allowed in DRM",
		"PCE shall be the first element in a frame",
		"Bitstream value not allowed by specification",
		"MAIN prediction not initialised",
	}

	if len(expected) != 34 {
		t.Fatalf("expected 34 error messages, got %d", len(expected))
	}

	for i, want := range expected {
		err := Error(i)
		got := err.Error()
		if got != want {
			t.Errorf("Error(%d).Error() = %q, want %q", i, got, want)
		}
	}
}

func TestErrorCode(t *testing.T) {
	tests := []struct {
		code     int
		wantCode int
	}{
		{0, 0},
		{1, 1},
		{33, 33},
	}

	for _, tt := range tests {
		err := Error(tt.code)
		if int(err) != tt.wantCode {
			t.Errorf("Error(%d) code = %d, want %d", tt.code, int(err), tt.wantCode)
		}
	}
}

func TestErrNone(t *testing.T) {
	if ErrNone != Error(0) {
		t.Error("ErrNone should be Error(0)")
	}
	if ErrNone.Error() != "No error" {
		t.Error("ErrNone.Error() should be 'No error'")
	}
}

func TestInitErrors(t *testing.T) {
	errors := []Error{
		ErrNilDecoder,
		ErrNilBuffer,
		ErrBufferTooSmall,
		ErrUnsupportedObjectType,
		ErrInvalidSampleRate,
	}

	for _, e := range errors {
		if e.Error() == "" {
			t.Errorf("Error %d has empty message", e)
		}
	}
}

func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		code Error
		want string
	}{
		{ErrNone, "No error"},
		{ErrGainControlNotImplemented, "Gain control not yet implemented"},
		{ErrInvalidNumChannels, "Invalid number of channels"},
		{Error(255), "unknown error"}, // Unknown error code
	}

	for _, tt := range tests {
		got := GetErrorMessage(tt.code)
		if got != tt.want {
			t.Errorf("GetErrorMessage(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
