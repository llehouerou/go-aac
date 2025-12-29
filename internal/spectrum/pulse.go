package spectrum

import "github.com/llehouerou/go-aac/internal/syntax"

// PulseDecode applies pulse data to spectral coefficients.
// Pulses add or subtract amplitude values at specific positions,
// used to efficiently encode transients and attacks.
//
// The function modifies specData in place. It should only be called
// for long blocks (pulse coding is not allowed in short blocks).
//
// Ported from: pulse_decode() in ~/dev/faad2/libfaad/pulse.c:36-58
func PulseDecode(ics *syntax.ICStream, specData []int16, frameLen uint16) error {
	pul := &ics.Pul

	// Start position is clamped to swb_offset_max
	k := ics.SWBOffset[pul.PulseStartSFB]
	if k > ics.SWBOffsetMax {
		k = ics.SWBOffsetMax
	}

	// Apply each pulse
	numPulses := pul.NumberPulse + 1
	for i := uint8(0); i < numPulses; i++ {
		k += uint16(pul.PulseOffset[i])

		if k >= frameLen {
			return syntax.ErrPulsePosition
		}

		if specData[k] > 0 {
			specData[k] += int16(pul.PulseAmp[i])
		} else {
			specData[k] -= int16(pul.PulseAmp[i])
		}
	}

	return nil
}
