//go:build faad2_validation

// Package filterbank filterbank_faad2_test.go validates filter bank output against FAAD2 reference.
package filterbank

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/llehouerou/go-aac/internal/syntax"
)

// TestIFilterBank_FAAD2Reference validates filter bank output against FAAD2 reference.
// This test is skipped by default; run with -tags=faad2_validation
func TestIFilterBank_FAAD2Reference(t *testing.T) {
	refDir := "/tmp/faad2_filterbank_ref"
	if _, err := os.Stat(refDir); os.IsNotExist(err) {
		t.Skipf("FAAD2 reference data not found at %s; generate with scripts/check_faad2", refDir)
	}

	// Load test cases from reference directory
	entries, err := os.ReadDir(refDir)
	if err != nil {
		t.Fatalf("failed to read reference directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			testDir := filepath.Join(refDir, entry.Name())
			validateFilterBankFrame(t, testDir)
		})
	}
}

func validateFilterBankFrame(t *testing.T, testDir string) {
	t.Helper()

	// Load input spectral data
	freqInPath := filepath.Join(testDir, "freq_in.bin")
	freqInData, err := os.ReadFile(freqInPath)
	if err != nil {
		t.Skipf("skipping: %v", err)
		return
	}

	// Load expected output
	timeOutPath := filepath.Join(testDir, "time_out.bin")
	expectedData, err := os.ReadFile(timeOutPath)
	if err != nil {
		t.Skipf("skipping: %v", err)
		return
	}

	// Load window sequence info
	infoPath := filepath.Join(testDir, "info.bin")
	infoData, err := os.ReadFile(infoPath)
	if err != nil {
		t.Skipf("skipping: %v", err)
		return
	}

	if len(infoData) < 3 {
		t.Skipf("info.bin too short: got %d bytes, need at least 3", len(infoData))
		return
	}

	// Parse info (window_sequence, window_shape, window_shape_prev)
	windowSeq := syntax.WindowSequence(infoData[0])
	windowShape := infoData[1]
	windowShapePrev := infoData[2]

	// Convert binary to float32 slices
	frameLen := len(freqInData) / 4
	freqIn := make([]float32, frameLen)
	for i := 0; i < frameLen; i++ {
		bits := binary.LittleEndian.Uint32(freqInData[i*4:])
		freqIn[i] = math.Float32frombits(bits)
	}

	expected := make([]float32, len(expectedData)/4)
	for i := 0; i < len(expected); i++ {
		bits := binary.LittleEndian.Uint32(expectedData[i*4:])
		expected[i] = math.Float32frombits(bits)
	}

	// Run filter bank
	fb := NewFilterBank(uint16(frameLen))
	timeOut := make([]float32, frameLen)
	overlap := make([]float32, frameLen) // Assume fresh start

	fb.IFilterBank(windowSeq, windowShape, windowShapePrev, freqIn, timeOut, overlap)

	// Compare output
	const tolerance = 1e-5
	errorCount := 0
	for i := 0; i < len(expected); i++ {
		diff := timeOut[i] - expected[i]
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			t.Errorf("sample %d: got %f, expected %f (diff %f)", i, timeOut[i], expected[i], diff)
			errorCount++
			if errorCount > 10 {
				t.Fatalf("too many errors, stopping after %d mismatches", errorCount)
			}
		}
	}
}
