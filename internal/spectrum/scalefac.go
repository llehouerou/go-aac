package spectrum

import "github.com/llehouerou/go-aac/internal/syntax"

// ApplyScaleFactorsConfig holds configuration for scale factor application.
type ApplyScaleFactorsConfig struct {
	// ICS contains window and scale factor information
	ICS *syntax.ICStream

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16
}
