package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/huffman"
)

func TestIsIntensity(t *testing.T) {
	tests := []struct {
		cb       huffman.Codebook
		expected int8
	}{
		{huffman.ZeroHCB, 0},
		{huffman.Codebook(1), 0},
		{huffman.EscHCB, 0},
		{huffman.NoiseHCB, 0},
		{huffman.IntensityHCB, 1},
		{huffman.IntensityHCB2, -1},
	}

	for _, tc := range tests {
		got := IsIntensity(tc.cb)
		if got != tc.expected {
			t.Errorf("IsIntensity(%d) = %d, want %d", tc.cb, got, tc.expected)
		}
	}
}

func TestIsNoise(t *testing.T) {
	tests := []struct {
		cb       huffman.Codebook
		expected bool
	}{
		{huffman.ZeroHCB, false},
		{huffman.Codebook(1), false},
		{huffman.EscHCB, false},
		{huffman.NoiseHCB, true},
		{huffman.IntensityHCB, false},
		{huffman.IntensityHCB2, false},
	}

	for _, tc := range tests {
		got := IsNoise(tc.cb)
		if got != tc.expected {
			t.Errorf("IsNoise(%d) = %v, want %v", tc.cb, got, tc.expected)
		}
	}
}
