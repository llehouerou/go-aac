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

func TestReconstructSingleChannel_WithTNS(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          4,
		NumSWB:          4,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		TNSDataPresent:  true,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 8
	ics.SWBOffset[2] = 16
	ics.SWBOffset[3] = 24
	ics.SWBOffset[4] = 32
	ics.SWBOffsetMax = 1024
	ics.SFBCB[0][0] = 1
	ics.SFBCB[0][1] = 1
	ics.SFBCB[0][2] = 1
	ics.SFBCB[0][3] = 1
	ics.ScaleFactors[0][0] = 100
	ics.ScaleFactors[0][1] = 100
	ics.ScaleFactors[0][2] = 100
	ics.ScaleFactors[0][3] = 100

	// Setup simple TNS data
	ics.TNS.NFilt[0] = 1
	ics.TNS.Length[0][0] = 4
	ics.TNS.Order[0][0] = 1
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefRes[0] = 1
	ics.TNS.Coef[0][0][0] = 4

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     &syntax.Element{},
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData := make([]int16, 1024)
	for i := 0; i < 32; i++ {
		quantData[i] = 10
	}

	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}

	// TNS should have modified the spectrum
	// Just verify no error and non-zero output
	hasValue := false
	for i := 0; i < 32; i++ {
		if specData[i] != 0 {
			hasValue = true
			break
		}
	}
	if !hasValue {
		t.Error("spectrum should have non-zero values after TNS")
	}
}

func TestReconstructSingleChannel_MainProfile_ICPrediction(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups:      1,
		NumWindows:           1,
		MaxSFB:               4,
		NumSWB:               4,
		WindowSequence:       syntax.OnlyLongSequence,
		GlobalGain:           100,
		PredictorDataPresent: true,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffset[3] = 12
	ics.SWBOffset[4] = 16
	ics.SWBOffsetMax = 1024
	ics.SFBCB[0][0] = 1
	ics.SFBCB[0][1] = 1
	ics.SFBCB[0][2] = 1
	ics.SFBCB[0][3] = 1
	ics.ScaleFactors[0][0] = 100
	ics.ScaleFactors[0][1] = 100
	ics.ScaleFactors[0][2] = 100
	ics.ScaleFactors[0][3] = 100

	// Enable prediction for first 2 bands
	ics.Pred.PredictionUsed[0] = true
	ics.Pred.PredictionUsed[1] = true

	// Create predictor state
	predState := make([]PredState, 1024)
	ResetAllPredictors(predState, 1024)

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     &syntax.Element{},
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeMain, // MAIN profile
		SRIndex:     4,
		PNSState:    NewPNSState(),
		PredState:   predState,
	}

	quantData := make([]int16, 1024)
	for i := 0; i < 16; i++ {
		quantData[i] = int16(i + 1)
	}

	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}

	// Predictor state should be updated
	stateUpdated := false
	for i := 0; i < 16; i++ {
		if predState[i].R[0] != 0 || predState[i].R[1] != 0 {
			stateUpdated = true
			break
		}
	}
	if !stateUpdated {
		t.Error("predictor state should be updated")
	}
}

func TestReconstructSingleChannel_LTPProfile(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          4,
		NumSWB:          4,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffset[3] = 12
	ics.SWBOffset[4] = 16
	ics.SWBOffsetMax = 1024
	ics.SFBCB[0][0] = 1
	ics.ScaleFactors[0][0] = 100

	// Enable LTP
	ics.LTP.DataPresent = true
	ics.LTP.Lag = 1024
	ics.LTP.Coef = 4
	ics.LTP.LastBand = 4
	ics.LTP.LongUsed[0] = true
	ics.LTP.LongUsed[1] = true

	// Create LTP state (empty for this test)
	ltpState := make([]int16, 4*1024)

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     &syntax.Element{},
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLTP, // LTP profile
		SRIndex:     4,
		PNSState:    NewPNSState(),
		LTPState:    ltpState,
		// LTPFilterBank is nil, so LTP will be skipped
	}

	quantData := make([]int16, 1024)
	specData := make([]float64, 1024)

	// Should succeed (LTP skipped due to no filterbank)
	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}
}

