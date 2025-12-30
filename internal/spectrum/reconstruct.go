// internal/spectrum/reconstruct.go
package spectrum

import (
	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/syntax"
)

// ReconstructChannelPairConfig holds configuration for channel pair reconstruction.
//
// Ported from: reconstruct_channel_pair() parameters in ~/dev/faad2/libfaad/specrec.c:1131-1132
type ReconstructChannelPairConfig struct {
	// ICS1 is the first channel's individual channel stream
	ICS1 *syntax.ICStream

	// ICS2 is the second channel's individual channel stream
	ICS2 *syntax.ICStream

	// Element is the syntax element (CPE)
	Element *syntax.Element

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16

	// ObjectType is the AAC object type
	ObjectType aac.ObjectType

	// SRIndex is the sample rate index (0-15)
	SRIndex uint8

	// PredState1 is the predictor state for MAIN profile, channel 1 (nil if not MAIN)
	PredState1 []PredState

	// PredState2 is the predictor state for MAIN profile, channel 2 (nil if not MAIN)
	PredState2 []PredState

	// LTPState1 is the LTP state buffer for LTP profile, channel 1 (nil if not LTP)
	LTPState1 []int16

	// LTPState2 is the LTP state buffer for LTP profile, channel 2 (nil if not LTP)
	LTPState2 []int16

	// LTPFilterBank is the forward MDCT for LTP (nil if not LTP)
	LTPFilterBank ForwardMDCT

	// WindowShape1 is the current window shape for channel 1
	WindowShape1 uint8

	// WindowShapePrev1 is the previous window shape for channel 1
	WindowShapePrev1 uint8

	// WindowShape2 is the current window shape for channel 2
	WindowShape2 uint8

	// WindowShapePrev2 is the previous window shape for channel 2
	WindowShapePrev2 uint8

	// PNSState is the PNS random number generator state
	PNSState *PNSState
}

