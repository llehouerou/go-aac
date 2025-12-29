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

func TestRawDataBlockResult_Fields(t *testing.T) {
	result := &RawDataBlockResult{
		NumChannels:  2,
		NumElements:  1,
		FirstElement: IDCPE,
		HasLFE:       false,
	}

	if result.NumChannels != 2 {
		t.Errorf("NumChannels = %d, want 2", result.NumChannels)
	}
	if result.NumElements != 1 {
		t.Errorf("NumElements = %d, want 1", result.NumElements)
	}
	if result.FirstElement != IDCPE {
		t.Errorf("FirstElement = %d, want %d", result.FirstElement, IDCPE)
	}
	if result.HasLFE {
		t.Error("HasLFE = true, want false")
	}
}

func TestRawDataBlockResult_ElementCapacity(t *testing.T) {
	result := &RawDataBlockResult{}

	// Verify capacity matches MaxSyntaxElements
	if len(result.SCEResults) != MaxSyntaxElements {
		t.Errorf("SCEResults capacity = %d, want %d", len(result.SCEResults), MaxSyntaxElements)
	}
	if len(result.CPEResults) != MaxSyntaxElements {
		t.Errorf("CPEResults capacity = %d, want %d", len(result.CPEResults), MaxSyntaxElements)
	}
}
