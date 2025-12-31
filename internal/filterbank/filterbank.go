// Package filterbank implements the AAC filter bank (IMDCT + windowing + overlap-add).
//
// The filter bank is responsible for converting frequency-domain spectral
// coefficients (output from spectral reconstruction) into time-domain PCM samples.
//
// Key operations:
//   - IMDCT (Inverse Modified Discrete Cosine Transform)
//   - Windowing (sine or Kaiser-Bessel Derived windows)
//   - Overlap-add (50% overlap between consecutive frames)
//
// Window sequences:
//   - OnlyLongSequence: Standard long blocks (1024 samples)
//   - LongStartSequence: Transition from long to short blocks
//   - EightShortSequence: 8 short blocks (128 samples each)
//   - LongStopSequence: Transition from short to long blocks
//
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
	transfBuf   []float32 // 2*frameLength for IMDCT output
	windowedBuf []float32 // 2*frameLength for LTP windowed input
}

// NewFilterBank creates and initializes a FilterBank for the given frame length.
// Standard AAC uses frameLength=1024.
//
// Ported from: filter_bank_init() in ~/dev/faad2/libfaad/filtbank.c:48-92
func NewFilterBank(frameLen uint16) *FilterBank {
	nshort := frameLen / 8 // 128 for standard AAC

	fb := &FilterBank{
		mdct256:     mdct.NewMDCT(2 * nshort),   // 256 for short blocks
		mdct2048:    mdct.NewMDCT(2 * frameLen), // 2048 for long blocks
		transfBuf:   make([]float32, 2*frameLen),
		windowedBuf: make([]float32, 2*frameLen),
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

	nflat_ls := (nlong - nshort) / 2

	// Get windows for current and previous frame
	windowLong := GetLongWindow(int(windowShape))
	windowLongPrev := GetLongWindow(int(windowShapePrev))
	windowShort := GetShortWindow(int(windowShape))
	windowShortPrev := GetShortWindow(int(windowShapePrev))

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

	case syntax.LongStartSequence:
		// Perform IMDCT
		fb.mdct2048.IMDCT(freqIn, transfBuf)

		// Add second half of previous frame to windowed output of current frame
		for i := 0; i < nlong; i++ {
			timeOut[i] = overlap[i] + transfBuf[i]*windowLongPrev[i]
		}

		// Window the second half and save as overlap for next frame
		// Construct second half window using padding with 1's and 0's
		for i := 0; i < nflat_ls; i++ {
			overlap[i] = transfBuf[nlong+i]
		}
		for i := 0; i < nshort; i++ {
			overlap[nflat_ls+i] = transfBuf[nlong+nflat_ls+i] * windowShort[nshort-i-1]
		}
		for i := 0; i < nflat_ls; i++ {
			overlap[nflat_ls+nshort+i] = 0
		}

	case syntax.LongStopSequence:
		// Perform IMDCT
		fb.mdct2048.IMDCT(freqIn, transfBuf)

		// Add second half of previous frame to windowed output of current frame
		// Construct first half window using padding with 1's and 0's
		for i := 0; i < nflat_ls; i++ {
			timeOut[i] = overlap[i]
		}
		for i := 0; i < nshort; i++ {
			timeOut[nflat_ls+i] = overlap[nflat_ls+i] + transfBuf[nflat_ls+i]*windowShortPrev[i]
		}
		for i := 0; i < nflat_ls; i++ {
			timeOut[nflat_ls+nshort+i] = overlap[nflat_ls+nshort+i] + transfBuf[nflat_ls+nshort+i]
		}

		// Window the second half and save as overlap for next frame
		for i := 0; i < nlong; i++ {
			overlap[i] = transfBuf[nlong+i] * windowLong[nlong-1-i]
		}

	case syntax.EightShortSequence:
		trans := nshort / 2

		// Perform IMDCT for each of the 8 short blocks
		// Ported from: ~/dev/faad2/libfaad/filtbank.c:266-273
		for blk := 0; blk < 8; blk++ {
			fb.mdct256.IMDCT(freqIn[blk*nshort:], transfBuf[2*nshort*blk:])
		}

		// Add second half of previous frame to windowed output of current frame
		// Ported from: ~/dev/faad2/libfaad/filtbank.c:276-286
		for i := 0; i < nflat_ls; i++ {
			timeOut[i] = overlap[i]
		}
		for i := 0; i < nshort; i++ {
			timeOut[nflat_ls+i] = overlap[nflat_ls+i] +
				transfBuf[nshort*0+i]*windowShortPrev[i]
			timeOut[nflat_ls+1*nshort+i] = overlap[nflat_ls+nshort*1+i] +
				transfBuf[nshort*1+i]*windowShort[nshort-1-i] +
				transfBuf[nshort*2+i]*windowShort[i]
			timeOut[nflat_ls+2*nshort+i] = overlap[nflat_ls+nshort*2+i] +
				transfBuf[nshort*3+i]*windowShort[nshort-1-i] +
				transfBuf[nshort*4+i]*windowShort[i]
			timeOut[nflat_ls+3*nshort+i] = overlap[nflat_ls+nshort*3+i] +
				transfBuf[nshort*5+i]*windowShort[nshort-1-i] +
				transfBuf[nshort*6+i]*windowShort[i]
			if i < trans {
				timeOut[nflat_ls+4*nshort+i] = overlap[nflat_ls+nshort*4+i] +
					transfBuf[nshort*7+i]*windowShort[nshort-1-i] +
					transfBuf[nshort*8+i]*windowShort[i]
			}
		}

		// Window the second half and save as overlap for next frame
		// Ported from: ~/dev/faad2/libfaad/filtbank.c:289-299
		for i := 0; i < nshort; i++ {
			if i >= trans {
				overlap[nflat_ls+4*nshort+i-nlong] =
					transfBuf[nshort*7+i]*windowShort[nshort-1-i] +
						transfBuf[nshort*8+i]*windowShort[i]
			}
			overlap[nflat_ls+5*nshort+i-nlong] =
				transfBuf[nshort*9+i]*windowShort[nshort-1-i] +
					transfBuf[nshort*10+i]*windowShort[i]
			overlap[nflat_ls+6*nshort+i-nlong] =
				transfBuf[nshort*11+i]*windowShort[nshort-1-i] +
					transfBuf[nshort*12+i]*windowShort[i]
			overlap[nflat_ls+7*nshort+i-nlong] =
				transfBuf[nshort*13+i]*windowShort[nshort-1-i] +
					transfBuf[nshort*14+i]*windowShort[i]
			overlap[nflat_ls+8*nshort+i-nlong] =
				transfBuf[nshort*15+i] * windowShort[nshort-1-i]
		}
		for i := 0; i < nflat_ls; i++ {
			overlap[nflat_ls+nshort+i] = 0
		}

	default:
		panic("window sequence not implemented")
	}
}

