// internal/syntax/raw_data_block_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

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

func TestParseRawDataBlock_EmptyFrame(t *testing.T) {
	// A frame with only ID_END (0x7 = 0b111)
	// Bits: 111 (ID_END)
	// Padded to byte: 11100000 = 0xE0
	data := []byte{0xE0}
	r := bits.NewReader(data)

	cfg := &RawDataBlockConfig{
		SFIndex:              4,
		FrameLength:          1024,
		ObjectType:           ObjectTypeLC,
		ChannelConfiguration: 2,
	}
	drc := &DRCInfo{}

	result, err := ParseRawDataBlock(r, cfg, drc)
	if err != nil {
		t.Fatalf("ParseRawDataBlock() error = %v", err)
	}

	if result.NumChannels != 0 {
		t.Errorf("NumChannels = %d, want 0", result.NumChannels)
	}
	if result.NumElements != 0 {
		t.Errorf("NumElements = %d, want 0", result.NumElements)
	}
}
