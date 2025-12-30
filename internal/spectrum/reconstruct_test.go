// internal/spectrum/reconstruct_test.go
package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestReconstructSingleChannelConfig_Defaults(t *testing.T) {
	ics := &syntax.ICStream{}
	ele := &syntax.Element{}

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4, // 44100 Hz
	}

	if cfg.FrameLength != 1024 {
		t.Errorf("FrameLength: got %d, want 1024", cfg.FrameLength)
	}
	if cfg.ObjectType != aac.ObjectTypeLC {
		t.Errorf("ObjectType: got %d, want %d", cfg.ObjectType, aac.ObjectTypeLC)
	}
}

func TestReconstructSingleChannel_BasicLC(t *testing.T) {
	// Setup minimal ICS for AAC-LC
	ics := &syntax.ICStream{
		NumWindowGroups:  1,
		NumWindows:       1,
		MaxSFB:           4,
		NumSWB:           4,
		WindowSequence:   syntax.OnlyLongSequence,
		GlobalGain:       100,
		PulseDataPresent: false,
		TNSDataPresent:   false,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffset[3] = 12
	ics.SWBOffset[4] = 16
	ics.SWBOffsetMax = 1024
	// Set codebooks to normal (not noise/intensity)
	for g := 0; g < 8; g++ {
		for sfb := 0; sfb < 51; sfb++ {
			ics.SFBCB[g][sfb] = 1 // Normal codebook
		}
	}
	// Set scale factors to 100 (neutral, multiplier = 1.0)
	for g := 0; g < 8; g++ {
		for sfb := 0; sfb < 51; sfb++ {
			ics.ScaleFactors[g][sfb] = 100
		}
	}

	ele := &syntax.Element{}

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	// Input: quantized data (small values for IQ table lookup)
	quantData := make([]int16, 1024)
	quantData[0] = 1
	quantData[1] = 2
	quantData[2] = -1
	quantData[3] = -2

	// Output buffer
	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}

	// Verify non-zero output for non-zero input
	if specData[0] == 0 {
		t.Error("specData[0] should be non-zero")
	}
	if specData[1] == 0 {
		t.Error("specData[1] should be non-zero")
	}
}
