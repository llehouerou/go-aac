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

func TestDownmixFrame(t *testing.T) {
	// 5.1 input with 3 samples per channel
	input := [][]float32{
		{1000.0, 2000.0, 3000.0}, // Center
		{500.0, 1000.0, 1500.0},  // Front Left
		{600.0, 1200.0, 1800.0},  // Front Right
		{200.0, 400.0, 600.0},    // Rear Left
		{300.0, 600.0, 900.0},    // Rear Right
		{100.0, 200.0, 300.0},    // LFE
	}
	channelMap := []uint8{0, 1, 2, 3, 4, 5}

	dm := NewDownmixer()
	left, right := dm.DownmixFrame(input, channelMap, 3)

	if len(left) != 3 || len(right) != 3 {
		t.Fatalf("expected 3 samples, got left=%d, right=%d", len(left), len(right))
	}

	// Verify first sample
	expectedL0 := DownmixMul * (input[1][0] + input[0][0]*InvSqrt2 + input[3][0]*InvSqrt2)
	expectedR0 := DownmixMul * (input[2][0] + input[0][0]*InvSqrt2 + input[4][0]*InvSqrt2)

	if math.Abs(float64(left[0]-expectedL0)) > 0.01 {
		t.Errorf("left[0]: got %v, want %v", left[0], expectedL0)
	}
	if math.Abs(float64(right[0]-expectedR0)) > 0.01 {
		t.Errorf("right[0]: got %v, want %v", right[0], expectedR0)
	}

	// Verify last sample
	expectedL2 := DownmixMul * (input[1][2] + input[0][2]*InvSqrt2 + input[3][2]*InvSqrt2)
	expectedR2 := DownmixMul * (input[2][2] + input[0][2]*InvSqrt2 + input[4][2]*InvSqrt2)

	if math.Abs(float64(left[2]-expectedL2)) > 0.01 {
		t.Errorf("left[2]: got %v, want %v", left[2], expectedL2)
	}
	if math.Abs(float64(right[2]-expectedR2)) > 0.01 {
		t.Errorf("right[2]: got %v, want %v", right[2], expectedR2)
	}
}

func TestDownmixFrame_Disabled(t *testing.T) {
	input := [][]float32{
		{1000.0, 2000.0}, // Center
		{500.0, 1000.0},  // Front Left
		{600.0, 1200.0},  // Front Right
		{200.0, 400.0},   // Rear Left
		{300.0, 600.0},   // Rear Right
		{100.0, 200.0},   // LFE
	}
	channelMap := []uint8{0, 1, 2, 3, 4, 5}

	dm := &Downmixer{Enabled: false}
	left, right := dm.DownmixFrame(input, channelMap, 2)

	// When disabled, return front L/R directly
	if left[0] != 500.0 || left[1] != 1000.0 {
		t.Errorf("disabled left: got %v, want [500.0, 1000.0]", left)
	}
	if right[0] != 600.0 || right[1] != 1200.0 {
		t.Errorf("disabled right: got %v, want [600.0, 1200.0]", right)
	}
}

// TestDownmixer_MatchesPCMGetSample verifies that the new Downmixer produces
// identical results to the existing getSample function in pcm.go.
// This is a critical validation to ensure the port is correct.
func TestDownmixer_MatchesPCMGetSample(t *testing.T) {
	// 5.1 input with multiple samples
	input := [][]float32{
		{1000.0, 500.0, -200.0, 32767.0}, // Center
		{500.0, 250.0, -100.0, 16000.0},  // Front Left
		{600.0, 300.0, -120.0, 16500.0},  // Front Right
		{200.0, 100.0, -50.0, 8000.0},    // Rear Left
		{300.0, 150.0, -75.0, 8500.0},    // Rear Right
		{100.0, 50.0, -25.0, 4000.0},     // LFE (ignored by both)
	}
	channelMap := []uint8{0, 1, 2, 3, 4, 5}

	dm := NewDownmixer()

	for i := uint16(0); i < 4; i++ {
		// Get sample using old getSample function from pcm.go
		oldLeft := getSample(input, 0, i, true, channelMap)
		oldRight := getSample(input, 1, i, true, channelMap)

		// Get sample using new Downmixer
		newLeft := dm.GetDownmixedSample(input, 0, i, channelMap)
		newRight := dm.GetDownmixedSample(input, 1, i, channelMap)

		if math.Abs(float64(oldLeft-newLeft)) > 1e-6 {
			t.Errorf("sample %d left mismatch: getSample=%v, Downmixer=%v", i, oldLeft, newLeft)
		}
		if math.Abs(float64(oldRight-newRight)) > 1e-6 {
			t.Errorf("sample %d right mismatch: getSample=%v, Downmixer=%v", i, oldRight, newRight)
		}
	}
}

