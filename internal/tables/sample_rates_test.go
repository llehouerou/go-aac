package tables

import (
	"testing"

	"github.com/llehouerou/go-aac"
)

func TestGetSampleRate(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/common.c:59-71
	tests := []struct {
		index    uint8
		expected uint32
	}{
		{0, 96000},
		{1, 88200},
		{2, 64000},
		{3, 48000},
		{4, 44100},
		{5, 32000},
		{6, 24000},
		{7, 22050},
		{8, 16000},
		{9, 12000},
		{10, 11025},
		{11, 8000},
		{12, 0}, // Invalid index
		{15, 0}, // Invalid index
	}

	for _, tt := range tests {
		got := GetSampleRate(tt.index)
		if got != tt.expected {
			t.Errorf("GetSampleRate(%d) = %d, want %d", tt.index, got, tt.expected)
		}
	}
}

func TestGetSRIndex(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/common.c:41-56
	// Uses threshold-based matching, not exact lookup
	tests := []struct {
		sampleRate uint32
		expected   uint8
	}{
		{96000, 0},
		{92017, 0}, // Threshold for index 0
		{92016, 1}, // Just below threshold
		{88200, 1},
		{75132, 1}, // Threshold for index 1
		{75131, 2}, // Just below threshold
		{64000, 2},
		{55426, 2}, // Threshold for index 2
		{55425, 3}, // Just below threshold
		{48000, 3},
		{46009, 3}, // Threshold for index 3
		{46008, 4}, // Just below threshold
		{44100, 4},
		{37566, 4}, // Threshold for index 4
		{37565, 5}, // Just below threshold
		{32000, 5},
		{27713, 5}, // Threshold for index 5
		{27712, 6}, // Just below threshold
		{24000, 6},
		{23004, 6}, // Threshold for index 6
		{23003, 7}, // Just below threshold
		{22050, 7},
		{18783, 7}, // Threshold for index 7
		{18782, 8}, // Just below threshold
		{16000, 8},
		{13856, 8}, // Threshold for index 8
		{13855, 9}, // Just below threshold
		{12000, 9},
		{11502, 9},  // Threshold for index 9
		{11501, 10}, // Just below threshold
		{11025, 10},
		{9391, 10}, // Threshold for index 10
		{9390, 11}, // Just below threshold
		{8000, 11},
		{7350, 11}, // Below 8000, still returns 11
		{100, 11},  // Any very low rate returns 11
	}

	for _, tt := range tests {
		got := GetSRIndex(tt.sampleRate)
		if got != tt.expected {
			t.Errorf("GetSRIndex(%d) = %d, want %d", tt.sampleRate, got, tt.expected)
		}
	}
}

func TestSampleRatesArray(t *testing.T) {
	// Verify SampleRates array has exactly 12 entries matching FAAD2
	// Source: ~/dev/faad2/libfaad/common.c:61-65
	expected := [12]uint32{
		96000, 88200, 64000, 48000, 44100, 32000,
		24000, 22050, 16000, 12000, 11025, 8000,
	}

	if SampleRates != expected {
		t.Errorf("SampleRates = %v, want %v", SampleRates, expected)
	}
}

func TestMaxPredSFB(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/common.c:73-85
	expected := [12]uint8{33, 33, 38, 40, 40, 40, 41, 41, 37, 37, 37, 34}

	for i := uint8(0); i < 12; i++ {
		got := MaxPredSFB(i)
		if got != expected[i] {
			t.Errorf("MaxPredSFB(%d) = %d, want %d", i, got, expected[i])
		}
	}

	// Test invalid index
	if got := MaxPredSFB(12); got != 0 {
		t.Errorf("MaxPredSFB(12) = %d, want 0", got)
	}
}

func TestMaxTNSSFB(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/common.c:87-121
	// Table columns: [Main/LC long, Main/LC short, SSR long, SSR short]
	tests := []struct {
		srIndex    uint8
		objectType aac.ObjectType
		isShort    bool
		expected   uint8
	}{
		// 96000 Hz
		{0, aac.ObjectTypeLC, false, 31},
		{0, aac.ObjectTypeLC, true, 9},
		{0, aac.ObjectTypeSSR, false, 28},
		{0, aac.ObjectTypeSSR, true, 7},
		// 48000 Hz
		{3, aac.ObjectTypeLC, false, 40},
		{3, aac.ObjectTypeLC, true, 14},
		{3, aac.ObjectTypeSSR, false, 26},
		{3, aac.ObjectTypeSSR, true, 6},
		// 44100 Hz
		{4, aac.ObjectTypeLC, false, 42},
		{4, aac.ObjectTypeLC, true, 14},
		// 8000 Hz
		{11, aac.ObjectTypeLC, false, 39},
		{11, aac.ObjectTypeLC, true, 14},
		// Invalid index returns 0
		{16, aac.ObjectTypeLC, false, 0},
	}

	for _, tt := range tests {
		got := MaxTNSSFB(tt.srIndex, tt.objectType, tt.isShort)
		if got != tt.expected {
			t.Errorf("MaxTNSSFB(%d, %d, %v) = %d, want %d",
				tt.srIndex, tt.objectType, tt.isShort, got, tt.expected)
		}
	}
}

func TestCanDecodeOT(t *testing.T) {
	// Source: ~/dev/faad2/libfaad/common.c:124-172
	// Note: We support LC, MAIN, LTP. SSR is not supported.
	tests := []struct {
		objectType aac.ObjectType
		canDecode  bool
	}{
		{aac.ObjectTypeLC, true},
		{aac.ObjectTypeMain, true},
		{aac.ObjectTypeLTP, true},
		{aac.ObjectTypeSSR, false},   // Not supported
		{aac.ObjectTypeHEAAC, false}, // SBR handled separately
		{aac.ObjectTypeERLC, true},
		{aac.ObjectTypeERLTP, true},
		{aac.ObjectTypeLD, true},
		{aac.ObjectTypeDRMERLC, true},
		{100, false}, // Unknown type
	}

	for _, tt := range tests {
		got := CanDecodeOT(tt.objectType)
		if got != tt.canDecode {
			t.Errorf("CanDecodeOT(%d) = %v, want %v", tt.objectType, got, tt.canDecode)
		}
	}
}
