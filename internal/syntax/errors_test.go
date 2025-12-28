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
