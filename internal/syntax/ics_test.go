// internal/syntax/ics_test.go
package syntax

import "testing"

func TestICStream_CoreFields(t *testing.T) {
	var ics ICStream

	// Core fields
	ics.MaxSFB = 0
	ics.GlobalGain = 0
	ics.NumSWB = 0
	ics.NumWindowGroups = 0
	ics.NumWindows = 0
	ics.WindowSequence = OnlyLongSequence
	ics.WindowShape = 0
	ics.ScaleFactorGrouping = 0
}

func TestICStream_WindowGroupArrays(t *testing.T) {
	var ics ICStream

	// Window group length array - 8 groups max
	if len(ics.WindowGroupLength) != MaxWindowGroups {
		t.Errorf("WindowGroupLength should have %d elements", MaxWindowGroups)
	}

	// SFB offset arrays
	if len(ics.SectSFBOffset) != MaxWindowGroups {
		t.Errorf("SectSFBOffset should have %d window groups", MaxWindowGroups)
	}
	if len(ics.SectSFBOffset[0]) != 15*8 {
		t.Errorf("SectSFBOffset[n] should have 120 elements, got %d", len(ics.SectSFBOffset[0]))
	}

	if len(ics.SWBOffset) != 52 {
		t.Errorf("SWBOffset should have 52 elements, got %d", len(ics.SWBOffset))
	}
}

func TestICStream_SectionData(t *testing.T) {
	var ics ICStream

	// Section data arrays
	if len(ics.SectCB) != MaxWindowGroups || len(ics.SectCB[0]) != 15*8 {
		t.Errorf("SectCB dimensions wrong")
	}
	if len(ics.SectStart) != MaxWindowGroups || len(ics.SectStart[0]) != 15*8 {
		t.Errorf("SectStart dimensions wrong")
	}
	if len(ics.SectEnd) != MaxWindowGroups || len(ics.SectEnd[0]) != 15*8 {
		t.Errorf("SectEnd dimensions wrong")
	}
	if len(ics.SFBCB) != MaxWindowGroups || len(ics.SFBCB[0]) != 8*15 {
		t.Errorf("SFBCB dimensions wrong")
	}
	if len(ics.NumSec) != MaxWindowGroups {
		t.Errorf("NumSec should have %d elements", MaxWindowGroups)
	}
}

func TestICStream_ScaleFactors(t *testing.T) {
	var ics ICStream

	// Scale factors array - [window_groups][sfb]
	if len(ics.ScaleFactors) != MaxWindowGroups {
		t.Errorf("ScaleFactors should have %d window groups", MaxWindowGroups)
	}
	if len(ics.ScaleFactors[0]) != MaxSFB {
		t.Errorf("ScaleFactors[n] should have %d elements", MaxSFB)
	}
}

func TestICStream_MSInfo(t *testing.T) {
	var ics ICStream

	// M/S stereo data
	ics.MSMaskPresent = 0
	if len(ics.MSUsed) != MaxWindowGroups || len(ics.MSUsed[0]) != MaxSFB {
		t.Errorf("MSUsed dimensions should be [%d][%d]", MaxWindowGroups, MaxSFB)
	}
}

func TestICStream_EmbeddedStructs(t *testing.T) {
	var ics ICStream

	// Verify embedded structs exist
	_ = ics.Pul.NumberPulse
	_ = ics.TNS.NFilt[0]
}

func TestICStream_Flags(t *testing.T) {
	var ics ICStream

	// Parsing flags
	ics.NoiseUsed = false
	ics.IsUsed = false
	ics.PulseDataPresent = false
	ics.TNSDataPresent = false
	ics.GainControlDataPresent = false
	ics.PredictorDataPresent = false
}

func TestICStream_LTPFields(t *testing.T) {
	var ics ICStream

	// LTP info (for LTP profile)
	_ = ics.LTP.DataPresent
	_ = ics.LTP2.DataPresent
}
