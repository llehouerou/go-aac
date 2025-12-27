// internal/syntax/pce_test.go
package syntax

import "testing"

func TestProgramConfig_CoreFields(t *testing.T) {
	var pce ProgramConfig

	pce.ElementInstanceTag = 0
	pce.ObjectType = 0
	pce.SFIndex = 0
	pce.Channels = 0
}

func TestProgramConfig_ChannelElements(t *testing.T) {
	var pce ProgramConfig

	// Element counts
	pce.NumFrontChannelElements = 0
	pce.NumSideChannelElements = 0
	pce.NumBackChannelElements = 0
	pce.NumLFEChannelElements = 0
	pce.NumAssocDataElements = 0
	pce.NumValidCCElements = 0

	// Element arrays (up to 16 each)
	if len(pce.FrontElementIsCPE) != 16 {
		t.Errorf("FrontElementIsCPE should have 16 elements")
	}
	if len(pce.FrontElementTagSelect) != 16 {
		t.Errorf("FrontElementTagSelect should have 16 elements")
	}
}

func TestProgramConfig_MixdownInfo(t *testing.T) {
	var pce ProgramConfig

	pce.MonoMixdownPresent = false
	pce.MonoMixdownElementNumber = 0
	pce.StereoMixdownPresent = false
	pce.StereoMixdownElementNumber = 0
	pce.MatrixMixdownIdxPresent = false
	pce.PseudoSurroundEnable = false
	pce.MatrixMixdownIdx = 0
}

func TestProgramConfig_Comment(t *testing.T) {
	var pce ProgramConfig

	pce.CommentFieldBytes = 0
	if len(pce.CommentFieldData) != 257 {
		t.Errorf("CommentFieldData should have 257 bytes")
	}
}

func TestProgramConfig_DerivedFields(t *testing.T) {
	var pce ProgramConfig

	// Derived channel counts
	pce.NumFrontChannels = 0
	pce.NumSideChannels = 0
	pce.NumBackChannels = 0
	pce.NumLFEChannels = 0

	// Channel mapping
	if len(pce.SCEChannel) != 16 {
		t.Errorf("SCEChannel should have 16 elements")
	}
	if len(pce.CPEChannel) != 16 {
		t.Errorf("CPEChannel should have 16 elements")
	}
}
