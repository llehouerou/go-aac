// Package filterbank implements the AAC filter bank (IMDCT + windowing + overlap-add).
// Ported from: ~/dev/faad2/libfaad/filtbank.c
package filterbank

import (
	"github.com/llehouerou/go-aac/internal/mdct"
)

// FilterBank holds state for inverse filter bank operations.
// It contains pre-initialized MDCT instances for short and long blocks.
//
// Ported from: fb_info struct in ~/dev/faad2/libfaad/structs.h:67-83
type FilterBank struct {
	mdct256  *mdct.MDCT // For short blocks (256-sample IMDCT)
	mdct2048 *mdct.MDCT // For long blocks (2048-sample IMDCT)

	// Internal buffers (reused to avoid allocations)
	transfBuf []float32 // 2*frameLength for IMDCT output
}

// NewFilterBank creates and initializes a FilterBank for the given frame length.
// Standard AAC uses frameLength=1024.
//
// Ported from: filter_bank_init() in ~/dev/faad2/libfaad/filtbank.c:48-92
func NewFilterBank(frameLen uint16) *FilterBank {
	nshort := frameLen / 8 // 128 for standard AAC

	fb := &FilterBank{
		mdct256:   mdct.NewMDCT(2 * nshort),   // 256 for short blocks
		mdct2048:  mdct.NewMDCT(2 * frameLen), // 2048 for long blocks
		transfBuf: make([]float32, 2*frameLen),
	}

	return fb
}