func TestReconstructSingleChannel_ShortBlocks(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 2,
		NumWindows:      8,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.EightShortSequence,
		GlobalGain:      100,
	}
	ics.WindowGroupLength[0] = 4
	ics.WindowGroupLength[1] = 4
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffsetMax = 128 // 1024/8 for short blocks
	ics.SFBCB[0][0] = 1
	ics.SFBCB[0][1] = 1
	ics.SFBCB[1][0] = 1
	ics.SFBCB[1][1] = 1
	ics.ScaleFactors[0][0] = 100
	ics.ScaleFactors[0][1] = 100
	ics.ScaleFactors[1][0] = 100
	ics.ScaleFactors[1][1] = 100

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     &syntax.Element{},
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData := make([]int16, 1024)
	for i := 0; i < 64; i++ {
		quantData[i] = 1
	}

	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}

	// Verify output has values
	hasValue := false
	for i := 0; i < 64; i++ {
		if specData[i] != 0 {
			hasValue = true
			break
		}
	}
	if !hasValue {
		t.Error("short block processing should produce non-zero values")
	}
}

func TestReconstructChannelPair_BasicStereo(t *testing.T) {
	// Setup minimal ICS for both channels
	ics1 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          4,
		NumSWB:          4,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		MSMaskPresent:   0, // No M/S
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 4
	ics1.SWBOffset[2] = 8
	ics1.SWBOffset[3] = 12
	ics1.SWBOffset[4] = 16
	ics1.SWBOffsetMax = 1024
	for g := 0; g < 8; g++ {
		for sfb := 0; sfb < 51; sfb++ {
			ics1.SFBCB[g][sfb] = 1
			ics1.ScaleFactors[g][sfb] = 100
		}
	}

	ics2 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          4,
		NumSWB:          4,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 4
	ics2.SWBOffset[2] = 8
	ics2.SWBOffset[3] = 12
	ics2.SWBOffset[4] = 16
	ics2.SWBOffsetMax = 1024
	for g := 0; g < 8; g++ {
		for sfb := 0; sfb < 51; sfb++ {
			ics2.SFBCB[g][sfb] = 1
			ics2.ScaleFactors[g][sfb] = 100
		}
	}

	ele := &syntax.Element{
		CommonWindow: false,
	}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)
	quantData1[0] = 1
	quantData1[1] = 2
	quantData2[0] = 3
	quantData2[1] = 4

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// Verify non-zero output
	if specData1[0] == 0 || specData2[0] == 0 {
		t.Error("both channels should have non-zero output")
	}
}

func TestReconstructChannelPair_WithMSStereo(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		MSMaskPresent:   2, // All bands use M/S
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 4
	ics1.SWBOffset[2] = 8
	ics1.SWBOffsetMax = 1024
	ics1.SFBCB[0][0] = 1
	ics1.SFBCB[0][1] = 1
	ics1.ScaleFactors[0][0] = 100
	ics1.ScaleFactors[0][1] = 100

	ics2 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 4
	ics2.SWBOffset[2] = 8
	ics2.SWBOffsetMax = 1024
	ics2.SFBCB[0][0] = 1
	ics2.SFBCB[0][1] = 1
	ics2.ScaleFactors[0][0] = 100
	ics2.ScaleFactors[0][1] = 100

	ele := &syntax.Element{CommonWindow: true}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	// Create M/S encoded data: M=10, S=2
	// Expected output: L = M + S = 12, R = M - S = 8
	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)
	quantData1[0] = 10 // Mid
	quantData2[0] = 2  // Side

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// After M/S decode: L should be different from R
	// M/S transform: L = M + S, R = M - S
	// Since we have non-zero M and S, L != R
	if specData1[0] == specData2[0] {
		t.Error("M/S stereo should produce different L and R values")
	}

	// L should be > R since M > 0 and S > 0
	if specData1[0] <= specData2[0] {
		t.Errorf("L (%f) should be > R (%f) with positive M and S", specData1[0], specData2[0])
	}
}

func TestReconstructChannelPair_WithIntensityStereo(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		MSMaskPresent:   0, // No M/S
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 4
	ics1.SWBOffset[2] = 8
	ics1.SWBOffsetMax = 1024
	ics1.SFBCB[0][0] = 1
	ics1.SFBCB[0][1] = 1
	ics1.ScaleFactors[0][0] = 100
	ics1.ScaleFactors[0][1] = 100

	ics2 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 4
	ics2.SWBOffset[2] = 8
	ics2.SWBOffsetMax = 1024
	// First band: normal, second band: intensity stereo
	ics2.SFBCB[0][0] = 1
	ics2.SFBCB[0][1] = uint8(huffman.IntensityHCB) // 15 = intensity stereo
	ics2.ScaleFactors[0][0] = 100
	ics2.ScaleFactors[0][1] = 0 // IS scale factor

	ele := &syntax.Element{CommonWindow: false}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)
	// Left channel has data in band 1 (indices 4-7)
	quantData1[4] = 10
	quantData1[5] = 10
	// Right channel has no data in band 1 (will be copied from left via IS)

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// Second band (indices 4-7) should have IS-scaled copy in right channel
	if specData2[4] == 0 {
		t.Error("intensity stereo should copy scaled values from left to right")
	}
}

