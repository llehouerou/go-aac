package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestPulseDecode_SinglePulse(t *testing.T) {
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffset[3] = 12

	ics.Pul = syntax.PulseInfo{
		NumberPulse:   0,
		PulseStartSFB: 2,
		PulseOffset:   [4]uint8{2, 0, 0, 0},
		PulseAmp:      [4]uint8{5, 0, 0, 0},
	}

	specData := make([]int16, 1024)
	specData[10] = 100

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if specData[10] != 105 {
		t.Errorf("specData[10]: got %d, want 105", specData[10])
	}
}

func TestPulseDecode_NegativeValue(t *testing.T) {
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	ics.SWBOffset[2] = 8

	ics.Pul = syntax.PulseInfo{
		NumberPulse:   0,
		PulseStartSFB: 2,
		PulseOffset:   [4]uint8{0, 0, 0, 0},
		PulseAmp:      [4]uint8{3, 0, 0, 0},
	}

	specData := make([]int16, 1024)
	specData[8] = -50

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if specData[8] != -53 {
		t.Errorf("specData[8]: got %d, want -53", specData[8])
	}
}

func TestPulseDecode_MultiplePulses(t *testing.T) {
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	ics.SWBOffset[0] = 0

	ics.Pul = syntax.PulseInfo{
		NumberPulse:   3,
		PulseStartSFB: 0,
		PulseOffset:   [4]uint8{5, 3, 2, 10},
		PulseAmp:      [4]uint8{1, 2, 3, 4},
	}

	specData := make([]int16, 1024)
	specData[5] = 10
	specData[8] = 20
	specData[10] = 30
	specData[20] = -40

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if specData[5] != 11 {
		t.Errorf("specData[5]: got %d, want 11", specData[5])
	}
	if specData[8] != 22 {
		t.Errorf("specData[8]: got %d, want 22", specData[8])
	}
	if specData[10] != 33 {
		t.Errorf("specData[10]: got %d, want 33", specData[10])
	}
	if specData[20] != -44 {
		t.Errorf("specData[20]: got %d, want -44", specData[20])
	}
}

func TestPulseDecode_ZeroValue(t *testing.T) {
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	ics.SWBOffset[0] = 0

	ics.Pul = syntax.PulseInfo{
		NumberPulse:   0,
		PulseStartSFB: 0,
		PulseOffset:   [4]uint8{0, 0, 0, 0},
		PulseAmp:      [4]uint8{7, 0, 0, 0},
	}

	specData := make([]int16, 1024)
	specData[0] = 0

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Zero is not > 0, so else branch subtracts
	if specData[0] != -7 {
		t.Errorf("specData[0]: got %d, want -7", specData[0])
	}
}

func TestPulseDecode_PositionExceedsFrame(t *testing.T) {
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 1024,
	}
	ics.SWBOffset[0] = 1020

	ics.Pul = syntax.PulseInfo{
		NumberPulse:   0,
		PulseStartSFB: 0,
		PulseOffset:   [4]uint8{10, 0, 0, 0},
		PulseAmp:      [4]uint8{1, 0, 0, 0},
	}

	specData := make([]int16, 1024)

	err := PulseDecode(ics, specData, 1024)
	if err != syntax.ErrPulsePosition {
		t.Errorf("expected ErrPulsePosition, got %v", err)
	}
}

func TestPulseDecode_SWBOffsetMaxClamp(t *testing.T) {
	ics := &syntax.ICStream{
		NumSWB:       10,
		SWBOffsetMax: 100,
	}
	ics.SWBOffset[5] = 200

	ics.Pul = syntax.PulseInfo{
		NumberPulse:   0,
		PulseStartSFB: 5,
		PulseOffset:   [4]uint8{10, 0, 0, 0},
		PulseAmp:      [4]uint8{1, 0, 0, 0},
	}

	specData := make([]int16, 1024)
	specData[110] = 50

	err := PulseDecode(ics, specData, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if specData[110] != 51 {
		t.Errorf("specData[110]: got %d, want 51", specData[110])
	}
}
