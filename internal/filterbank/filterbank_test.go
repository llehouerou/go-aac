package filterbank

import "testing"

func TestNewFilterBank(t *testing.T) {
	fb := NewFilterBank(1024)
	if fb == nil {
		t.Fatal("expected non-nil FilterBank")
	}
	if fb.mdct256 == nil {
		t.Error("expected mdct256 to be initialized")
	}
	if fb.mdct2048 == nil {
		t.Error("expected mdct2048 to be initialized")
	}
}