func TestReconstructChannelPair_ShortBlocks(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups: 2,
		NumWindows:      8,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.EightShortSequence,
		GlobalGain:      100,
	}
	ics1.WindowGroupLength[0] = 4
	ics1.WindowGroupLength[1] = 4
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 4
	ics1.SWBOffset[2] = 8
	ics1.SWBOffsetMax = 128
	ics1.SFBCB[0][0] = 1
	ics1.SFBCB[0][1] = 1
	ics1.SFBCB[1][0] = 1
	ics1.SFBCB[1][1] = 1
	ics1.ScaleFactors[0][0] = 100
	ics1.ScaleFactors[0][1] = 100
	ics1.ScaleFactors[1][0] = 100
	ics1.ScaleFactors[1][1] = 100

	ics2 := &syntax.ICStream{
		NumWindowGroups: 2,
		NumWindows:      8,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.EightShortSequence,
		GlobalGain:      100,
	}
	ics2.WindowGroupLength[0] = 4
	ics2.WindowGroupLength[1] = 4
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 4
	ics2.SWBOffset[2] = 8
	ics2.SWBOffsetMax = 128
	ics2.SFBCB[0][0] = 1
	ics2.SFBCB[0][1] = 1
	ics2.SFBCB[1][0] = 1
	ics2.SFBCB[1][1] = 1
	ics2.ScaleFactors[0][0] = 100
	ics2.ScaleFactors[0][1] = 100
	ics2.ScaleFactors[1][0] = 100
	ics2.ScaleFactors[1][1] = 100

	ele := &syntax.Element{CommonWindow: false}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)
	for i := 0; i < 64; i++ {
		quantData1[i] = 1
		quantData2[i] = 1
	}

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// Verify both channels have values
	hasValue1 := false
	hasValue2 := false
	for i := 0; i < 64; i++ {
		if specData1[i] != 0 {
			hasValue1 = true
		}
		if specData2[i] != 0 {
			hasValue2 = true
		}
	}
	if !hasValue1 || !hasValue2 {
		t.Error("short block processing should produce non-zero values in both channels")
	}
}

func TestReconstructChannelPair_WithTNS(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          4,
		NumSWB:          4,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		TNSDataPresent:  true,
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 8
	ics1.SWBOffset[2] = 16
	ics1.SWBOffset[3] = 24
	ics1.SWBOffset[4] = 32
	ics1.SWBOffsetMax = 1024
	ics1.SFBCB[0][0] = 1
	ics1.ScaleFactors[0][0] = 100
	ics1.TNS.NFilt[0] = 1
	ics1.TNS.Length[0][0] = 4
	ics1.TNS.Order[0][0] = 1
	ics1.TNS.Direction[0][0] = 0
	ics1.TNS.CoefRes[0] = 1
	ics1.TNS.Coef[0][0][0] = 4

	ics2 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          4,
		NumSWB:          4,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		TNSDataPresent:  true,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 8
	ics2.SWBOffset[2] = 16
	ics2.SWBOffset[3] = 24
	ics2.SWBOffset[4] = 32
	ics2.SWBOffsetMax = 1024
	ics2.SFBCB[0][0] = 1
	ics2.ScaleFactors[0][0] = 100
	ics2.TNS.NFilt[0] = 1
	ics2.TNS.Length[0][0] = 4
	ics2.TNS.Order[0][0] = 1
	ics2.TNS.Direction[0][0] = 0
	ics2.TNS.CoefRes[0] = 1
	ics2.TNS.Coef[0][0][0] = 4

	ele := &syntax.Element{CommonWindow: false}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)
	for i := 0; i < 32; i++ {
		quantData1[i] = 10
		quantData2[i] = 10
	}

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// TNS should have modified both channels
	hasValue1 := false
	hasValue2 := false
	for i := 0; i < 32; i++ {
		if specData1[i] != 0 {
			hasValue1 = true
		}
		if specData2[i] != 0 {
			hasValue2 = true
		}
	}
	if !hasValue1 || !hasValue2 {
		t.Error("TNS should produce non-zero values in both channels")
	}
}

