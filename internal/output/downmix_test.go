// internal/output/downmix_test.go
package output

import "testing"

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
