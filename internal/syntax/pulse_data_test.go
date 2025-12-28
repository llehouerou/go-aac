// internal/syntax/pulse_data_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParsePulseData_TwoPulses(t *testing.T) {
	// number_pulse: 1 (2 bits) = 2 pulses (number_pulse + 1)
	// pulse_start_sfb: 5 (6 bits)
	// pulse_offset[0]: 10 (5 bits)
	// pulse_amp[0]: 7 (4 bits)
	// pulse_offset[1]: 15 (5 bits)
	// pulse_amp[1]: 3 (4 bits)
	// Total: 2 + 6 + 5 + 4 + 5 + 4 = 26 bits
	// Bits: 01 000101 01010 0111 01111 0011
	// Byte layout: 01000101 01010011 10111100 11xxxxxx
	data := []byte{0x45, 0x53, 0xBC, 0xC0}
	r := bits.NewReader(data)

	ics := &ICStream{
		NumSWB: 40, // Valid for the start SFB
	}
	pul := &PulseInfo{}

	err := ParsePulseData(r, ics, pul)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pul.NumberPulse != 1 { // 1 means 2 pulses
		t.Errorf("NumberPulse: got %d, want 1", pul.NumberPulse)
	}
	if pul.PulseStartSFB != 5 {
		t.Errorf("PulseStartSFB: got %d, want 5", pul.PulseStartSFB)
	}
	if pul.PulseOffset[0] != 10 {
		t.Errorf("PulseOffset[0]: got %d, want 10", pul.PulseOffset[0])
	}
	if pul.PulseAmp[0] != 7 {
		t.Errorf("PulseAmp[0]: got %d, want 7", pul.PulseAmp[0])
	}
	if pul.PulseOffset[1] != 15 {
		t.Errorf("PulseOffset[1]: got %d, want 15", pul.PulseOffset[1])
	}
	if pul.PulseAmp[1] != 3 {
		t.Errorf("PulseAmp[1]: got %d, want 3", pul.PulseAmp[1])
	}
}

func TestParsePulseData_OnePulse(t *testing.T) {
	// number_pulse: 0 (2 bits) = 1 pulse
	// pulse_start_sfb: 3 (6 bits)
	// pulse_offset[0]: 20 (5 bits)
	// pulse_amp[0]: 15 (4 bits)
	// Total: 2 + 6 + 5 + 4 = 17 bits
	// Bits: 00 000011 10100 1111 + padding
	// Binary: 00000011 10100111 1xxxxxxx
	data := []byte{0x03, 0xA7, 0x80}
	r := bits.NewReader(data)

	ics := &ICStream{
		NumSWB: 40,
	}
	pul := &PulseInfo{}

	err := ParsePulseData(r, ics, pul)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pul.NumberPulse != 0 {
		t.Errorf("NumberPulse: got %d, want 0", pul.NumberPulse)
	}
	if pul.PulseStartSFB != 3 {
		t.Errorf("PulseStartSFB: got %d, want 3", pul.PulseStartSFB)
	}
	if pul.PulseOffset[0] != 20 {
		t.Errorf("PulseOffset[0]: got %d, want 20", pul.PulseOffset[0])
	}
	if pul.PulseAmp[0] != 15 {
		t.Errorf("PulseAmp[0]: got %d, want 15", pul.PulseAmp[0])
	}
}

func TestParsePulseData_FourPulses(t *testing.T) {
	// number_pulse: 3 (2 bits) = 4 pulses (max)
	// pulse_start_sfb: 10 (6 bits)
	// pulse_offset[0]: 1 (5 bits), pulse_amp[0]: 2 (4 bits)
	// pulse_offset[1]: 3 (5 bits), pulse_amp[1]: 4 (4 bits)
	// pulse_offset[2]: 5 (5 bits), pulse_amp[2]: 6 (4 bits)
	// pulse_offset[3]: 7 (5 bits), pulse_amp[3]: 8 (4 bits)
	// Total: 2 + 6 + 4*(5+4) = 44 bits = 5.5 bytes
	// Bits: 11 001010 00001 0010 00011 0100 00101 0110 00111 1000
	// Binary layout:
	// 11001010 00001001 00001101 00001010 11000111 1000xxxx
	data := []byte{0xCA, 0x09, 0x0D, 0x0A, 0xC7, 0x80}
	r := bits.NewReader(data)

	ics := &ICStream{
		NumSWB: 40,
	}
	pul := &PulseInfo{}

	err := ParsePulseData(r, ics, pul)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pul.NumberPulse != 3 {
		t.Errorf("NumberPulse: got %d, want 3", pul.NumberPulse)
	}
	if pul.PulseStartSFB != 10 {
		t.Errorf("PulseStartSFB: got %d, want 10", pul.PulseStartSFB)
	}

	expectedOffsets := [4]uint8{1, 3, 5, 7}
	expectedAmps := [4]uint8{2, 4, 6, 8}

	for i := 0; i < 4; i++ {
		if pul.PulseOffset[i] != expectedOffsets[i] {
			t.Errorf("PulseOffset[%d]: got %d, want %d", i, pul.PulseOffset[i], expectedOffsets[i])
		}
		if pul.PulseAmp[i] != expectedAmps[i] {
			t.Errorf("PulseAmp[%d]: got %d, want %d", i, pul.PulseAmp[i], expectedAmps[i])
		}
	}
}

func TestParsePulseData_InvalidStartSFB(t *testing.T) {
	// number_pulse: 0 (2 bits)
	// pulse_start_sfb: 50 (6 bits) - exceeds NumSWB of 40
	// Bits: 00 110010 ...
	data := []byte{0x32, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		NumSWB: 40,
	}
	pul := &PulseInfo{}

	err := ParsePulseData(r, ics, pul)
	if err != ErrPulseStartSFB {
		t.Errorf("expected ErrPulseStartSFB, got %v", err)
	}
}

func TestParsePulseData_StartSFBAtBoundary(t *testing.T) {
	// Test edge case: pulse_start_sfb exactly equals num_swb
	// FAAD2 uses > (not >=), so this should be valid
	// number_pulse: 0 (2 bits)
	// pulse_start_sfb: 40 (6 bits) - equals NumSWB of 40, should be valid
	// pulse_offset[0]: 0 (5 bits)
	// pulse_amp[0]: 0 (4 bits)
	// Bits: 00 101000 00000 0000
	data := []byte{0x28, 0x00, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		NumSWB: 40,
	}
	pul := &PulseInfo{}

	err := ParsePulseData(r, ics, pul)
	if err != nil {
		t.Fatalf("unexpected error for boundary case: %v", err)
	}

	if pul.PulseStartSFB != 40 {
		t.Errorf("PulseStartSFB: got %d, want 40", pul.PulseStartSFB)
	}
}
