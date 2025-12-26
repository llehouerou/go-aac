package huffman

import "testing"

// TestCodebookConstants verifies Huffman codebook constants match FAAD2.
// Source: ~/dev/faad2/libfaad/syntax.h:101-108
func TestCodebookConstants(t *testing.T) {
	tests := []struct {
		name  string
		value Codebook
		want  Codebook
	}{
		{"ZERO_HCB", ZeroHCB, 0},
		{"FIRST_PAIR_HCB", FirstPairHCB, 5},
		{"ESC_HCB", EscHCB, 11},
		{"NOISE_HCB", NoiseHCB, 13},
		{"INTENSITY_HCB2", IntensityHCB2, 14},
		{"INTENSITY_HCB", IntensityHCB, 15},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

// TestCodewordLengths verifies codeword length constants match FAAD2.
// Source: ~/dev/faad2/libfaad/syntax.h:104-105
func TestCodewordLengths(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{"QUAD_LEN", QuadLen, 4},
		{"PAIR_LEN", PairLen, 2},
	}

	for _, tt := range tests {
		if tt.value != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}
