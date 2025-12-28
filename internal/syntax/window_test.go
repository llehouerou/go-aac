// internal/syntax/window_test.go
package syntax

import "testing"

func TestWindowGroupingInfo_LongWindow(t *testing.T) {
	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		MaxSFB:         49,
	}

	err := WindowGroupingInfo(ics, 4, 1024) // sf_index=4 (44100 Hz), frameLength=1024
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumWindows != 1 {
		t.Errorf("NumWindows: got %d, want 1", ics.NumWindows)
	}
	if ics.NumWindowGroups != 1 {
		t.Errorf("NumWindowGroups: got %d, want 1", ics.NumWindowGroups)
	}
	if ics.WindowGroupLength[0] != 1 {
		t.Errorf("WindowGroupLength[0]: got %d, want 1", ics.WindowGroupLength[0])
	}
	// For 44100 Hz, num_swb should be 49
	if ics.NumSWB != 49 {
		t.Errorf("NumSWB: got %d, want 49", ics.NumSWB)
	}
}

func TestWindowGroupingInfo_LongStartSequence(t *testing.T) {
	ics := &ICStream{
		WindowSequence: LongStartSequence,
		MaxSFB:         41,
	}

	err := WindowGroupingInfo(ics, 0, 1024) // sf_index=0 (96000 Hz), frameLength=1024
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumWindows != 1 {
		t.Errorf("NumWindows: got %d, want 1", ics.NumWindows)
	}
	if ics.NumWindowGroups != 1 {
		t.Errorf("NumWindowGroups: got %d, want 1", ics.NumWindowGroups)
	}
	// For 96000 Hz, num_swb should be 41
	if ics.NumSWB != 41 {
		t.Errorf("NumSWB: got %d, want 41", ics.NumSWB)
	}
	// Verify SWBOffsetMax is set to frame length
	if ics.SWBOffsetMax != 1024 {
		t.Errorf("SWBOffsetMax: got %d, want 1024", ics.SWBOffsetMax)
	}
}

func TestWindowGroupingInfo_LongStopSequence(t *testing.T) {
	ics := &ICStream{
		WindowSequence: LongStopSequence,
		MaxSFB:         47,
	}

	err := WindowGroupingInfo(ics, 2, 1024) // sf_index=2 (64000 Hz), frameLength=1024
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumWindows != 1 {
		t.Errorf("NumWindows: got %d, want 1", ics.NumWindows)
	}
	// For 64000 Hz, num_swb should be 47
	if ics.NumSWB != 47 {
		t.Errorf("NumSWB: got %d, want 47", ics.NumSWB)
	}
}

func TestWindowGroupingInfo_ShortWindow(t *testing.T) {
	// 0b1011010 = 90 decimal
	// For i=0 to 6, check bit (6-i):
	// i=0: bit 6 = (90 >> 6) & 1 = 1 -> same group
	// i=1: bit 5 = (90 >> 5) & 1 = 0 -> new group
	// i=2: bit 4 = (90 >> 4) & 1 = 1 -> same group
	// i=3: bit 3 = (90 >> 3) & 1 = 1 -> same group
	// i=4: bit 2 = (90 >> 2) & 1 = 0 -> new group
	// i=5: bit 1 = (90 >> 1) & 1 = 1 -> same group
	// i=6: bit 0 = (90 >> 0) & 1 = 0 -> new group
	//
	// So groups are: [0,1], [2,3,4], [5,6], [7]
	// Lengths: 2, 3, 2, 1
	// NumWindowGroups = 4
	ics := &ICStream{
		WindowSequence:      EightShortSequence,
		MaxSFB:              14,
		ScaleFactorGrouping: 0b1011010, // Groups: [2, 3, 2, 1]
	}

	err := WindowGroupingInfo(ics, 4, 1024) // 44100 Hz
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumWindows != 8 {
		t.Errorf("NumWindows: got %d, want 8", ics.NumWindows)
	}

	if ics.NumWindowGroups != 4 {
		t.Errorf("NumWindowGroups: got %d, want 4", ics.NumWindowGroups)
	}

	expectedLengths := []uint8{2, 3, 2, 1}
	for i, want := range expectedLengths {
		if ics.WindowGroupLength[i] != want {
			t.Errorf("WindowGroupLength[%d]: got %d, want %d", i, ics.WindowGroupLength[i], want)
		}
	}

	// For 44100 Hz short windows, num_swb should be 14
	if ics.NumSWB != 14 {
		t.Errorf("NumSWB: got %d, want 14", ics.NumSWB)
	}

	// SWBOffsetMax should be frameLength/8 = 128
	if ics.SWBOffsetMax != 128 {
		t.Errorf("SWBOffsetMax: got %d, want 128", ics.SWBOffsetMax)
	}
}

