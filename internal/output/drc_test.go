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
