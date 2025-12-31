// internal/output/downmix_test.go
package output

import (
	"math"
	"testing"
)

func TestChannelConstants(t *testing.T) {
	// Verify channel positions match FAAD2 internal_channel ordering
	// Source: ~/dev/faad2/libfaad/output.c:45-61
	// For 5.1 downmix: [0]=C, [1]=L, [2]=R, [3]=Ls, [4]=Rs, [5]=LFE

	if ChannelCenter != 0 {
		t.Errorf("ChannelCenter: got %d, want 0", ChannelCenter)
	}
	if ChannelFrontLeft != 1 {
		t.Errorf("ChannelFrontLeft: got %d, want 1", ChannelFrontLeft)
	}
	if ChannelFrontRight != 2 {
		t.Errorf("ChannelFrontRight: got %d, want 2", ChannelFrontRight)
	}
	if ChannelRearLeft != 3 {
		t.Errorf("ChannelRearLeft: got %d, want 3", ChannelRearLeft)
	}
	if ChannelRearRight != 4 {
		t.Errorf("ChannelRearRight: got %d, want 4", ChannelRearRight)
	}
	if ChannelLFE != 5 {
		t.Errorf("ChannelLFE: got %d, want 5", ChannelLFE)
	}
}

func TestDownmixConstants(t *testing.T) {
	// DMMul = 1/(1+sqrt(2)+1/sqrt(2))
	// Source: ~/dev/faad2/libfaad/output.c:41
	expectedDMMul := float32(1.0 / (1.0 + math.Sqrt(2) + 1.0/math.Sqrt(2)))
	if math.Abs(float64(DownmixMul-expectedDMMul)) > 1e-6 {
		t.Errorf("DownmixMul: got %v, want %v", DownmixMul, expectedDMMul)
	}

	// InvSqrt2 = 1/sqrt(2) ≈ 0.7071
	// Source: ~/dev/faad2/libfaad/output.c:42
	expectedInvSqrt2 := float32(1.0 / math.Sqrt(2))
	if math.Abs(float64(InvSqrt2-expectedInvSqrt2)) > 1e-6 {
		t.Errorf("InvSqrt2: got %v, want %v", InvSqrt2, expectedInvSqrt2)
	}

	// Verify these match the existing DMMul and RSQRT2 from pcm.go
	if DownmixMul != DMMul {
		t.Errorf("DownmixMul != DMMul: %v vs %v", DownmixMul, DMMul)
	}
	if InvSqrt2 != RSQRT2 {
		t.Errorf("InvSqrt2 != RSQRT2: %v vs %v", InvSqrt2, RSQRT2)
	}
}

func TestNewDownmixer(t *testing.T) {
	// Default downmixer
	dm := NewDownmixer()
	if !dm.Enabled {
		t.Error("default downmixer should be enabled")
	}
	if dm.IncludeLFE {
		t.Error("default downmixer should not include LFE")
	}
}

func TestDownmixerConfig(t *testing.T) {
	dm := &Downmixer{
		Enabled:    true,
		IncludeLFE: true,
		LFEGain:    0.5,
	}

	if !dm.Enabled {
		t.Error("Enabled should be true")
	}
	if !dm.IncludeLFE {
		t.Error("IncludeLFE should be true")
	}
	if dm.LFEGain != 0.5 {
		t.Errorf("LFEGain: got %v, want 0.5", dm.LFEGain)
	}
}
