// internal/spectrum/reconstruct_test.go
package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/huffman"
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

func TestReconstructSingleChannel_WithPulse(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups:  1,
		NumWindows:       1,
		MaxSFB:           4,
		NumSWB:           4,
		WindowSequence:   syntax.OnlyLongSequence,
		GlobalGain:       100,
		PulseDataPresent: true,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffset[3] = 12
	ics.SWBOffset[4] = 16
	ics.SWBOffsetMax = 1024
	// Normal codebook
	ics.SFBCB[0][0] = 1
	ics.SFBCB[0][1] = 1
	ics.SFBCB[0][2] = 1
	ics.SFBCB[0][3] = 1
	// Scale factor 100 = 1.0
	ics.ScaleFactors[0][0] = 100
	ics.ScaleFactors[0][1] = 100
	ics.ScaleFactors[0][2] = 100
	ics.ScaleFactors[0][3] = 100

	// Setup pulse data
	ics.Pul.NumberPulse = 0   // 1 pulse
	ics.Pul.PulseStartSFB = 0 // Start at SFB 0
	ics.Pul.PulseOffset[0] = 2
	ics.Pul.PulseAmp[0] = 5 // Add 5

	ele := &syntax.Element{}

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	// Input: quantized data
	quantData := make([]int16, 1024)
	quantData[0] = 1
	quantData[1] = 2
	quantData[2] = 3 // Will become 8 after pulse (3+5)
	quantData[3] = 4

	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}

	// Verify pulse was applied (quantData should be modified)
	if quantData[2] != 8 {
		t.Errorf("quantData[2] after pulse: got %d, want 8", quantData[2])
	}
}

func TestReconstructSingleChannel_PulseInShortBlock_Error(t *testing.T) {
	ics := &syntax.ICStream{
		WindowSequence:   syntax.EightShortSequence,
		PulseDataPresent: true,
	}

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     &syntax.Element{},
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
	}

	quantData := make([]int16, 1024)
	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err == nil {
		t.Error("expected error for pulse in short block")
	}
	if err != syntax.ErrPulseInShortBlock {
		t.Errorf("got error %v, want ErrPulseInShortBlock", err)
	}
}

func TestReconstructSingleChannel_WithNoiseBand(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 8
	ics.SWBOffset[2] = 16
	ics.SWBOffsetMax = 1024

	// First band: noise, second band: normal
	ics.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	ics.SFBCB[0][1] = 1
	ics.ScaleFactors[0][0] = 0 // Noise scale
	ics.ScaleFactors[0][1] = 100

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     &syntax.Element{},
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData := make([]int16, 1024)
	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}

	// First 8 samples should have noise (non-zero)
	hasNoise := false
	for i := 0; i < 8; i++ {
		if specData[i] != 0 {
			hasNoise = true
			break
		}
	}
	if !hasNoise {
		t.Error("noise band should have non-zero values")
	}
}
