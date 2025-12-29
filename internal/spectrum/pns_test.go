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
