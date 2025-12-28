// internal/syntax/ics_info_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseICSInfo_LongWindow(t *testing.T) {
	// Build bitstream:
	// ics_reserved_bit: 0 (1 bit)
	// window_sequence: 0 (2 bits) = ONLY_LONG_SEQUENCE
	// window_shape: 1 (1 bit) = KBD
	// max_sfb: 49 (6 bits) = 0b110001
	// Predictor data present: 0 (1 bit)
	// Total: 1 + 2 + 1 + 6 + 1 = 11 bits
	// Bits: 0 00 1 110001 0 = 0b0001_1100_01_0 padded = 0x1C40
	data := []byte{0x1C, 0x40}
	r := bits.NewReader(data)

	ics := &ICStream{}
	cfg := &ICSInfoConfig{
		SFIndex:      4, // 44100 Hz
		FrameLength:  1024,
		ObjectType:   2, // LC
		CommonWindow: false,
	}

	err := ParseICSInfo(r, ics, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.WindowSequence != OnlyLongSequence {
		t.Errorf("WindowSequence: got %d, want %d", ics.WindowSequence, OnlyLongSequence)
	}
	if ics.WindowShape != 1 {
		t.Errorf("WindowShape: got %d, want 1", ics.WindowShape)
	}
	if ics.MaxSFB != 49 {
		t.Errorf("MaxSFB: got %d, want 49", ics.MaxSFB)
	}
	if ics.PredictorDataPresent {
		t.Error("PredictorDataPresent should be false")
	}
}

