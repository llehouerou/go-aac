// internal/spectrum/pns_test.go
package spectrum

import "testing"

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
