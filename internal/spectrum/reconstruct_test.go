// internal/spectrum/reconstruct_test.go
package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestReconstructSingleChannelConfig_Defaults(t *testing.T) {
	ics := &syntax.ICStream{}
	ele := &syntax.Element{}

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4, // 44100 Hz
	}

	if cfg.FrameLength != 1024 {
		t.Errorf("FrameLength: got %d, want 1024", cfg.FrameLength)
	}
	if cfg.ObjectType != aac.ObjectTypeLC {
		t.Errorf("ObjectType: got %d, want %d", cfg.ObjectType, aac.ObjectTypeLC)
	}
}
