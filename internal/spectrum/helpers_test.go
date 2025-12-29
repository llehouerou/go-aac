package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/huffman"
	"github.com/llehouerou/go-aac/internal/syntax"
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

func TestIsIntensityICS(t *testing.T) {
	ics := &syntax.ICStream{}

	// Test normal codebook
	ics.SFBCB[0][0] = uint8(huffman.EscHCB)
	if got := IsIntensityICS(ics, 0, 0); got != 0 {
		t.Errorf("IsIntensityICS with EscHCB = %d, want 0", got)
	}

	// Test intensity HCB (in-phase)
	ics.SFBCB[1][2] = uint8(huffman.IntensityHCB)
	if got := IsIntensityICS(ics, 1, 2); got != 1 {
		t.Errorf("IsIntensityICS with IntensityHCB = %d, want 1", got)
	}

	// Test intensity HCB2 (out-of-phase)
	ics.SFBCB[2][3] = uint8(huffman.IntensityHCB2)
	if got := IsIntensityICS(ics, 2, 3); got != -1 {
		t.Errorf("IsIntensityICS with IntensityHCB2 = %d, want -1", got)
	}
}

func TestIsNoiseICS(t *testing.T) {
	ics := &syntax.ICStream{}

	// Test normal codebook
	ics.SFBCB[0][0] = uint8(huffman.EscHCB)
	if IsNoiseICS(ics, 0, 0) {
		t.Error("IsNoiseICS with EscHCB = true, want false")
	}

	// Test noise codebook
	ics.SFBCB[1][2] = uint8(huffman.NoiseHCB)
	if !IsNoiseICS(ics, 1, 2) {
		t.Error("IsNoiseICS with NoiseHCB = false, want true")
	}
}
