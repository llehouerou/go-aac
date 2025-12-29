// internal/spectrum/pns_test.go
package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/huffman"
	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestPNSState_InitialValues(t *testing.T) {
	state := NewPNSState()

	// Initial state should be non-zero for proper RNG behavior
	if state.R1 == 0 || state.R2 == 0 {
		t.Error("PNSState should have non-zero initial values")
	}
}

func TestNewPNSState(t *testing.T) {
	state := NewPNSState()

	// Check default values match FAAD2's pre-warmed state
	if state.R1 != 0x2bb431ea {
		t.Errorf("PNSState.R1: got 0x%08x, want 0x2bb431ea", state.R1)
	}
	if state.R2 != 0x206155b7 {
		t.Errorf("PNSState.R2: got 0x%08x, want 0x206155b7", state.R2)
	}
}

func TestNoiseOffset(t *testing.T) {
	// Verify NoiseOffset constant matches FAAD2
	if NoiseOffset != 90 {
		t.Errorf("NoiseOffset: got %d, want 90", NoiseOffset)
	}
}

func TestPNSDecodeConfig(t *testing.T) {
	// Test that config struct can be created and fields set
	cfg := &PNSDecodeConfig{
		FrameLength: 1024,
		ChannelPair: true,
		ObjectType:  2,
	}

	if cfg.FrameLength != 1024 {
		t.Errorf("PNSDecodeConfig.FrameLength: got %d, want 1024", cfg.FrameLength)
	}
	if !cfg.ChannelPair {
		t.Error("PNSDecodeConfig.ChannelPair: got false, want true")
	}
	if cfg.ObjectType != 2 {
		t.Errorf("PNSDecodeConfig.ObjectType: got %d, want 2", cfg.ObjectType)
	}
	// ICSL and ICSR should be nil by default
	if cfg.ICSL != nil {
		t.Error("PNSDecodeConfig.ICSL: should be nil by default")
	}
	if cfg.ICSR != nil {
		t.Error("PNSDecodeConfig.ICSR: should be nil by default")
	}
}

func TestGenRandVector_BasicGeneration(t *testing.T) {
	spec := make([]float64, 16)
	r1, r2 := uint32(1), uint32(1)

	genRandVector(spec, 0, &r1, &r2)

	// Should have non-zero values
	allZero := true
	for _, v := range spec {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("genRandVector produced all zeros")
	}
}

func TestGenRandVector_EnergyNormalization(t *testing.T) {
	// With scale_factor = 0, the vector is normalized to unit total energy
	// 1/sqrt(energy) * 2^(0.25*0) = 1/sqrt(energy) * 1.0
	// So total energy after normalization should be approximately 1.0
	spec := make([]float64, 1024)
	r1, r2 := uint32(1), uint32(1)

	genRandVector(spec, 0, &r1, &r2)

	// Calculate total energy
	energy := 0.0
	for _, v := range spec {
		energy += v * v
	}

	// With scale_factor=0, 2^(0.25*0) = 1.0, so total energy should be ~1.0
	// Allow some tolerance
	if energy < 0.5 || energy > 2.0 {
		t.Errorf("total energy = %v, expected close to 1.0", energy)
	}
}

func TestGenRandVector_ScaleFactorScaling(t *testing.T) {
	// Higher scale factor = more energy
	spec1 := make([]float64, 256)
	spec2 := make([]float64, 256)

	r1a, r2a := uint32(1), uint32(1)
	r1b, r2b := uint32(1), uint32(1)

	genRandVector(spec1, 0, &r1a, &r2a) // scale_factor = 0
	genRandVector(spec2, 8, &r1b, &r2b) // scale_factor = 8 -> 2^2 = 4x energy

	energy1 := 0.0
	energy2 := 0.0
	for i := range spec1 {
		energy1 += spec1[i] * spec1[i]
		energy2 += spec2[i] * spec2[i]
	}

	// spec2 should have ~16x more energy (scale factor 8 -> 2^(0.25*8) = 4, 4^2 = 16)
	ratio := energy2 / energy1
	if ratio < 8 || ratio > 32 {
		t.Errorf("energy ratio = %v, expected ~16", ratio)
	}
}

