// internal/spectrum/reconstruct.go
package spectrum

import (
	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/syntax"
)

// ReconstructSingleChannelConfig holds configuration for single channel reconstruction.
//
// Ported from: reconstruct_single_channel() parameters in ~/dev/faad2/libfaad/specrec.c:905-906
type ReconstructSingleChannelConfig struct {
	// ICS is the individual channel stream containing parsed syntax data
	ICS *syntax.ICStream

	// Element is the syntax element (SCE/LFE)
	Element *syntax.Element

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16

	// ObjectType is the AAC object type
	ObjectType aac.ObjectType

	// SRIndex is the sample rate index (0-15)
	SRIndex uint8

	// PredState is the predictor state for MAIN profile (nil if not MAIN)
	PredState []PredState

	// LTPState is the LTP state buffer for LTP profile (nil if not LTP)
	LTPState []int16

	// LTPFilterBank is the forward MDCT for LTP (nil if not LTP)
	LTPFilterBank ForwardMDCT

	// WindowShape is the current window shape
	WindowShape uint8

	// WindowShapePrev is the previous window shape
	WindowShapePrev uint8

	// PNSState is the PNS random number generator state
	PNSState *PNSState
}
