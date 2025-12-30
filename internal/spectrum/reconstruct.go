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

// ReconstructSingleChannel performs spectral reconstruction for a single channel.
// This is the main entry point for converting parsed syntax data and quantized
// spectral coefficients into dequantized, processed spectral data ready for
// the filter bank.
//
// Processing order:
// 1. Pulse decode (if present, long blocks only)
// 2. Inverse quantization (|x|^(4/3))
// 3. Apply scale factors (multiply by 2^((sf-100)/4))
// 4. PNS decode (noise substitution)
// 5. IC Prediction (MAIN profile only)
// 6. PNS reset pred state (MAIN profile only)
// 7. LTP prediction (LTP profile only)
// 8. TNS decode (temporal noise shaping)
//
// Ported from: reconstruct_single_channel() in ~/dev/faad2/libfaad/specrec.c:905-1129
func ReconstructSingleChannel(quantData []int16, specData []float64, cfg *ReconstructSingleChannelConfig) error {
	ics := cfg.ICS
	frameLen := cfg.FrameLength

	// 1. Pulse decode (long blocks only)
	if ics.PulseDataPresent {
		if ics.WindowSequence == syntax.EightShortSequence {
			return syntax.ErrPulseInShortBlock
		}
		if err := PulseDecode(ics, quantData, frameLen); err != nil {
			return err
		}
	}

	// 2. Inverse quantization: spec[i] = sign(quant[i]) * |quant[i]|^(4/3)
	if err := InverseQuantize(quantData, specData); err != nil {
		return err
	}

	// 3. Apply scale factors: spec[i] *= 2^((sf-100)/4)
	ApplyScaleFactors(specData, &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: frameLen,
	})

	// 4. PNS decode (generate noise for noise bands)
	if cfg.PNSState != nil {
		PNSDecode(specData, nil, cfg.PNSState, &PNSDecodeConfig{
			ICSL:        ics,
			ICSR:        nil,
			FrameLength: frameLen,
			ChannelPair: false,
			ObjectType:  uint8(cfg.ObjectType),
		})
	}

	// 5 & 6. IC Prediction (MAIN profile only)
	if cfg.ObjectType == aac.ObjectTypeMain && cfg.PredState != nil {
		// Convert float64 to float32 for IC prediction
		specData32 := make([]float32, len(specData))
		for i, v := range specData {
			specData32[i] = float32(v)
		}

		ICPrediction(ics, specData32, cfg.PredState, frameLen, cfg.SRIndex)

		// Convert back to float64
		for i, v := range specData32 {
			specData[i] = float64(v)
		}

		// Reset predictors for PNS bands
		PNSResetPredState(ics, cfg.PredState)
	}

	// 7. LTP prediction (LTP profile only)
	if IsLTPObjectType(cfg.ObjectType) && cfg.LTPState != nil {
		LTPPredictionWithMDCT(specData, cfg.LTPState, cfg.LTPFilterBank, &LTPConfig{
			ICS:             ics,
			LTP:             &ics.LTP,
			SRIndex:         cfg.SRIndex,
			ObjectType:      cfg.ObjectType,
			FrameLength:     frameLen,
			WindowShape:     cfg.WindowShape,
			WindowShapePrev: cfg.WindowShapePrev,
		})
	}

	// 8. TNS decode (temporal noise shaping)
	if ics.TNSDataPresent {
		TNSDecodeFrame(specData, &TNSDecodeConfig{
			ICS:         ics,
			SRIndex:     cfg.SRIndex,
			ObjectType:  cfg.ObjectType,
			FrameLength: frameLen,
		})
	}

	return nil
}
