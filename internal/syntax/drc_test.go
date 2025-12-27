// internal/syntax/drc_test.go
package syntax

import "testing"

func TestDRCInfo_Fields(t *testing.T) {
	var drc DRCInfo

	drc.Present = false
	drc.NumBands = 0
	drc.PCEInstanceTag = 0
	drc.ExcludedChnsPresent = false
	drc.ProgRefLevel = 0
}

func TestDRCInfo_Bands(t *testing.T) {
	var drc DRCInfo

	// Up to 17 DRC bands
	if len(drc.BandTop) != 17 {
		t.Errorf("BandTop should have 17 elements")
	}
	if len(drc.DynRngSgn) != 17 {
		t.Errorf("DynRngSgn should have 17 elements")
	}
	if len(drc.DynRngCtl) != 17 {
		t.Errorf("DynRngCtl should have 17 elements")
	}
}

func TestDRCInfo_ExcludeMask(t *testing.T) {
	var drc DRCInfo

	if len(drc.ExcludeMask) != MaxChannels {
		t.Errorf("ExcludeMask should have %d elements", MaxChannels)
	}
	if len(drc.AdditionalExcludedChns) != MaxChannels {
		t.Errorf("AdditionalExcludedChns should have %d elements", MaxChannels)
	}
}

func TestDRCInfo_Control(t *testing.T) {
	var drc DRCInfo

	// Control parameters are float32
	drc.Ctrl1 = 1.0
	drc.Ctrl2 = 1.0
}
