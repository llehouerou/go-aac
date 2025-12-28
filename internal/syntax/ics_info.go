// internal/syntax/ics_info.go
package syntax

import (
	"github.com/llehouerou/go-aac/internal/bits"
)

// ICSInfoConfig holds configuration needed for ICS info parsing.
type ICSInfoConfig struct {
	SFIndex      uint8  // Sample rate index (0-11)
	FrameLength  uint16 // Frame length (960 or 1024)
	ObjectType   uint8  // Audio object type
	CommonWindow bool   // True if CPE with common window
}

// ObjectType constants.
// Ported from: ~/dev/faad2/libfaad/neaacdec.h:85-100
const (
	ObjectTypeMain = 1  // AAC Main
	ObjectTypeLC   = 2  // AAC Low Complexity
	ObjectTypeSSR  = 3  // AAC Scalable Sample Rate
	ObjectTypeLTP  = 4  // AAC Long Term Prediction
	ObjectTypeSBR  = 5  // Spectral Band Replication
	ObjectTypeLD   = 23 // AAC Low Delay
)

// ParseICSInfo parses the ics_info() element from the bitstream.
// Ported from: ics_info() in ~/dev/faad2/libfaad/syntax.c:829-952
func ParseICSInfo(r *bits.Reader, ics *ICStream, cfg *ICSInfoConfig) error {
	// ics_reserved_bit - must be 0
	reserved := r.Get1Bit()
	if reserved != 0 {
		return ErrICSReservedBit
	}

	// window_sequence (2 bits)
	ics.WindowSequence = WindowSequence(r.GetBits(2))

	// window_shape (1 bit)
	ics.WindowShape = r.Get1Bit()

	// max_sfb depends on window sequence
	if ics.WindowSequence == EightShortSequence {
		// Short blocks: 4 bits for max_sfb
		ics.MaxSFB = uint8(r.GetBits(4))
		// scale_factor_grouping (7 bits)
		ics.ScaleFactorGrouping = uint8(r.GetBits(7))
	} else {
		// Long blocks: 6 bits for max_sfb
		ics.MaxSFB = uint8(r.GetBits(6))
	}

	// Calculate window grouping
	if err := WindowGroupingInfo(ics, cfg.SFIndex, cfg.FrameLength); err != nil {
		return err
	}

	// Predictor data (only for long blocks)
	if ics.WindowSequence != EightShortSequence {
		ics.PredictorDataPresent = r.Get1Bit() != 0

		if ics.PredictorDataPresent {
			if cfg.ObjectType == ObjectTypeMain {
				// MAIN profile: MPEG-2 style prediction
				if err := parseMainPrediction(r, ics, cfg.SFIndex); err != nil {
					return err
				}
			} else if cfg.ObjectType < ERObjectStart {
				// LTP profile: Long Term Prediction (non-ER objects only)
				if err := parseLTPPrediction(r, ics, cfg); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// parseMainPrediction parses MAIN profile prediction data.
// Ported from: ics_info() MAIN profile section in syntax.c:876-905
func parseMainPrediction(r *bits.Reader, ics *ICStream, sfIndex uint8) error {
	// Get max prediction SFB for this sample rate
	limit := maxPredSFB(sfIndex)
	if ics.MaxSFB < limit {
		limit = ics.MaxSFB
	}

	// predictor_reset (1 bit)
	predictorReset := r.Get1Bit() != 0
	var predictorResetGroup uint8
	if predictorReset {
		predictorResetGroup = uint8(r.GetBits(5))
	}

	// prediction_used flags for each SFB
	for sfb := uint8(0); sfb < limit; sfb++ {
		_ = r.Get1Bit() // prediction_used[sfb]
	}

	// Store in ICS if needed (currently not storing MAIN pred data)
	_ = predictorReset
	_ = predictorResetGroup

	return nil
}

// parseLTPPrediction parses LTP (Long Term Prediction) data.
// Ported from: ics_info() LTP section in syntax.c:907-947
func parseLTPPrediction(r *bits.Reader, ics *ICStream, cfg *ICSInfoConfig) error {
	// First LTP data
	ics.LTP.DataPresent = r.Get1Bit() != 0
	if ics.LTP.DataPresent {
		if err := ParseLTPData(r, ics, &ics.LTP, cfg.FrameLength); err != nil {
			return err
		}
	}

	// Second LTP data (only for common_window in CPE)
	if cfg.CommonWindow {
		ics.LTP2.DataPresent = r.Get1Bit() != 0
		if ics.LTP2.DataPresent {
			if err := ParseLTPData(r, ics, &ics.LTP2, cfg.FrameLength); err != nil {
				return err
			}
		}
	}

	return nil
}

// maxPredSFB returns the maximum SFB for MAIN profile prediction.
// Ported from: max_pred_sfb() in ~/dev/faad2/libfaad/common.c:73-85
func maxPredSFB(sfIndex uint8) uint8 {
	maxPredSFBTable := [12]uint8{
		33, 33, 38, 40, 40, 40, 41, 41, 37, 37, 37, 34,
	}
	if sfIndex >= 12 {
		return 0
	}
	return maxPredSFBTable[sfIndex]
}

// ParseLTPData parses LTP (Long Term Prediction) data from the bitstream.
// Ported from: ltp_data() in ~/dev/faad2/libfaad/syntax.c:2093-2152
func ParseLTPData(r *bits.Reader, ics *ICStream, ltp *LTPInfo, frameLength uint16) error {
	// ltp_lag (11 bits) - note: LD object type uses 10 bits, not handled here
	ltp.Lag = uint16(r.GetBits(11))

	// Validate lag (must not exceed 2 * frameLength)
	if ltp.Lag > frameLength<<1 {
		return ErrLTPLagTooLarge
	}

	// ltp_coef (3 bits)
	ltp.Coef = uint8(r.GetBits(3))

	if ics.WindowSequence == EightShortSequence {
		// Short windows: ltp_short_used flags for each window
		for w := uint8(0); w < ics.NumWindows; w++ {
			ltp.ShortUsed[w] = r.Get1Bit() != 0
			if ltp.ShortUsed[w] {
				ltp.ShortLagPresent[w] = r.Get1Bit() != 0
				if ltp.ShortLagPresent[w] {
					ltp.ShortLag[w] = uint8(r.GetBits(4))
				}
			}
		}
	} else {
		// Long window: ltp_long_used flags for each SFB up to last_band
		// last_band = min(max_sfb, MAX_LTP_SFB)
		ltp.LastBand = ics.MaxSFB
		if ltp.LastBand > MaxLTPSFB {
			ltp.LastBand = MaxLTPSFB
		}

		for sfb := uint8(0); sfb < ltp.LastBand; sfb++ {
			ltp.LongUsed[sfb] = r.Get1Bit() != 0
		}
	}

	return nil
}
