// internal/output/drc_test.go
package output

import "testing"

func TestDRCRefLevel(t *testing.T) {
	// DRC_REF_LEVEL = 20 * 4 = 80 (represents -20 dB)
	// Source: ~/dev/faad2/libfaad/drc.h:38
	if DRCRefLevel != 80 {
		t.Errorf("DRCRefLevel: got %d, want 80", DRCRefLevel)
	}
}

func TestNewDRC(t *testing.T) {
	drc := NewDRC(0.5, 0.75)

	if drc.Cut != 0.5 {
		t.Errorf("Cut: got %v, want 0.5", drc.Cut)
	}
	if drc.Boost != 0.75 {
		t.Errorf("Boost: got %v, want 0.75", drc.Boost)
	}
}