// TestDownmixer_MatchesPCMConstants verifies that the downmix constants match
// between downmix.go and pcm.go.
func TestDownmixer_MatchesPCMConstants(t *testing.T) {
	if DownmixMul != DMMul {
		t.Errorf("DownmixMul (%v) != DMMul (%v)", DownmixMul, DMMul)
	}
	if InvSqrt2 != RSQRT2 {
		t.Errorf("InvSqrt2 (%v) != RSQRT2 (%v)", InvSqrt2, RSQRT2)
	}
}

// TestDownmixer_MatchesPCMGetSample_Passthrough verifies that both implementations
// behave identically when downmix is disabled (passthrough mode).
func TestDownmixer_MatchesPCMGetSample_Passthrough(t *testing.T) {
	// Stereo input
	input := [][]float32{
		{100.0, 200.0, 300.0},
		{150.0, 250.0, 350.0},
	}
	channelMap := []uint8{0, 1}

	dm := &Downmixer{Enabled: false}

	for i := uint16(0); i < 3; i++ {
		// Get sample using old getSample function (downMatrix=false)
		oldCh0 := getSample(input, 0, i, false, channelMap)
		oldCh1 := getSample(input, 1, i, false, channelMap)

		// Get sample using new Downmixer (disabled)
		newCh0 := dm.GetDownmixedSample(input, 0, i, channelMap)
		newCh1 := dm.GetDownmixedSample(input, 1, i, channelMap)

		if oldCh0 != newCh0 {
			t.Errorf("sample %d ch0 passthrough mismatch: getSample=%v, Downmixer=%v", i, oldCh0, newCh0)
		}
		if oldCh1 != newCh1 {
			t.Errorf("sample %d ch1 passthrough mismatch: getSample=%v, Downmixer=%v", i, oldCh1, newCh1)
		}
	}
}

// TestDownmixer_EdgeCases verifies both implementations handle edge cases identically.
func TestDownmixer_EdgeCases(t *testing.T) {
	testCases := []struct {
		name  string
		input [][]float32
	}{
		{
			name: "all zeros",
			input: [][]float32{
				{0.0}, {0.0}, {0.0}, {0.0}, {0.0}, {0.0},
			},
		},
		{
			name: "max positive values",
			input: [][]float32{
				{32767.0}, {32767.0}, {32767.0}, {32767.0}, {32767.0}, {32767.0},
			},
		},
		{
			name: "max negative values",
			input: [][]float32{
				{-32768.0}, {-32768.0}, {-32768.0}, {-32768.0}, {-32768.0}, {-32768.0},
			},
		},
		{
			name: "center only",
			input: [][]float32{
				{10000.0}, {0.0}, {0.0}, {0.0}, {0.0}, {0.0},
			},
		},
		{
			name: "surrounds only",
			input: [][]float32{
				{0.0}, {0.0}, {0.0}, {5000.0}, {5000.0}, {0.0},
			},
		},
	}

	channelMap := []uint8{0, 1, 2, 3, 4, 5}
	dm := NewDownmixer()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldLeft := getSample(tc.input, 0, 0, true, channelMap)
			oldRight := getSample(tc.input, 1, 0, true, channelMap)

			newLeft := dm.GetDownmixedSample(tc.input, 0, 0, channelMap)
			newRight := dm.GetDownmixedSample(tc.input, 1, 0, channelMap)

			if math.Abs(float64(oldLeft-newLeft)) > 1e-6 {
				t.Errorf("left mismatch: getSample=%v, Downmixer=%v", oldLeft, newLeft)
			}
			if math.Abs(float64(oldRight-newRight)) > 1e-6 {
				t.Errorf("right mismatch: getSample=%v, Downmixer=%v", oldRight, newRight)
			}
		})
	}
}
