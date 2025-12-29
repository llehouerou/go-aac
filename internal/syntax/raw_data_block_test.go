// internal/syntax/raw_data_block_test.go
package syntax

import "testing"

func TestRawDataBlockConfig_Fields(t *testing.T) {
	cfg := &RawDataBlockConfig{
		SFIndex:              4, // 44100 Hz
		FrameLength:          1024,
		ObjectType:           ObjectTypeLC,
		ChannelConfiguration: 2, // Stereo
	}

	if cfg.SFIndex != 4 {
		t.Errorf("SFIndex = %d, want 4", cfg.SFIndex)
	}
	if cfg.FrameLength != 1024 {
		t.Errorf("FrameLength = %d, want 1024", cfg.FrameLength)
	}
	if cfg.ObjectType != ObjectTypeLC {
		t.Errorf("ObjectType = %d, want %d", cfg.ObjectType, ObjectTypeLC)
	}
	if cfg.ChannelConfiguration != 2 {
		t.Errorf("ChannelConfiguration = %d, want 2", cfg.ChannelConfiguration)
	}
}
