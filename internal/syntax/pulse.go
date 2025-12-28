package syntax

import "github.com/llehouerou/go-aac/internal/bits"

// PulseInfo contains pulse data for spectral coefficient modification.
// Up to 4 pulses can be added to the spectral data in long blocks only.
//
// Ported from: pulse_info in ~/dev/faad2/libfaad/structs.h:210-216
type PulseInfo struct {
	NumberPulse   uint8    // Number of pulses - 1 (0-3 = 1-4 pulses)
	PulseStartSFB uint8    // Starting scale factor band
	PulseOffset   [4]uint8 // Offset from start of SFB for each pulse
	PulseAmp      [4]uint8 // Amplitude of each pulse
}

// ParsePulseData parses pulse data from the bitstream.
// Pulse data modifies spectral coefficients at specific positions,
// used for encoding transients/attacks efficiently.
//
// Ported from: pulse_data() in ~/dev/faad2/libfaad/syntax.c:955-983
func ParsePulseData(r *bits.Reader, ics *ICStream, pul *PulseInfo) error {
	// number_pulse (2 bits) - actual count is number_pulse + 1
	pul.NumberPulse = uint8(r.GetBits(2))

	// pulse_start_sfb (6 bits)
	pul.PulseStartSFB = uint8(r.GetBits(6))

	// Validate start SFB - must not exceed num_swb
	// FAAD2 uses > (not >=), so pulse_start_sfb == num_swb is valid
	if pul.PulseStartSFB > ics.NumSWB {
		return ErrPulseStartSFB
	}

	// Read offset and amplitude for each pulse
	numPulses := pul.NumberPulse + 1
	for i := uint8(0); i < numPulses; i++ {
		pul.PulseOffset[i] = uint8(r.GetBits(5))
		pul.PulseAmp[i] = uint8(r.GetBits(4))
	}

	return nil
}