func TestReconstructChannelPair_MainProfile_ICPrediction(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups:      1,
		NumWindows:           1,
		MaxSFB:               4,
		NumSWB:               4,
		WindowSequence:       syntax.OnlyLongSequence,
		GlobalGain:           100,
		PredictorDataPresent: true,
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 4
	ics1.SWBOffset[2] = 8
	ics1.SWBOffset[3] = 12
	ics1.SWBOffset[4] = 16
	ics1.SWBOffsetMax = 1024
	ics1.SFBCB[0][0] = 1
	ics1.ScaleFactors[0][0] = 100
	ics1.Pred.PredictionUsed[0] = true
	ics1.Pred.PredictionUsed[1] = true

	ics2 := &syntax.ICStream{
		NumWindowGroups:      1,
		NumWindows:           1,
		MaxSFB:               4,
		NumSWB:               4,
		WindowSequence:       syntax.OnlyLongSequence,
		GlobalGain:           100,
		PredictorDataPresent: true,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 4
	ics2.SWBOffset[2] = 8
	ics2.SWBOffset[3] = 12
	ics2.SWBOffset[4] = 16
	ics2.SWBOffsetMax = 1024
	ics2.SFBCB[0][0] = 1
	ics2.ScaleFactors[0][0] = 100
	ics2.Pred.PredictionUsed[0] = true
	ics2.Pred.PredictionUsed[1] = true

	predState1 := make([]PredState, 1024)
	predState2 := make([]PredState, 1024)
	ResetAllPredictors(predState1, 1024)
	ResetAllPredictors(predState2, 1024)

	ele := &syntax.Element{CommonWindow: false}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeMain,
		SRIndex:     4,
		PNSState:    NewPNSState(),
		PredState1:  predState1,
		PredState2:  predState2,
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)
	for i := 0; i < 16; i++ {
		quantData1[i] = int16(i + 1)
		quantData2[i] = int16(i + 1)
	}

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// Predictor state should be updated for both channels
	stateUpdated1 := false
	stateUpdated2 := false
	for i := 0; i < 16; i++ {
		if predState1[i].R[0] != 0 || predState1[i].R[1] != 0 {
			stateUpdated1 = true
		}
		if predState2[i].R[0] != 0 || predState2[i].R[1] != 0 {
			stateUpdated2 = true
		}
	}
	if !stateUpdated1 || !stateUpdated2 {
		t.Error("predictor state should be updated for both channels")
	}
}

func TestReconstructChannelPair_CorrelatedPNS(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		MSMaskPresent:   2, // All bands - enables PNS correlation
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 8
	ics1.SWBOffset[2] = 16
	ics1.SWBOffsetMax = 1024
	// Both bands are noise
	ics1.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	ics1.SFBCB[0][1] = uint8(huffman.NoiseHCB)
	ics1.ScaleFactors[0][0] = 0
	ics1.ScaleFactors[0][1] = 0

	ics2 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 8
	ics2.SWBOffset[2] = 16
	ics2.SWBOffsetMax = 1024
	// Both bands are noise
	ics2.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	ics2.SFBCB[0][1] = uint8(huffman.NoiseHCB)
	ics2.ScaleFactors[0][0] = 0
	ics2.ScaleFactors[0][1] = 0

	ele := &syntax.Element{CommonWindow: true}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// With ms_mask_present=2, PNS should be correlated (same random sequence)
	// The noise values should be proportional (same pattern, possibly different scale)
	// Check that the ratio is consistent across samples
	if specData1[0] == 0 || specData2[0] == 0 {
		t.Skip("PNS generated zero - need non-zero for correlation test")
	}

	ratio := specData1[0] / specData2[0]
	for i := 1; i < 8; i++ {
		if specData2[i] == 0 {
			continue
		}
		thisRatio := specData1[i] / specData2[i]
		// Allow small tolerance for floating point
		if (thisRatio-ratio)/ratio > 0.01 || (thisRatio-ratio)/ratio < -0.01 {
			t.Errorf("sample %d: ratio %f differs from expected %f", i, thisRatio, ratio)
		}
	}
}