func TestWindowGroupingInfo_ShortWindowAllSameGroup(t *testing.T) {
	// All 7 bits set = 0b1111111 = 127
	// All windows in same group
	ics := &ICStream{
		WindowSequence:      EightShortSequence,
		MaxSFB:              14,
		ScaleFactorGrouping: 0b1111111, // All windows in one group
	}

	err := WindowGroupingInfo(ics, 4, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumWindowGroups != 1 {
		t.Errorf("NumWindowGroups: got %d, want 1", ics.NumWindowGroups)
	}

	if ics.WindowGroupLength[0] != 8 {
		t.Errorf("WindowGroupLength[0]: got %d, want 8", ics.WindowGroupLength[0])
	}
}

func TestWindowGroupingInfo_ShortWindowEachSeparate(t *testing.T) {
	// All 7 bits clear = 0b0000000 = 0
	// Each window in its own group
	ics := &ICStream{
		WindowSequence:      EightShortSequence,
		MaxSFB:              14,
		ScaleFactorGrouping: 0b0000000, // Each window separate
	}

	err := WindowGroupingInfo(ics, 4, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ics.NumWindowGroups != 8 {
		t.Errorf("NumWindowGroups: got %d, want 8", ics.NumWindowGroups)
	}

	for i := 0; i < 8; i++ {
		if ics.WindowGroupLength[i] != 1 {
			t.Errorf("WindowGroupLength[%d]: got %d, want 1", i, ics.WindowGroupLength[i])
		}
	}
}

func TestWindowGroupingInfo_InvalidSRIndex(t *testing.T) {
	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		MaxSFB:         10,
	}

	err := WindowGroupingInfo(ics, 12, 1024) // Invalid: must be 0-11
	if err != ErrInvalidSRIndex {
		t.Errorf("expected ErrInvalidSRIndex, got %v", err)
	}

	err = WindowGroupingInfo(ics, 255, 1024) // Invalid: way out of range
	if err != ErrInvalidSRIndex {
		t.Errorf("expected ErrInvalidSRIndex, got %v", err)
	}
}

func TestWindowGroupingInfo_InvalidWindowSequence(t *testing.T) {
	ics := &ICStream{
		WindowSequence: WindowSequence(99), // Invalid window sequence
		MaxSFB:         10,
	}

	err := WindowGroupingInfo(ics, 4, 1024)
	if err != ErrInvalidWindowSequence {
		t.Errorf("expected ErrInvalidWindowSequence, got %v", err)
	}
}

func TestWindowGroupingInfo_MaxSFBTooLarge(t *testing.T) {
	// For long window at 44100 Hz, num_swb is 49
	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		MaxSFB:         50, // Too large: 50 > 49
	}

	err := WindowGroupingInfo(ics, 4, 1024)
	if err != ErrMaxSFBTooLarge {
		t.Errorf("expected ErrMaxSFBTooLarge, got %v", err)
	}
}

func TestWindowGroupingInfo_MaxSFBTooLargeShort(t *testing.T) {
	// For short window at 44100 Hz, num_swb is 14
	ics := &ICStream{
		WindowSequence: EightShortSequence,
		MaxSFB:         15, // Too large: 15 > 14
	}

	err := WindowGroupingInfo(ics, 4, 1024)
	if err != ErrMaxSFBTooLarge {
		t.Errorf("expected ErrMaxSFBTooLarge, got %v", err)
	}
}

