// Package filterbank implements the AAC filter bank (IMDCT + windowing + overlap-add).
// Ported from: ~/dev/faad2/libfaad/filtbank.c
package filterbank

import (
	"github.com/llehouerou/go-aac/internal/mdct"
	"github.com/llehouerou/go-aac/internal/syntax"
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

// IFilterBank performs the inverse filter bank operation.
// This converts frequency-domain spectral data to time-domain samples.
//
// Parameters:
//   - windowSequence: One of OnlyLongSequence, LongStartSequence, EightShortSequence, LongStopSequence
//   - windowShape: Current frame's window shape (SineWindow or KBDWindow)
//   - windowShapePrev: Previous frame's window shape
//   - freqIn: Input spectral coefficients (frameLen samples)
//   - timeOut: Output time samples (frameLen samples)
//   - overlap: Overlap buffer from previous frame (frameLen samples, modified in place)
//
// The overlap buffer is modified to contain the overlap for the next frame.
//
// Ported from: ifilter_bank() in ~/dev/faad2/libfaad/filtbank.c:164-334
func (fb *FilterBank) IFilterBank(
	windowSequence syntax.WindowSequence,
	windowShape uint8,
	windowShapePrev uint8,
	freqIn []float32,
	timeOut []float32,
	overlap []float32,
) {
	nlong := len(freqIn)
	nshort := nlong / 8
	transfBuf := fb.transfBuf

	// Get windows for current and previous frame
	windowLong := GetLongWindow(int(windowShape))
	windowLongPrev := GetLongWindow(int(windowShapePrev))
	windowShort := GetShortWindow(int(windowShape))
	windowShortPrev := GetShortWindow(int(windowShapePrev))

	// Suppress unused variable warnings for cases we haven't implemented yet
	_ = windowShort
	_ = windowShortPrev
	_ = nshort

	switch windowSequence {
	case syntax.OnlyLongSequence:
		// Perform IMDCT
		fb.mdct2048.IMDCT(freqIn, transfBuf)

		// Add second half of previous frame to windowed output of current frame
		// time_out[i] = overlap[i] + transf_buf[i] * window_long_prev[i]
		for i := 0; i < nlong; i++ {
			timeOut[i] = overlap[i] + transfBuf[i]*windowLongPrev[i]
		}

		// Window the second half and save as overlap for next frame
		// overlap[i] = transf_buf[nlong+i] * window_long[nlong-1-i]
		for i := 0; i < nlong; i++ {
			overlap[i] = transfBuf[nlong+i] * windowLong[nlong-1-i]
		}

	default:
		// TODO: implement other window sequences
		panic("window sequence not implemented")
	}
}
