// internal/syntax/ltp_test.go
package syntax

import "testing"

func TestLTPInfo_Fields(t *testing.T) {
	var ltp LTPInfo

	ltp.LastBand = 0
	ltp.DataPresent = false
	ltp.Lag = 0
	ltp.LagUpdate = false
	ltp.Coef = 0
}

func TestLTPInfo_LongWindow(t *testing.T) {
	var ltp LTPInfo

	if len(ltp.LongUsed) != MaxSFB {
		t.Errorf("LongUsed should have %d elements", MaxSFB)
	}
}

func TestLTPInfo_ShortWindows(t *testing.T) {
	var ltp LTPInfo

	// 8 short windows
	if len(ltp.ShortUsed) != 8 {
		t.Errorf("ShortUsed should have 8 elements")
	}
	if len(ltp.ShortLagPresent) != 8 {
		t.Errorf("ShortLagPresent should have 8 elements")
	}
	if len(ltp.ShortLag) != 8 {
		t.Errorf("ShortLag should have 8 elements")
	}
}