func TestWindowGroupingInfo_SFBOffsetsLong(t *testing.T) {
	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		MaxSFB:         10,
	}

	err := WindowGroupingInfo(ics, 4, 1024) // 44100 Hz
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify first few SFB offsets match expected values for 44100 Hz
	// From tables: SWBOffset1024_48 = {0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, ...}
	expectedOffsets := []uint16{0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40}
	for i, want := range expectedOffsets {
		if ics.SWBOffset[i] != want {
			t.Errorf("SWBOffset[%d]: got %d, want %d", i, ics.SWBOffset[i], want)
		}
		if ics.SectSFBOffset[0][i] != want {
			t.Errorf("SectSFBOffset[0][%d]: got %d, want %d", i, ics.SectSFBOffset[0][i], want)
		}
	}

	// Last offset should be frame length
	if ics.SWBOffset[ics.NumSWB] != 1024 {
		t.Errorf("SWBOffset[%d]: got %d, want 1024", ics.NumSWB, ics.SWBOffset[ics.NumSWB])
	}
}

func TestWindowGroupingInfo_SFBOffsetsShort(t *testing.T) {
	ics := &ICStream{
		WindowSequence:      EightShortSequence,
		MaxSFB:              10,
		ScaleFactorGrouping: 0b0000000, // Each window separate
	}

	err := WindowGroupingInfo(ics, 4, 1024) // 44100 Hz
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify first few SFB offsets match expected values for 44100 Hz short
	// From tables: SWBOffset128_48 = {0, 4, 8, 12, 16, 20, 28, 36, 44, 56, 68, 80, 96, 112, 128}
	expectedOffsets := []uint16{0, 4, 8, 12, 16, 20, 28, 36, 44, 56}
	for i, want := range expectedOffsets {
		if ics.SWBOffset[i] != want {
			t.Errorf("SWBOffset[%d]: got %d, want %d", i, ics.SWBOffset[i], want)
		}
	}

	// Last offset should be frame length / 8 = 128
	if ics.SWBOffset[ics.NumSWB] != 128 {
		t.Errorf("SWBOffset[%d]: got %d, want 128", ics.NumSWB, ics.SWBOffset[ics.NumSWB])
	}
}

func TestWindowGroupingInfo_960FrameLength(t *testing.T) {
	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		MaxSFB:         40,
	}

	err := WindowGroupingInfo(ics, 0, 960) // 96000 Hz, 960 frame
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// For 96000 Hz, 960-sample frame, num_swb should be 40
	if ics.NumSWB != 40 {
		t.Errorf("NumSWB: got %d, want 40", ics.NumSWB)
	}

	// SWBOffsetMax should be 960
	if ics.SWBOffsetMax != 960 {
		t.Errorf("SWBOffsetMax: got %d, want 960", ics.SWBOffsetMax)
	}
}

func TestBitSet(t *testing.T) {
	testCases := []struct {
		a      uint8
		b      uint8
		expect bool
	}{
		{0b10000000, 7, true},
		{0b10000000, 6, false},
		{0b01000000, 6, true},
		{0b00000001, 0, true},
		{0b00000001, 1, false},
		{0b11111111, 0, true},
		{0b11111111, 7, true},
		{0b00000000, 0, false},
		{0b00000000, 7, false},
		{0b1011010, 6, true},  // bit 6 of 90
		{0b1011010, 5, false}, // bit 5 of 90
		{0b1011010, 4, true},  // bit 4 of 90
		{0b1011010, 3, true},  // bit 3 of 90
		{0b1011010, 2, false}, // bit 2 of 90
		{0b1011010, 1, true},  // bit 1 of 90
		{0b1011010, 0, false}, // bit 0 of 90
	}

	for _, tc := range testCases {
		result := bitSet(tc.a, tc.b)
		if result != tc.expect {
			t.Errorf("bitSet(%d, %d): got %v, want %v", tc.a, tc.b, result, tc.expect)
		}
	}
}