func TestGenRandVector_Deterministic(t *testing.T) {
	// Same RNG state should produce same noise
	spec1 := make([]float64, 64)
	spec2 := make([]float64, 64)

	r1a, r2a := uint32(12345), uint32(67890)
	r1b, r2b := uint32(12345), uint32(67890)

	genRandVector(spec1, 10, &r1a, &r2a)
	genRandVector(spec2, 10, &r1b, &r2b)

	for i := range spec1 {
		if spec1[i] != spec2[i] {
			t.Errorf("spec[%d]: %v != %v", i, spec1[i], spec2[i])
		}
	}
}

func TestGenRandVector_EmptySlice(t *testing.T) {
	// Should handle empty slice without panic
	spec := make([]float64, 0)
	r1, r2 := uint32(1), uint32(1)

	// Should not panic
	genRandVector(spec, 0, &r1, &r2)
}

func TestGenRandVector_ScaleFactorClamping(t *testing.T) {
	// Test extreme scale factors don't cause overflow
	spec1 := make([]float64, 16)
	spec2 := make([]float64, 16)

	r1a, r2a := uint32(1), uint32(1)
	r1b, r2b := uint32(1), uint32(1)

	// Very negative scale factor
	genRandVector(spec1, -200, &r1a, &r2a)

	// Very positive scale factor
	genRandVector(spec2, 200, &r1b, &r2b)

	// Both should produce valid (non-NaN, non-Inf) values
	for i, v := range spec1 {
		if v != v { // NaN check
			t.Errorf("spec1[%d] is NaN", i)
		}
	}
	for i, v := range spec2 {
		if v != v { // NaN check
			t.Errorf("spec2[%d] is NaN", i)
		}
	}
}

func TestPNSDecode_NoNoiseBands(t *testing.T) {
	// When no noise bands exist, spectra should be unchanged
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          2,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffsetMax = 1024
	ics.SFBCB[0][0] = 1 // Normal codebook
	ics.SFBCB[0][1] = 1 // Normal codebook

	spec := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0}
	original := make([]float64, len(spec))
	copy(original, spec)

	state := NewPNSState()
	cfg := &PNSDecodeConfig{
		ICSL:        ics,
		FrameLength: 1024,
	}

	PNSDecode(spec, nil, state, cfg)

	// Should be unchanged
	for i := range spec {
		if spec[i] != original[i] {
			t.Errorf("spec[%d] = %v, want %v", i, spec[i], original[i])
		}
	}
}

func TestPNSDecode_SingleNoiseBand(t *testing.T) {
	// One noise band should be filled with random values
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffsetMax = 1024
	ics.SFBCB[0][0] = uint8(huffman.NoiseHCB) // Noise codebook
	ics.SFBCB[0][1] = 1                       // Normal codebook
	ics.ScaleFactors[0][0] = 0                // scale = 1.0

	spec := make([]float64, 8)

	state := NewPNSState()
	cfg := &PNSDecodeConfig{
		ICSL:        ics,
		FrameLength: 1024,
	}

	PNSDecode(spec, nil, state, cfg)

	// First SFB (0-3) should have noise (non-zero values)
	allZero := true
	for i := 0; i < 4; i++ {
		if spec[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("noise band should have non-zero values")
	}

	// Second SFB (4-7) should remain zero (not noise)
	for i := 4; i < 8; i++ {
		if spec[i] != 0 {
			t.Errorf("spec[%d] = %v, want 0 (non-noise band)", i, spec[i])
		}
	}
}

func TestPNSDecode_DeterministicWithState(t *testing.T) {
	// Same initial state should produce same noise
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 16
	ics.SWBOffsetMax = 1024
	ics.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	ics.ScaleFactors[0][0] = 5

	spec1 := make([]float64, 16)
	spec2 := make([]float64, 16)

	state1 := NewPNSState()
	state2 := NewPNSState()

	cfg := &PNSDecodeConfig{
		ICSL:        ics,
		FrameLength: 1024,
	}

	PNSDecode(spec1, nil, state1, cfg)
	PNSDecode(spec2, nil, state2, cfg)

	for i := range spec1 {
		if spec1[i] != spec2[i] {
			t.Errorf("spec[%d]: %v != %v", i, spec1[i], spec2[i])
		}
	}
}

func TestPNSDecode_StereoIndependent(t *testing.T) {
	// Without ms_used, left and right get independent noise
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
		MSMaskPresent:   0, // No M/S
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 16
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	icsL.ScaleFactors[0][0] = 0

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 16
	icsR.SWBOffsetMax = 1024
	icsR.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	icsR.ScaleFactors[0][0] = 0 // Same scale factor

	specL := make([]float64, 16)
	specR := make([]float64, 16)

	state := NewPNSState()
	cfg := &PNSDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
		ChannelPair: true,
	}

	PNSDecode(specL, specR, state, cfg)

	// Left and right should be different (independent noise)
	allSame := true
	for i := range specL {
		if specL[i] != specR[i] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("left and right should have independent noise")
	}
}

