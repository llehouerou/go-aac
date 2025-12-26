package syntax

import "testing"

// TestSyntaxElementIDs verifies syntax element IDs match FAAD2.
// Source: ~/dev/faad2/libfaad/syntax.h:85-94
func TestSyntaxElementIDs(t *testing.T) {
	tests := []struct {
		name  string
		value ElementID
		want  ElementID
	}{
		{"ID_SCE", IDSCE, 0x0},
		{"ID_CPE", IDCPE, 0x1},
		{"ID_CCE", IDCCE, 0x2},
		{"ID_LFE", IDLFE, 0x3},
		{"ID_DSE", IDDSE, 0x4},
		{"ID_PCE", IDPCE, 0x5},
		{"ID_FIL", IDFIL, 0x6},
		{"ID_END", IDEND, 0x7},
		{"INVALID_ELEMENT_ID", InvalidElementID, 255},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

// TestWindowSequences verifies window sequence values match FAAD2.
// Source: ~/dev/faad2/libfaad/syntax.h:96-99
func TestWindowSequences(t *testing.T) {
	tests := []struct {
		name  string
		value WindowSequence
		want  WindowSequence
	}{
		{"ONLY_LONG_SEQUENCE", OnlyLongSequence, 0x0},
		{"LONG_START_SEQUENCE", LongStartSequence, 0x1},
		{"EIGHT_SHORT_SEQUENCE", EightShortSequence, 0x2},
		{"LONG_STOP_SEQUENCE", LongStopSequence, 0x3},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

// TestExtensionTypes verifies extension type values match FAAD2.
// Source: ~/dev/faad2/libfaad/syntax.h:79-83
func TestExtensionTypes(t *testing.T) {
	tests := []struct {
		name  string
		value ExtensionType
		want  ExtensionType
	}{
		{"EXT_FIL", ExtFil, 0},
		{"EXT_FILL_DATA", ExtFillData, 1},
		{"EXT_DATA_ELEMENT", ExtDataElement, 2},
		{"EXT_DYNAMIC_RANGE", ExtDynamicRange, 11},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

// TestBitLengths verifies bit length constants match FAAD2.
// Source: ~/dev/faad2/libfaad/syntax.h:74-77
func TestBitLengths(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{"LEN_SE_ID", LenSEID, 3},
		{"LEN_TAG", LenTag, 4},
		{"LEN_BYTE", LenByte, 8},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

// TestDRMChannelConfigs verifies DRM channel config values match FAAD2.
// Source: ~/dev/faad2/libfaad/syntax.h:62-67
func TestDRMChannelConfigs(t *testing.T) {
	tests := []struct {
		name  string
		value DRMChannelConfig
		want  DRMChannelConfig
	}{
		{"DRMCH_MONO", DRMCHMono, 1},
		{"DRMCH_STEREO", DRMCHStereo, 2},
		{"DRMCH_SBR_MONO", DRMCHSBRMono, 3},
		{"DRMCH_SBR_STEREO", DRMCHSBRStereo, 4},
		{"DRMCH_SBR_PS_STEREO", DRMCHSBRPSStereo, 5},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

// TestERObjectStart verifies ER object start value matches FAAD2.
// Source: ~/dev/faad2/libfaad/syntax.h:71
func TestERObjectStart(t *testing.T) {
	if ERObjectStart != 17 {
		t.Errorf("ERObjectStart = %d, want 17", ERObjectStart)
	}
}

// TestInvalidSBRElement verifies invalid SBR element value matches FAAD2.
// Source: ~/dev/faad2/libfaad/syntax.h:110
func TestInvalidSBRElement(t *testing.T) {
	if InvalidSBRElement != 255 {
		t.Errorf("InvalidSBRElement = %d, want 255", InvalidSBRElement)
	}
}
