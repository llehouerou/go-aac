package syntax

import "testing"

// TestLimits verifies limit constants match FAAD2.
// Source: ~/dev/faad2/libfaad/structs.h:43-48
func TestLimits(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{"MAX_CHANNELS", MaxChannels, 64},
		{"MAX_SYNTAX_ELEMENTS", MaxSyntaxElements, 48},
		{"MAX_WINDOW_GROUPS", MaxWindowGroups, 8},
		{"MAX_SFB", MaxSFB, 51},
		{"MAX_LTP_SFB", MaxLTPSFB, 40},
		{"MAX_LTP_SFB_S", MaxLTPSFBS, 8},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}