func TestPNSDecode_StereoCorrelated(t *testing.T) {
	// With ms_used=1, left and right get correlated noise (same pattern)
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
		MSMaskPresent:   1, // Per-band M/S
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 16
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	icsL.ScaleFactors[0][0] = 0
	icsL.MSUsed[0][0] = 1 // Correlated noise

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 16
	icsR.SWBOffsetMax = 1024
	icsR.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	icsR.ScaleFactors[0][0] = 0 // Same scale factor

	specL := make([]float64, 16)
	specR := make([]float64, 16)

	state := NewPNSState()
	cfg := &PNSDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
		ChannelPair: true,
	}

	PNSDecode(specL, specR, state, cfg)

	// Left and right should be the same (correlated noise)
	for i := range specL {
		if specL[i] != specR[i] {
			t.Errorf("spec[%d]: L=%v R=%v, should be equal (correlated)", i, specL[i], specR[i])
		}
	}
}

func TestPNSDecode_StereoCorrelated_MSMaskPresent2(t *testing.T) {
	// With ms_mask_present=2, all bands are correlated
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
		MSMaskPresent:   2, // All bands M/S
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 16
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	icsL.ScaleFactors[0][0] = 0
	// MSUsed not set, but ms_mask_present=2 implies all

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 16
	icsR.SWBOffsetMax = 1024
	icsR.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	icsR.ScaleFactors[0][0] = 0

	specL := make([]float64, 16)
	specR := make([]float64, 16)

	state := NewPNSState()
	cfg := &PNSDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
		ChannelPair: true,
	}

	PNSDecode(specL, specR, state, cfg)

	// Should be correlated
	for i := range specL {
		if specL[i] != specR[i] {
			t.Errorf("spec[%d]: L=%v R=%v, should be equal", i, specL[i], specR[i])
		}
	}
}

func TestPNSDecode_OnlyRightHasPNS(t *testing.T) {
	// Only right channel has PNS - no correlation possible
	icsL := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
		MSMaskPresent:   1,
	}
	icsL.WindowGroupLength[0] = 1
	icsL.SWBOffset[0] = 0
	icsL.SWBOffset[1] = 16
	icsL.SWBOffsetMax = 1024
	icsL.SFBCB[0][0] = 1 // Normal, not noise
	icsL.MSUsed[0][0] = 1

	icsR := &syntax.ICStream{
		NumWindowGroups: 1,
		MaxSFB:          1,
		WindowSequence:  syntax.OnlyLongSequence,
	}
	icsR.WindowGroupLength[0] = 1
	icsR.SWBOffset[0] = 0
	icsR.SWBOffset[1] = 16
	icsR.SWBOffsetMax = 1024
	icsR.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	icsR.ScaleFactors[0][0] = 0

	specL := make([]float64, 16)
	specR := make([]float64, 16)
	for i := range specL {
		specL[i] = float64(i + 1) // Non-zero to verify unchanged
	}

	state := NewPNSState()
	cfg := &PNSDecodeConfig{
		ICSL:        icsL,
		ICSR:        icsR,
		FrameLength: 1024,
		ChannelPair: true,
	}

	PNSDecode(specL, specR, state, cfg)

	// Left should be unchanged
	for i := range specL {
		expected := float64(i + 1)
		if specL[i] != expected {
			t.Errorf("specL[%d] = %v, want %v (unchanged)", i, specL[i], expected)
		}
	}

	// Right should have noise
	allZero := true
	for _, v := range specR {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("specR should have noise")
	}
}

