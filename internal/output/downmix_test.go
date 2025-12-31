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

func TestDownmix5_1ToStereo(t *testing.T) {
	// 5.1 input: C=0, L=1, R=2, Ls=3, Rs=4, LFE=5
	// Single sample per channel
	input := [][]float32{
		{1000.0}, // Center
		{500.0},  // Front Left
		{600.0},  // Front Right
		{200.0},  // Rear Left (surround)
		{300.0},  // Rear Right (surround)
		{100.0},  // LFE (ignored by default)
	}
	channelMap := []uint8{0, 1, 2, 3, 4, 5}

	dm := NewDownmixer()

	// Left output = DM_MUL * (L + C*InvSqrt2 + Ls*InvSqrt2)
	// = 0.3204 * (500 + 1000*0.7071 + 200*0.7071)
	// = 0.3204 * (500 + 707.1 + 141.4)
	// = 0.3204 * 1348.5 ≈ 432.1
	left, right := dm.Downmix5_1ToStereo(input, channelMap, 0)

	expectedL := DownmixMul * (input[1][0] + input[0][0]*InvSqrt2 + input[3][0]*InvSqrt2)
	expectedR := DownmixMul * (input[2][0] + input[0][0]*InvSqrt2 + input[4][0]*InvSqrt2)

	if math.Abs(float64(left-expectedL)) > 0.01 {
		t.Errorf("left: got %v, want %v", left, expectedL)
	}
	if math.Abs(float64(right-expectedR)) > 0.01 {
		t.Errorf("right: got %v, want %v", right, expectedR)
	}
}

func TestDownmix5_1ToStereo_WithLFE(t *testing.T) {
	input := [][]float32{
		{1000.0}, // Center
		{500.0},  // Front Left
		{600.0},  // Front Right
		{200.0},  // Rear Left
		{300.0},  // Rear Right
		{400.0},  // LFE
	}
	channelMap := []uint8{0, 1, 2, 3, 4, 5}

	dm := &Downmixer{
		Enabled:    true,
		IncludeLFE: true,
		LFEGain:    0.5,
	}

	left, right := dm.Downmix5_1ToStereo(input, channelMap, 0)

	// Base downmix plus LFE contribution
	baseL := DownmixMul * (input[1][0] + input[0][0]*InvSqrt2 + input[3][0]*InvSqrt2)
	baseR := DownmixMul * (input[2][0] + input[0][0]*InvSqrt2 + input[4][0]*InvSqrt2)
	lfeContrib := input[5][0] * dm.LFEGain * DownmixMul

	expectedL := baseL + lfeContrib
	expectedR := baseR + lfeContrib

	if math.Abs(float64(left-expectedL)) > 0.01 {
		t.Errorf("left with LFE: got %v, want %v", left, expectedL)
	}
	if math.Abs(float64(right-expectedR)) > 0.01 {
		t.Errorf("right with LFE: got %v, want %v", right, expectedR)
	}
}

func TestDownmix5_1ToStereo_Disabled(t *testing.T) {
	input := [][]float32{
		{1000.0}, // Center
		{500.0},  // Front Left
		{600.0},  // Front Right
		{200.0},  // Rear Left
		{300.0},  // Rear Right
		{100.0},  // LFE
	}
	channelMap := []uint8{0, 1, 2, 3, 4, 5}

	dm := &Downmixer{Enabled: false}

	// When disabled, returns front L/R directly
	left, right := dm.Downmix5_1ToStereo(input, channelMap, 0)

	if left != 500.0 {
		t.Errorf("disabled left: got %v, want 500.0", left)
	}
	if right != 600.0 {
		t.Errorf("disabled right: got %v, want 600.0", right)
	}
}

func TestGetDownmixedSample(t *testing.T) {
	// 5.1 input
	input := [][]float32{
		{1000.0}, // Center
		{500.0},  // Front Left
		{600.0},  // Front Right
		{200.0},  // Rear Left
		{300.0},  // Rear Right
		{100.0},  // LFE
	}
	channelMap := []uint8{0, 1, 2, 3, 4, 5}

	dm := NewDownmixer()

	// Request left channel (0) - should return downmixed left
	leftSample := dm.GetDownmixedSample(input, 0, 0, channelMap)
	expectedL := DownmixMul * (input[1][0] + input[0][0]*InvSqrt2 + input[3][0]*InvSqrt2)
	if math.Abs(float64(leftSample-expectedL)) > 0.01 {
		t.Errorf("left sample: got %v, want %v", leftSample, expectedL)
	}

	// Request right channel (1) - should return downmixed right
	rightSample := dm.GetDownmixedSample(input, 1, 0, channelMap)
	expectedR := DownmixMul * (input[2][0] + input[0][0]*InvSqrt2 + input[4][0]*InvSqrt2)
	if math.Abs(float64(rightSample-expectedR)) > 0.01 {
		t.Errorf("right sample: got %v, want %v", rightSample, expectedR)
	}
}

func TestGetDownmixedSample_Passthrough(t *testing.T) {
	input := [][]float32{
		{100.0},
		{200.0},
	}
	channelMap := []uint8{0, 1}

	// When downmix is disabled, pass through unchanged
	dm := &Downmixer{Enabled: false}

	sample := dm.GetDownmixedSample(input, 0, 0, channelMap)
	if sample != 100.0 {
		t.Errorf("passthrough ch0: got %v, want 100.0", sample)
	}

	sample = dm.GetDownmixedSample(input, 1, 0, channelMap)
	if sample != 200.0 {
		t.Errorf("passthrough ch1: got %v, want 200.0", sample)
	}
}