// FilterBankLTP performs the forward filter bank operation for Long Term Prediction.
// This converts time-domain samples to frequency-domain MDCT coefficients.
//
// Parameters:
//   - windowSequence: One of OnlyLongSequence, LongStartSequence, LongStopSequence
//     (EIGHT_SHORT_SEQUENCE is NOT supported for LTP)
//   - windowShape: Current frame's window shape (SineWindow or KBDWindow)
//   - windowShapePrev: Previous frame's window shape
//   - inData: Input time samples (2*frameLen samples)
//   - outMDCT: Output MDCT coefficients (frameLen samples)
//
// Ported from: filter_bank_ltp() in ~/dev/faad2/libfaad/filtbank.c:337-408
func (fb *FilterBank) FilterBankLTP(
	windowSequence syntax.WindowSequence,
	windowShape uint8,
	windowShapePrev uint8,
	inData []float32,
	outMDCT []float32,
) {
	nlong := len(outMDCT)
	nshort := nlong / 8
	_ = nshort // Will be used for other window sequences

	windowedBuf := fb.windowedBuf

	// Clear windowed buffer
	for i := range windowedBuf[:2*nlong] {
		windowedBuf[i] = 0
	}

	// Get windows for current and previous frame
	windowLong := GetLongWindow(int(windowShape))
	windowLongPrev := GetLongWindow(int(windowShapePrev))

	// Use transfBuf as intermediate for forward MDCT output (needs 2*nlong)
	// Only the first nlong coefficients are used in AAC
	transfBuf := fb.transfBuf

	switch windowSequence {
	case syntax.OnlyLongSequence:
		// Window first half with previous window (ascending)
		// Window second half with current window (descending)
		// Ported from: filtbank.c:374-380
		for i := nlong - 1; i >= 0; i-- {
			windowedBuf[i] = inData[i] * windowLongPrev[i]
			windowedBuf[i+nlong] = inData[i+nlong] * windowLong[nlong-1-i]
		}
		// Forward MDCT (outputs 2*nlong samples to transfBuf)
		fb.mdct2048.Forward(windowedBuf[:2*nlong], transfBuf[:2*nlong])
		// Copy only the first nlong coefficients to output
		copy(outMDCT, transfBuf[:nlong])

	case syntax.LongStartSequence:
		panic("LongStartSequence not yet implemented in FilterBankLTP")

	case syntax.LongStopSequence:
		panic("LongStopSequence not yet implemented in FilterBankLTP")

	case syntax.EightShortSequence:
		panic("EightShortSequence is not supported for LTP")

	default:
		panic("unknown window sequence in FilterBankLTP")
	}
}