func TestPNSDecode_ShortBlocks(t *testing.T) {
	// Test with 8 short windows grouped into 2 groups.
	// Note: Due to how FAAD2 clamps begin/end against swb_offset_max (which is 128 for
	// short blocks), noise is only generated for windows where base + swb_offset < swb_offset_max.
	// This means only window 0 gets noise when using the per-window swb_offset_max.
	//
	// To test proper short block handling, we use the full frame length as swb_offset_max,
	// which matches MS stereo behavior and allows noise generation in all windows.
	ics := &syntax.ICStream{
		NumWindowGroups: 2,
		NumWindows:      8,
		MaxSFB:          1,
		WindowSequence:  syntax.EightShortSequence,
	}
	ics.WindowGroupLength[0] = 4
	ics.WindowGroupLength[1] = 4
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	// Use frame length (1024) as swb_offset_max to allow noise in all windows
	ics.SWBOffsetMax = 1024
	ics.SFBCB[0][0] = uint8(huffman.NoiseHCB) // Noise in group 0
	ics.SFBCB[1][0] = uint8(huffman.NoiseHCB) // Noise in group 1
	ics.ScaleFactors[0][0] = 0
	ics.ScaleFactors[1][0] = 0

	spec := make([]float64, 1024)

	state := NewPNSState()
	cfg := &PNSDecodeConfig{
		ICSL:        ics,
		FrameLength: 1024,
	}

	PNSDecode(spec, nil, state, cfg)

	// Check that noise was generated in both groups
	// Group 0: windows 0-3, each 128 samples, first 4 coeffs per window
	for win := 0; win < 4; win++ {
		base := win * 128
		for i := 0; i < 4; i++ {
			if spec[base+i] == 0 {
				t.Errorf("spec[%d] (group 0, win %d) = 0, expected noise", base+i, win)
			}
		}
	}

	// Group 1: windows 4-7
	for win := 4; win < 8; win++ {
		base := win * 128
		for i := 0; i < 4; i++ {
			if spec[base+i] == 0 {
				t.Errorf("spec[%d] (group 1, win %d) = 0, expected noise", base+i, win)
			}
		}
	}
}

func TestPNSDecode_MixedGroups(t *testing.T) {
	// One group has PNS, one doesn't.
	// Using frame length as swb_offset_max to test multi-window behavior.
	ics := &syntax.ICStream{
		NumWindowGroups: 2,
		NumWindows:      8,
		MaxSFB:          1,
		WindowSequence:  syntax.EightShortSequence,
	}
	ics.WindowGroupLength[0] = 4
	ics.WindowGroupLength[1] = 4
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	// Use frame length (1024) as swb_offset_max to allow noise in all windows
	ics.SWBOffsetMax = 1024
	ics.SFBCB[0][0] = uint8(huffman.NoiseHCB) // Noise in group 0
	ics.SFBCB[1][0] = 1                       // Normal in group 1
	ics.ScaleFactors[0][0] = 0

	spec := make([]float64, 1024)
	// Mark group 1 with specific values
	for win := 4; win < 8; win++ {
		base := win * 128
		for i := 0; i < 4; i++ {
			spec[base+i] = 99.0
		}
	}

	state := NewPNSState()
	cfg := &PNSDecodeConfig{
		ICSL:        ics,
		FrameLength: 1024,
	}

	PNSDecode(spec, nil, state, cfg)

	// Group 0: should have noise (not 0 or 99)
	for win := 0; win < 4; win++ {
		base := win * 128
		for i := 0; i < 4; i++ {
			if spec[base+i] == 0 || spec[base+i] == 99.0 {
				t.Errorf("spec[%d] (group 0) = %v, expected random noise", base+i, spec[base+i])
			}
		}
	}

	// Group 1: should remain 99.0
	for win := 4; win < 8; win++ {
		base := win * 128
		for i := 0; i < 4; i++ {
			if spec[base+i] != 99.0 {
				t.Errorf("spec[%d] (group 1) = %v, want 99.0", base+i, spec[base+i])
			}
		}
	}
}
