package syntax

// PulseInfo contains pulse data for spectral coefficient modification.
// Up to 4 pulses can be added to the spectral data in long blocks only.
//
// Ported from: pulse_info in ~/dev/faad2/libfaad/structs.h:210-216
type PulseInfo struct {
	NumberPulse   uint8    // Number of pulses (0-4)
	PulseStartSFB uint8    // Starting scale factor band
	PulseOffset   [4]uint8 // Offset from start of SFB for each pulse
	PulseAmp      [4]uint8 // Amplitude of each pulse
}