func TestParseICSInfo_ShortWindow(t *testing.T) {
	// Build bitstream:
	// ics_reserved_bit: 0 (1 bit)
	// window_sequence: 10 (2 bits) = EIGHT_SHORT_SEQUENCE
	// window_shape: 0 (1 bit) = sine
	// max_sfb: 14 (4 bits) = 0b1110
	// scale_factor_grouping: 1111111 (7 bits) = all same group
	// Total: 1 + 2 + 1 + 4 + 7 = 15 bits
	// Bits: 0 10 0 1110 1111111 = 0b0100_1110_1111_111 padded
	data := []byte{0x4E, 0xFE} // 0b01001110 0b11111110
	r := bits.NewReader(data)

	ics := &ICStream{}
	cfg := &ICSInfoConfig{
		SFIndex:     4,
		FrameLength: 1024,
		ObjectType:  ObjectTypeLC,
	}

	err := ParseICSInfo(r, ics, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.WindowSequence != EightShortSequence {
		t.Errorf("WindowSequence: got %d, want %d", ics.WindowSequence, EightShortSequence)
	}
	if ics.MaxSFB != 14 {
		t.Errorf("MaxSFB: got %d, want 14", ics.MaxSFB)
	}
	if ics.ScaleFactorGrouping != 0x7F {
		t.Errorf("ScaleFactorGrouping: got 0x%02X, want 0x7F", ics.ScaleFactorGrouping)
	}
	// All bits set = 1 group with 8 windows
	if ics.NumWindowGroups != 1 {
		t.Errorf("NumWindowGroups: got %d, want 1", ics.NumWindowGroups)
	}
}

func TestParseICSInfo_ReservedBitError(t *testing.T) {
	// ics_reserved_bit: 1 (should error)
	// Bits: 1 xx xxx...
	data := []byte{0x80, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{}
	cfg := &ICSInfoConfig{
		SFIndex:     4,
		FrameLength: 1024,
		ObjectType:  ObjectTypeLC,
	}

	err := ParseICSInfo(r, ics, cfg)
	if err != ErrICSReservedBit {
		t.Errorf("expected ErrICSReservedBit, got %v", err)
	}
}

func TestParseICSInfo_LongWindowWithWindowGroups(t *testing.T) {
	// For long windows, NumWindowGroups should be 1 and NumWindows should be 1
	// ics_reserved_bit: 0 (1 bit)
	// window_sequence: 01 (2 bits) = LONG_START_SEQUENCE
	// window_shape: 0 (1 bit) = sine
	// max_sfb: 40 (6 bits) = 0b101000
	// Predictor data present: 0 (1 bit)
	// Total: 1 + 2 + 1 + 6 + 1 = 11 bits
	// Bits: 0 01 0 101000 0 = 0b0010_1010_000
	data := []byte{0x2A, 0x00} // 0b00101010 0b00000000
	r := bits.NewReader(data)

	ics := &ICStream{}
	cfg := &ICSInfoConfig{
		SFIndex:     4, // 44100 Hz
		FrameLength: 1024,
		ObjectType:  ObjectTypeLC,
	}

	err := ParseICSInfo(r, ics, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.WindowSequence != LongStartSequence {
		t.Errorf("WindowSequence: got %d, want %d", ics.WindowSequence, LongStartSequence)
	}
	if ics.NumWindowGroups != 1 {
		t.Errorf("NumWindowGroups: got %d, want 1", ics.NumWindowGroups)
	}
	if ics.NumWindows != 1 {
		t.Errorf("NumWindows: got %d, want 1", ics.NumWindows)
	}
}

func TestParseICSInfo_ShortWindowMultipleGroups(t *testing.T) {
	// Test with scale_factor_grouping that creates multiple groups
	// ics_reserved_bit: 0 (1 bit)
	// window_sequence: 10 (2 bits) = EIGHT_SHORT_SEQUENCE
	// window_shape: 1 (1 bit) = KBD
	// max_sfb: 12 (4 bits) = 0b1100
	// scale_factor_grouping: 0101010 (7 bits) - alternating new groups
	// Total: 1 + 2 + 1 + 4 + 7 = 15 bits
	// Bits: 0 10 1 1100 0101010 = 0b0101_1100_0101_010 padded
	// = 0x5C54 with one bit padding (0x5C, 0x54)
	data := []byte{0x5C, 0x54}
	r := bits.NewReader(data)

	ics := &ICStream{}
	cfg := &ICSInfoConfig{
		SFIndex:     4,
		FrameLength: 1024,
		ObjectType:  ObjectTypeLC,
	}

	err := ParseICSInfo(r, ics, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.WindowSequence != EightShortSequence {
		t.Errorf("WindowSequence: got %d, want %d", ics.WindowSequence, EightShortSequence)
	}
	if ics.WindowShape != 1 {
		t.Errorf("WindowShape: got %d, want 1", ics.WindowShape)
	}
	if ics.MaxSFB != 12 {
		t.Errorf("MaxSFB: got %d, want 12", ics.MaxSFB)
	}
	// scale_factor_grouping = 0101010 (bits 6..0)
	// Bit 6 = 0 -> new group at window 1 (groups: [0], [1...])
	// Bit 5 = 1 -> same group (groups: [0], [1,2...])
	// Bit 4 = 0 -> new group at window 3 (groups: [0], [1,2], [3...])
	// Bit 3 = 1 -> same group (groups: [0], [1,2], [3,4...])
	// Bit 2 = 0 -> new group at window 5 (groups: [0], [1,2], [3,4], [5...])
	// Bit 1 = 1 -> same group (groups: [0], [1,2], [3,4], [5,6...])
	// Bit 0 = 0 -> new group at window 7 (groups: [0], [1,2], [3,4], [5,6], [7])
	// Total: 5 groups with lengths [1, 2, 2, 2, 1]
	if ics.NumWindowGroups != 5 {
		t.Errorf("NumWindowGroups: got %d, want 5", ics.NumWindowGroups)
	}
	expectedLengths := [MaxWindowGroups]uint8{1, 2, 2, 2, 1}
	for i := uint8(0); i < ics.NumWindowGroups; i++ {
		if ics.WindowGroupLength[i] != expectedLengths[i] {
			t.Errorf("WindowGroupLength[%d]: got %d, want %d", i, ics.WindowGroupLength[i], expectedLengths[i])
		}
	}
}

func TestParseICSInfo_MainPrediction(t *testing.T) {
	// Test MAIN profile prediction data parsing
	// ics_reserved_bit: 0 (1 bit)
	// window_sequence: 0 (2 bits) = ONLY_LONG_SEQUENCE
	// window_shape: 0 (1 bit) = sine
	// max_sfb: 20 (6 bits) = 0b010100
	// Predictor data present: 1 (1 bit)
	// predictor_reset: 1 (1 bit)
	// predictor_reset_group_number: 5 (5 bits) = 0b00101
	// prediction_used[0..19]: 20 bits (all zeros)
	// Total: 1+2+1+6+1+1+5+20 = 37 bits
	// Bits: 0 00 0 010100 1 1 00101 00000000000000000000
	// Byte 0: 0_00_0_0101 = 0b00000101 = 0x05
	// Byte 1: 00_1_1_0010 = 0b00110010 = 0x32
	// Byte 2: 1_0000000 = 0b10000000 = 0x80
	// Byte 3: 0_0000000 = 0b00000000 = 0x00
	// Byte 4: 000_00000 = 0b00000000 = 0x00
	data := []byte{0x05, 0x32, 0x80, 0x00, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{}
	cfg := &ICSInfoConfig{
		SFIndex:     4,
		FrameLength: 1024,
		ObjectType:  ObjectTypeMain,
	}

	err := ParseICSInfo(r, ics, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.WindowSequence != OnlyLongSequence {
		t.Errorf("WindowSequence: got %d, want %d", ics.WindowSequence, OnlyLongSequence)
	}
	if ics.WindowShape != 0 {
		t.Errorf("WindowShape: got %d, want 0", ics.WindowShape)
	}
	if ics.MaxSFB != 20 {
		t.Errorf("MaxSFB: got %d, want 20", ics.MaxSFB)
	}
	if !ics.PredictorDataPresent {
		t.Error("PredictorDataPresent should be true")
	}
}

func TestParseICSInfo_LTPPrediction(t *testing.T) {
	// Test LTP prediction data parsing
	// ics_reserved_bit: 0 (1 bit)
	// window_sequence: 0 (2 bits) = ONLY_LONG_SEQUENCE
	// window_shape: 0 (1 bit) = sine
	// max_sfb: 10 (6 bits) = 0b001010
	// Predictor data present: 1 (1 bit)
	// ltp.data_present: 1 (1 bit)
	// ltp_lag: 512 (11 bits) = 0b01000000000
	// ltp_coef: 3 (3 bits) = 0b011
	// ltp_long_used[0..9]: 10 bits (all zeros for simplicity)
	// Total: 1+2+1+6+1+1+11+3+10 = 36 bits
	// Bits: 0 00 0 001010 1 1 01000000000 011 0000000000
	// Byte 0: 0_00_0_0010 = 0b00000010 = 0x02
	// Byte 1: 10_1_1_0100 = 0b10110100 = 0xB4
	// Byte 2: 0000000_0 = 0b00000000 = 0x00
	// Byte 3: 11_000000 = 0b11000000 = 0xC0
	// Byte 4: 0000_0000 = 0b00000000 = 0x00
	data := []byte{0x02, 0xB4, 0x00, 0xC0, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{}
	cfg := &ICSInfoConfig{
		SFIndex:     4,
		FrameLength: 1024,
		ObjectType:  ObjectTypeLTP, // LTP profile
	}

	err := ParseICSInfo(r, ics, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.WindowSequence != OnlyLongSequence {
		t.Errorf("WindowSequence: got %d, want %d", ics.WindowSequence, OnlyLongSequence)
	}
	if ics.MaxSFB != 10 {
		t.Errorf("MaxSFB: got %d, want 10", ics.MaxSFB)
	}
	if !ics.PredictorDataPresent {
		t.Error("PredictorDataPresent should be true")
	}
	if !ics.LTP.DataPresent {
		t.Error("LTP.DataPresent should be true")
	}
	if ics.LTP.Lag != 512 {
		t.Errorf("LTP.Lag: got %d, want 512", ics.LTP.Lag)
	}
	if ics.LTP.Coef != 3 {
		t.Errorf("LTP.Coef: got %d, want 3", ics.LTP.Coef)
	}
}

func TestParseLTPData_LagTooLarge(t *testing.T) {
	// Test LTP lag validation for a short frame (frameLength=480)
	// Max valid lag = 480 * 2 = 960
	// ltp_lag: 961 (11 bits) = 0b01111000001
	// ltp_coef: 0 (3 bits)
	// Bits: 01111000001 000 = 0b0111_1000_001_000_00
	// Byte 0: 0111_1000 = 0x78
	// Byte 1: 001_000_00 = 0x20
	data := []byte{0x78, 0x20}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		MaxSFB:         10,
	}
	ltp := &LTPInfo{}

	err := ParseLTPData(r, ics, ltp, 480)
	if err != ErrLTPLagTooLarge {
		t.Errorf("expected ErrLTPLagTooLarge, got %v (lag=%d)", err, ltp.Lag)
	}
}

func TestParseLTPData_ValidLag(t *testing.T) {
	// Test LTP with valid lag (960 for frameLength=480, limit is 960)
	// ltp_lag: 960 (11 bits) = 0b01111000000
	// ltp_coef: 5 (3 bits) = 0b101
	// ltp_long_used[0..9]: 10 bits (all zeros)
	// Bits: 01111000000 101 0000000000 = 0b0111_1000_000_101_00_00000000
	// Byte 0: 0111_1000 = 0x78
	// Byte 1: 000_101_00 = 0x14
	// Byte 2: 0000_0000 = 0x00
	data := []byte{0x78, 0x14, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		MaxSFB:         10,
	}
	ltp := &LTPInfo{}

	err := ParseLTPData(r, ics, ltp, 480)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ltp.Lag != 960 {
		t.Errorf("Lag: got %d, want 960", ltp.Lag)
	}
	if ltp.Coef != 5 {
		t.Errorf("Coef: got %d, want 5", ltp.Coef)
	}
	if ltp.LastBand != 10 {
		t.Errorf("LastBand: got %d, want 10", ltp.LastBand)
	}
}
