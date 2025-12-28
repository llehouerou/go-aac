package syntax

import "testing"

func TestErrMSMaskReserved(t *testing.T) {
	if ErrMSMaskReserved == nil {
		t.Error("ErrMSMaskReserved should not be nil")
	}

	expectedMsg := "syntax: ms_mask_present value 3 is reserved"
	if ErrMSMaskReserved.Error() != expectedMsg {
		t.Errorf("Error message = %q, want %q", ErrMSMaskReserved.Error(), expectedMsg)
	}
}

func TestCCEErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrIntensityStereoInCCE",
			err:  ErrIntensityStereoInCCE,
			msg:  "syntax: intensity stereo not allowed in coupling channel element",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("error should not be nil")
			}
			if tt.err.Error() != tt.msg {
				t.Errorf("got %q, want %q", tt.err.Error(), tt.msg)
			}
		})
	}
}