// ReconstructChannelPair performs spectral reconstruction for a channel pair (stereo).
// This is the main entry point for converting parsed syntax data and quantized
// spectral coefficients into dequantized, processed spectral data ready for
// the filter bank.
//
// Processing order:
// 1. Inverse quantization + scale factors for both channels
// 2. PNS decode (with correlation based on ms_mask_present)
// 3. M/S stereo decode
// 4. Intensity stereo decode
// 5. IC Prediction (MAIN profile, both channels)
// 6. PNS reset pred state (MAIN profile, both channels)
// 7. LTP prediction (LTP profile, both channels, using ltp2 for channel 2 when common_window)
// 8. TNS decode for both channels
//
// Ported from: reconstruct_channel_pair() in ~/dev/faad2/libfaad/specrec.c:1131-1365
func ReconstructChannelPair(quantData1, quantData2 []int16, specData1, specData2 []float64, cfg *ReconstructChannelPairConfig) error {
	ics1 := cfg.ICS1
	ics2 := cfg.ICS2
	ele := cfg.Element
	frameLen := cfg.FrameLength

	// 1a. Pulse decode channel 1 (long blocks only)
	if ics1.PulseDataPresent {
		if ics1.WindowSequence == syntax.EightShortSequence {
			return syntax.ErrPulseInShortBlock
		}
		if err := PulseDecode(ics1, quantData1, frameLen); err != nil {
			return err
		}
	}

	// 1b. Pulse decode channel 2 (long blocks only)
	if ics2.PulseDataPresent {
		if ics2.WindowSequence == syntax.EightShortSequence {
			return syntax.ErrPulseInShortBlock
		}
		if err := PulseDecode(ics2, quantData2, frameLen); err != nil {
			return err
		}
	}

	// 1c. Inverse quantization: spec[i] = sign(quant[i]) * |quant[i]|^(4/3)
	if err := InverseQuantize(quantData1, specData1); err != nil {
		return err
	}
	if err := InverseQuantize(quantData2, specData2); err != nil {
		return err
	}

	// 1d. Apply scale factors: spec[i] *= 2^((sf-100)/4)
	ApplyScaleFactors(specData1, &ApplyScaleFactorsConfig{
		ICS:         ics1,
		FrameLength: frameLen,
	})
	ApplyScaleFactors(specData2, &ApplyScaleFactorsConfig{
		ICS:         ics2,
		FrameLength: frameLen,
	})

	// 2. PNS decode (with correlation based on ms_mask_present)
	// FAAD2: pns_decode() in specrec.c:1169-1177
	if cfg.PNSState != nil {
		channelPair := ics1.MSMaskPresent > 0
		PNSDecode(specData1, specData2, cfg.PNSState, &PNSDecodeConfig{
			ICSL:        ics1,
			ICSR:        ics2,
			FrameLength: frameLen,
			ChannelPair: channelPair,
			ObjectType:  uint8(cfg.ObjectType),
		})
	}

	// 3. M/S stereo decode
	// FAAD2: ms_decode() in specrec.c:1180
	MSDecode(specData1, specData2, &MSDecodeConfig{
		ICSL:        ics1,
		ICSR:        ics2,
		FrameLength: frameLen,
	})

	// 4. Intensity stereo decode
	// FAAD2: is_decode() in specrec.c:1199
	ISDecode(specData1, specData2, &ISDecodeConfig{
		ICSL:        ics1,
		ICSR:        ics2,
		FrameLength: frameLen,
	})

	// 5 & 6. IC Prediction (MAIN profile only)
	// FAAD2: specrec.c:1219-1233
	if cfg.ObjectType == aac.ObjectTypeMain {
		if cfg.PredState1 != nil {
			// Convert float64 to float32 for IC prediction (channel 1)
			specData32_1 := make([]float32, len(specData1))
			for i, v := range specData1 {
				specData32_1[i] = float32(v)
			}
			ICPrediction(ics1, specData32_1, cfg.PredState1, frameLen, cfg.SRIndex)
			for i, v := range specData32_1 {
				specData1[i] = float64(v)
			}
			PNSResetPredState(ics1, cfg.PredState1)
		}
		if cfg.PredState2 != nil {
			// Convert float64 to float32 for IC prediction (channel 2)
			specData32_2 := make([]float32, len(specData2))
			for i, v := range specData2 {
				specData32_2[i] = float32(v)
			}
			ICPrediction(ics2, specData32_2, cfg.PredState2, frameLen, cfg.SRIndex)
			for i, v := range specData32_2 {
				specData2[i] = float64(v)
			}
			PNSResetPredState(ics2, cfg.PredState2)
		}
	}

	// 7. LTP prediction (LTP profile only)
	// FAAD2: specrec.c:1236-1266
	// Note: For channel 2, use ltp2 when common_window is set
	if IsLTPObjectType(cfg.ObjectType) {
		if cfg.LTPState1 != nil {
			LTPPredictionWithMDCT(specData1, cfg.LTPState1, cfg.LTPFilterBank, &LTPConfig{
				ICS:             ics1,
				LTP:             &ics1.LTP,
				SRIndex:         cfg.SRIndex,
				ObjectType:      cfg.ObjectType,
				FrameLength:     frameLen,
				WindowShape:     cfg.WindowShape1,
				WindowShapePrev: cfg.WindowShapePrev1,
			})
		}
		if cfg.LTPState2 != nil {
			// Use ltp2 when common_window is set, otherwise use ltp
			var ltp2 *syntax.LTPInfo
			if ele.CommonWindow {
				ltp2 = &ics2.LTP2
			} else {
				ltp2 = &ics2.LTP
			}
			LTPPredictionWithMDCT(specData2, cfg.LTPState2, cfg.LTPFilterBank, &LTPConfig{
				ICS:             ics2,
				LTP:             ltp2,
				SRIndex:         cfg.SRIndex,
				ObjectType:      cfg.ObjectType,
				FrameLength:     frameLen,
				WindowShape:     cfg.WindowShape2,
				WindowShapePrev: cfg.WindowShapePrev2,
			})
		}
	}

	// 8. TNS decode (temporal noise shaping)
	// FAAD2: tns_decode_frame() in specrec.c:1270-1273
	if ics1.TNSDataPresent {
		TNSDecodeFrame(specData1, &TNSDecodeConfig{
			ICS:         ics1,
			SRIndex:     cfg.SRIndex,
			ObjectType:  cfg.ObjectType,
			FrameLength: frameLen,
		})
	}
	if ics2.TNSDataPresent {
		TNSDecodeFrame(specData2, &TNSDecodeConfig{
			ICS:         ics2,
			SRIndex:     cfg.SRIndex,
			ObjectType:  cfg.ObjectType,
			FrameLength: frameLen,
		})
	}

	return nil
}

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
