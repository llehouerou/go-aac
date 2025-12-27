// internal/syntax/tns_test.go
package syntax

import "testing"

func TestTNSInfo_FieldTypes(t *testing.T) {
	var tns TNSInfo

	// Verify field existence - 8 window groups max
	tns.NFilt[0] = 0
	tns.CoefRes[0] = 0
	tns.Length[0][0] = 0
	tns.Order[0][0] = 0
	tns.Direction[0][0] = 0
	tns.CoefCompress[0][0] = 0
	tns.Coef[0][0][0] = 0

	// Verify dimensions
	if len(tns.NFilt) != MaxWindowGroups {
		t.Errorf("NFilt should have %d elements", MaxWindowGroups)
	}
	if len(tns.Length) != MaxWindowGroups || len(tns.Length[0]) != 4 {
		t.Errorf("Length should be [%d][4]", MaxWindowGroups)
	}
	if len(tns.Coef) != MaxWindowGroups || len(tns.Coef[0]) != 4 || len(tns.Coef[0][0]) != 32 {
		t.Errorf("Coef should be [%d][4][32]", MaxWindowGroups)
	}
}

func TestTNSInfo_MaxFilters(t *testing.T) {
	// TNS allows up to 4 filters per window
	var tns TNSInfo
	for w := 0; w < MaxWindowGroups; w++ {
		for f := 0; f < 4; f++ {
			tns.Order[w][f] = 1
		}
	}
}
