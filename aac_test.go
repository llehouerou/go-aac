package aac

import "testing"

// TestObjectTypeConstants verifies object type values match FAAD2.
// Source: ~/dev/faad2/include/neaacdec.h:74-83
func TestObjectTypeConstants(t *testing.T) {
	tests := []struct {
		name  string
		value ObjectType
		want  ObjectType
	}{
		{"MAIN", ObjectTypeMain, 1},
		{"LC", ObjectTypeLC, 2},
		{"SSR", ObjectTypeSSR, 3},
		{"LTP", ObjectTypeLTP, 4},
		{"HE_AAC", ObjectTypeHEAAC, 5},
		{"ER_LC", ObjectTypeERLC, 17},
		{"ER_LTP", ObjectTypeERLTP, 19},
		{"LD", ObjectTypeLD, 23},
		{"DRM_ER_LC", ObjectTypeDRMERLC, 27},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

// TestHeaderTypeConstants verifies header type values match FAAD2.
// Source: ~/dev/faad2/include/neaacdec.h:85-89
func TestHeaderTypeConstants(t *testing.T) {
	tests := []struct {
		name  string
		value HeaderType
		want  HeaderType
	}{
		{"RAW", HeaderTypeRAW, 0},
		{"ADIF", HeaderTypeADIF, 1},
		{"ADTS", HeaderTypeADTS, 2},
		{"LATM", HeaderTypeLATM, 3},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

// TestOutputFormatConstants verifies output format values match FAAD2.
// Source: ~/dev/faad2/include/neaacdec.h:97-103
func TestOutputFormatConstants(t *testing.T) {
	tests := []struct {
		name  string
		value OutputFormat
		want  OutputFormat
	}{
		{"16BIT", OutputFormat16Bit, 1},
		{"24BIT", OutputFormat24Bit, 2},
		{"32BIT", OutputFormat32Bit, 3},
		{"FLOAT", OutputFormatFloat, 4},
		{"DOUBLE", OutputFormatDouble, 5},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

// TestChannelPositionConstants verifies channel position values match FAAD2.
// Source: ~/dev/faad2/include/neaacdec.h:113-123
func TestChannelPositionConstants(t *testing.T) {
	tests := []struct {
		name  string
		value ChannelPosition
		want  ChannelPosition
	}{
		{"UNKNOWN", ChannelUnknown, 0},
		{"FRONT_CENTER", ChannelFrontCenter, 1},
		{"FRONT_LEFT", ChannelFrontLeft, 2},
		{"FRONT_RIGHT", ChannelFrontRight, 3},
		{"SIDE_LEFT", ChannelSideLeft, 4},
		{"SIDE_RIGHT", ChannelSideRight, 5},
		{"BACK_LEFT", ChannelBackLeft, 6},
		{"BACK_RIGHT", ChannelBackRight, 7},
		{"BACK_CENTER", ChannelBackCenter, 8},
		{"LFE", ChannelLFE, 9},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

// TestSBRSignallingConstants verifies SBR signalling values match FAAD2.
// Source: ~/dev/faad2/include/neaacdec.h:91-95
func TestSBRSignallingConstants(t *testing.T) {
	tests := []struct {
		name  string
		value SBRSignalling
		want  SBRSignalling
	}{
		{"NO_SBR", SBRNone, 0},
		{"SBR_UPSAMPLED", SBRUpsampled, 1},
		{"SBR_DOWNSAMPLED", SBRDownsampled, 2},
		{"NO_SBR_UPSAMPLED", SBRNoneUpsampled, 3},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

// TestMinStreamSize verifies minimum stream size matches FAAD2.
// Source: ~/dev/faad2/include/neaacdec.h:135
func TestMinStreamSize(t *testing.T) {
	if MinStreamSize != 768 {
		t.Errorf("MinStreamSize = %d, want 768", MinStreamSize)
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{}
	// Verify zero values are valid defaults
	if cfg.DefObjectType != 0 {
		t.Error("DefObjectType should default to 0")
	}
}

func TestFrameInfoFields(t *testing.T) {
	// Verify ChannelPosition array size matches FAAD2's MAX_CHANNELS
	var info FrameInfo
	got := len(info.ChannelPosition)
	if got != 64 {
		t.Errorf("ChannelPosition array size = %d, want 64", got)
	}
}

func TestVersion(t *testing.T) {
	version := Version()
	if version == "" {
		t.Error("Version() returned empty string")
	}
}

func TestGetCapabilities(t *testing.T) {
	caps := GetCapabilities()
	// Must support LC at minimum
	if caps&CapabilityLC == 0 {
		t.Error("GetCapabilities() must include LC capability")
	}
}

func TestCapabilityConstants(t *testing.T) {
	// Verify capability bits match FAAD2 neaacdec.h:106-111
	tests := []struct {
		name string
		got  Capability
		want Capability
	}{
		{"LC_DEC_CAP", CapabilityLC, 1 << 0},
		{"MAIN_DEC_CAP", CapabilityMain, 1 << 1},
		{"LTP_DEC_CAP", CapabilityLTP, 1 << 2},
		{"LD_DEC_CAP", CapabilityLD, 1 << 3},
		{"ERROR_RESILIENCE_CAP", CapabilityER, 1 << 4},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}
