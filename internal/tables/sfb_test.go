package tables

import "testing"

func TestGetSWBOffset(t *testing.T) {
	tests := []struct {
		srIndex     uint8
		frameLength uint16
		isShort     bool
		wantLen     int
	}{
		{3, 1024, false, 50},  // 48kHz long
		{3, 1024, true, 15},   // 48kHz short
		{0, 1024, false, 42},  // 96kHz long
		{0, 1024, true, 13},   // 96kHz short
		{11, 1024, false, 41}, // 8kHz long
		{11, 1024, true, 16},  // 8kHz short
	}

	for _, tt := range tests {
		offsets, err := GetSWBOffset(tt.srIndex, tt.frameLength, tt.isShort)
		if err != nil {
			t.Errorf("GetSWBOffset(%d, %d, %v) error: %v", tt.srIndex, tt.frameLength, tt.isShort, err)
			continue
		}
		if len(offsets) != tt.wantLen {
			t.Errorf("GetSWBOffset(%d, %d, %v) len = %d, want %d",
				tt.srIndex, tt.frameLength, tt.isShort, len(offsets), tt.wantLen)
		}
	}
}

func TestGetSWBOffsetInvalidIndex(t *testing.T) {
	_, err := GetSWBOffset(12, 1024, false)
	if err == nil {
		t.Error("GetSWBOffset(12, 1024, false) should return error")
	}
	if err != ErrInvalidSRIndex {
		t.Errorf("GetSWBOffset(12, 1024, false) error = %v, want ErrInvalidSRIndex", err)
	}
}

func TestGetNumSWB(t *testing.T) {
	tests := []struct {
		srIndex     uint8
		frameLength uint16
		isShort     bool
		want        uint8
	}{
		{3, 1024, false, 49}, // 48kHz long 1024
		{3, 1024, true, 14},  // 48kHz short
		{0, 1024, false, 41}, // 96kHz long 1024
		{4, 960, false, 49},  // 44.1kHz 960-sample
		{0, 960, false, 40},  // 96kHz 960-sample
	}

	for _, tt := range tests {
		got, err := GetNumSWB(tt.srIndex, tt.frameLength, tt.isShort)
		if err != nil {
			t.Errorf("GetNumSWB(%d, %d, %v) error: %v", tt.srIndex, tt.frameLength, tt.isShort, err)
			continue
		}
		if got != tt.want {
			t.Errorf("GetNumSWB(%d, %d, %v) = %d, want %d",
				tt.srIndex, tt.frameLength, tt.isShort, got, tt.want)
		}
	}
}

func TestGetNumSWBInvalidIndex(t *testing.T) {
	_, err := GetNumSWB(12, 1024, false)
	if err == nil {
		t.Error("GetNumSWB(12, 1024, false) should return error")
	}
	if err != ErrInvalidSRIndex {
		t.Errorf("GetNumSWB(12, 1024, false) error = %v, want ErrInvalidSRIndex", err)
	}
}
