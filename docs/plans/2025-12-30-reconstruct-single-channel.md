# Spectral Reconstruction - Single Channel Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement `ReconstructSingleChannel` function that orchestrates all spectral processing tools for a single audio channel.

**Architecture:** The reconstruction function combines all already-implemented spectral tools (pulse decode, inverse quantization, scale factors, PNS, TNS, LTP, IC prediction) into a single unified pipeline. The function takes parsed syntax data and quantized spectral coefficients as input and produces dequantized, processed spectral coefficients ready for the filter bank.

**Tech Stack:** Go 1.21+, existing spectrum/syntax packages

---

## Background

### FAAD2 Reference

The function ports `reconstruct_single_channel()` from `~/dev/faad2/libfaad/specrec.c:905-1129`.

### Processing Order (from FAAD2)

1. Pulse decode (add pulse amplitudes to quantized data)
2. `quant_to_spec` (inverse quantization + scale factor application combined)
3. PNS decode (generate noise for noise bands)
4. IC Prediction (MAIN profile only)
5. PNS reset pred state (MAIN profile only)
6. LTP prediction (LTP profile only)
7. TNS decode (temporal noise shaping)
8. (DRC - optional, deferred to Phase 6)
9. (Filter bank - Phase 5, not part of this step)

### Existing Implementations

All the individual tools are already implemented:
- `spectrum/pulse.go` - `PulseDecode()`
- `spectrum/requant.go` - `InverseQuantize()`
- `spectrum/scalefac.go` - `ApplyScaleFactors()`
- `spectrum/pns.go` - `PNSDecode()`
- `spectrum/predict.go` - `ICPrediction()`, `PNSResetPredState()`
- `spectrum/ltp.go` - `LTPPrediction()`
- `spectrum/tns.go` - `TNSDecodeFrame()`

### Key Insight

FAAD2's `quant_to_spec` combines inverse quantization AND scale factor application in one loop. The Go implementation has them separated (`InverseQuantize` + `ApplyScaleFactors`). This is fine for correctness but could be optimized later if needed.

**Important:** FAAD2 uses `float32` (via `real_t`) in its floating-point mode. The current Go implementation uses `float64` for intermediate spectral data. This is intentional for precision and matches the approach used in other Go audio libraries.

However, the `ICPrediction` function currently uses `float32` because it ports the quantization routines that operate on IEEE 754 single-precision bit patterns. We need to handle this type mismatch carefully.

---

## Task 1: Create ReconstructConfig Structure

**Files:**
- Create: `internal/spectrum/reconstruct.go`

**Step 1.1: Write the failing test**

Create a test file with a basic test for the config structure.

```go
// internal/spectrum/reconstruct_test.go
package spectrum

import (
	"testing"

	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/syntax"
)

func TestReconstructConfig_Defaults(t *testing.T) {
	ics := &syntax.ICStream{}
	ele := &syntax.Element{}

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4, // 44100 Hz
	}

	if cfg.FrameLength != 1024 {
		t.Errorf("FrameLength: got %d, want 1024", cfg.FrameLength)
	}
	if cfg.ObjectType != aac.ObjectTypeLC {
		t.Errorf("ObjectType: got %d, want %d", cfg.ObjectType, aac.ObjectTypeLC)
	}
}
```

**Step 1.2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run TestReconstructConfig -v`
Expected: FAIL with "ReconstructSingleChannelConfig" not defined

**Step 1.3: Write minimal implementation**

```go
// internal/spectrum/reconstruct.go
package spectrum

import (
	"github.com/llehouerou/go-aac"
	"github.com/llehouerou/go-aac/internal/syntax"
)

// ReconstructSingleChannelConfig holds configuration for single channel reconstruction.
//
// Ported from: reconstruct_single_channel() parameters in ~/dev/faad2/libfaad/specrec.c:905-906
type ReconstructSingleChannelConfig struct {
	// ICS is the individual channel stream containing parsed syntax data
	ICS *syntax.ICStream

	// Element is the syntax element (SCE/LFE)
	Element *syntax.Element

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16

	// ObjectType is the AAC object type
	ObjectType aac.ObjectType

	// SRIndex is the sample rate index (0-15)
	SRIndex uint8

	// PredState is the predictor state for MAIN profile (nil if not MAIN)
	PredState []PredState

	// LTPState is the LTP state buffer for LTP profile (nil if not LTP)
	LTPState []int16

	// LTPFilterBank is the forward MDCT for LTP (nil if not LTP)
	LTPFilterBank ForwardMDCT

	// WindowShape is the current window shape
	WindowShape uint8

	// WindowShapePrev is the previous window shape
	WindowShapePrev uint8

	// PNSState is the PNS random number generator state
	PNSState *PNSState
}
```

**Step 1.4: Run test to verify it passes**

Run: `go test ./internal/spectrum -run TestReconstructConfig -v`
Expected: PASS

**Step 1.5: Commit**

```bash
git add internal/spectrum/reconstruct.go internal/spectrum/reconstruct_test.go
git commit -m "feat(spectrum): add ReconstructSingleChannelConfig structure

Ported from: reconstruct_single_channel() parameters
Source: ~/dev/faad2/libfaad/specrec.c:905-906

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Implement Basic ReconstructSingleChannel Function

**Files:**
- Modify: `internal/spectrum/reconstruct.go`
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 2.1: Write failing test for basic reconstruction**

```go
func TestReconstructSingleChannel_BasicLC(t *testing.T) {
	// Setup minimal ICS for AAC-LC
	ics := &syntax.ICStream{
		NumWindowGroups:   1,
		NumWindows:        1,
		MaxSFB:            4,
		NumSWB:            4,
		WindowSequence:    syntax.OnlyLongSequence,
		GlobalGain:        100,
		PulseDataPresent:  false,
		TNSDataPresent:    false,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffset[3] = 12
	ics.SWBOffset[4] = 16
	ics.SWBOffsetMax = 1024
	// Set codebooks to normal (not noise/intensity)
	for g := 0; g < 8; g++ {
		for sfb := 0; sfb < 51; sfb++ {
			ics.SFBCB[g][sfb] = 1 // Normal codebook
		}
	}
	// Set scale factors to 100 (neutral, multiplier = 1.0)
	for g := 0; g < 8; g++ {
		for sfb := 0; sfb < 51; sfb++ {
			ics.ScaleFactors[g][sfb] = 100
		}
	}

	ele := &syntax.Element{}

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	// Input: quantized data (small values for IQ table lookup)
	quantData := make([]int16, 1024)
	quantData[0] = 1
	quantData[1] = 2
	quantData[2] = -1
	quantData[3] = -2

	// Output buffer
	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}

	// Verify non-zero output for non-zero input
	if specData[0] == 0 {
		t.Error("specData[0] should be non-zero")
	}
	if specData[1] == 0 {
		t.Error("specData[1] should be non-zero")
	}
}
```

**Step 2.2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run TestReconstructSingleChannel_BasicLC -v`
Expected: FAIL with "ReconstructSingleChannel" not defined

**Step 2.3: Write implementation**

```go
// ReconstructSingleChannel performs spectral reconstruction for a single channel.
// This is the main entry point for converting parsed syntax data and quantized
// spectral coefficients into dequantized, processed spectral data ready for
// the filter bank.
//
// Processing order:
// 1. Pulse decode (if present, long blocks only)
// 2. Inverse quantization (|x|^(4/3))
// 3. Apply scale factors (multiply by 2^((sf-100)/4))
// 4. PNS decode (noise substitution)
// 5. IC Prediction (MAIN profile only)
// 6. PNS reset pred state (MAIN profile only)
// 7. LTP prediction (LTP profile only)
// 8. TNS decode (temporal noise shaping)
//
// Ported from: reconstruct_single_channel() in ~/dev/faad2/libfaad/specrec.c:905-1129
func ReconstructSingleChannel(quantData []int16, specData []float64, cfg *ReconstructSingleChannelConfig) error {
	ics := cfg.ICS
	frameLen := cfg.FrameLength

	// 1. Pulse decode (long blocks only)
	if ics.PulseDataPresent {
		if ics.WindowSequence == syntax.EightShortSequence {
			return syntax.ErrPulseInShortBlock
		}
		if err := PulseDecode(ics, quantData, frameLen); err != nil {
			return err
		}
	}

	// 2. Inverse quantization: spec[i] = sign(quant[i]) * |quant[i]|^(4/3)
	if err := InverseQuantize(quantData, specData); err != nil {
		return err
	}

	// 3. Apply scale factors: spec[i] *= 2^((sf-100)/4)
	ApplyScaleFactors(specData, &ApplyScaleFactorsConfig{
		ICS:         ics,
		FrameLength: frameLen,
	})

	// 4. PNS decode (generate noise for noise bands)
	if cfg.PNSState != nil {
		PNSDecode(specData, nil, cfg.PNSState, &PNSDecodeConfig{
			ICSL:        ics,
			ICSR:        nil,
			FrameLength: frameLen,
			ChannelPair: false,
			ObjectType:  uint8(cfg.ObjectType),
		})
	}

	// 5 & 6. IC Prediction (MAIN profile only)
	if cfg.ObjectType == aac.ObjectTypeMain && cfg.PredState != nil {
		// Convert float64 to float32 for IC prediction
		specData32 := make([]float32, len(specData))
		for i, v := range specData {
			specData32[i] = float32(v)
		}

		ICPrediction(ics, specData32, cfg.PredState, frameLen, cfg.SRIndex)

		// Convert back to float64
		for i, v := range specData32 {
			specData[i] = float64(v)
		}

		// Reset predictors for PNS bands
		PNSResetPredState(ics, cfg.PredState)
	}

	// 7. LTP prediction (LTP profile only)
	if IsLTPObjectType(cfg.ObjectType) && cfg.LTPState != nil {
		LTPPredictionWithMDCT(specData, cfg.LTPState, cfg.LTPFilterBank, &LTPConfig{
			ICS:             ics,
			LTP:             &ics.LTP,
			SRIndex:         cfg.SRIndex,
			ObjectType:      cfg.ObjectType,
			FrameLength:     frameLen,
			WindowShape:     cfg.WindowShape,
			WindowShapePrev: cfg.WindowShapePrev,
		})
	}

	// 8. TNS decode (temporal noise shaping)
	if ics.TNSDataPresent {
		TNSDecodeFrame(specData, &TNSDecodeConfig{
			ICS:         ics,
			SRIndex:     cfg.SRIndex,
			ObjectType:  cfg.ObjectType,
			FrameLength: frameLen,
		})
	}

	return nil
}
```

**Step 2.4: Run test to verify it passes**

Run: `go test ./internal/spectrum -run TestReconstructSingleChannel_BasicLC -v`
Expected: PASS

**Step 2.5: Commit**

```bash
git add internal/spectrum/reconstruct.go internal/spectrum/reconstruct_test.go
git commit -m "feat(spectrum): implement ReconstructSingleChannel function

Combines all spectral tools for single channel processing:
- Pulse decode
- Inverse quantization
- Scale factor application
- PNS decode
- IC prediction (MAIN profile)
- LTP prediction (LTP profile)
- TNS decode

Ported from: reconstruct_single_channel()
Source: ~/dev/faad2/libfaad/specrec.c:905-1129

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Add Tests for Pulse Processing

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 3.1: Write test for pulse processing**

```go
func TestReconstructSingleChannel_WithPulse(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups:  1,
		NumWindows:       1,
		MaxSFB:           4,
		NumSWB:           4,
		WindowSequence:   syntax.OnlyLongSequence,
		GlobalGain:       100,
		PulseDataPresent: true,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffset[3] = 12
	ics.SWBOffset[4] = 16
	ics.SWBOffsetMax = 1024
	// Normal codebook
	ics.SFBCB[0][0] = 1
	ics.SFBCB[0][1] = 1
	ics.SFBCB[0][2] = 1
	ics.SFBCB[0][3] = 1
	// Scale factor 100 = 1.0
	ics.ScaleFactors[0][0] = 100
	ics.ScaleFactors[0][1] = 100
	ics.ScaleFactors[0][2] = 100
	ics.ScaleFactors[0][3] = 100

	// Setup pulse data
	ics.Pul.NumberPulse = 0       // 1 pulse
	ics.Pul.PulseStartSFB = 0     // Start at SFB 0
	ics.Pul.PulseOffset[0] = 2    // Position 2
	ics.Pul.PulseAmp[0] = 5       // Add 5

	ele := &syntax.Element{}

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	// Input: quantized data
	quantData := make([]int16, 1024)
	quantData[0] = 1
	quantData[1] = 2
	quantData[2] = 3 // Will become 8 after pulse (3+5)
	quantData[3] = 4

	// Copy for reference
	quantDataNoPulse := make([]int16, 1024)
	copy(quantDataNoPulse, quantData)

	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}

	// Verify pulse was applied (quantData should be modified)
	if quantData[2] != 8 {
		t.Errorf("quantData[2] after pulse: got %d, want 8", quantData[2])
	}
}
```

**Step 3.2: Run test**

Run: `go test ./internal/spectrum -run TestReconstructSingleChannel_WithPulse -v`
Expected: PASS

**Step 3.3: Write test for pulse in short block (should error)**

```go
func TestReconstructSingleChannel_PulseInShortBlock_Error(t *testing.T) {
	ics := &syntax.ICStream{
		WindowSequence:   syntax.EightShortSequence,
		PulseDataPresent: true,
	}

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     &syntax.Element{},
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
	}

	quantData := make([]int16, 1024)
	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err == nil {
		t.Error("expected error for pulse in short block")
	}
	if err != syntax.ErrPulseInShortBlock {
		t.Errorf("got error %v, want ErrPulseInShortBlock", err)
	}
}
```

**Step 3.4: Run test**

Run: `go test ./internal/spectrum -run TestReconstructSingleChannel_PulseInShortBlock -v`
Expected: PASS

**Step 3.5: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "test(spectrum): add pulse processing tests for ReconstructSingleChannel

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Add Tests for PNS Processing

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 4.1: Write test for noise band handling**

```go
func TestReconstructSingleChannel_WithNoiseBand(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 8
	ics.SWBOffset[2] = 16
	ics.SWBOffsetMax = 1024

	// First band: noise, second band: normal
	ics.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	ics.SFBCB[0][1] = 1
	ics.ScaleFactors[0][0] = 0 // Noise scale
	ics.ScaleFactors[0][1] = 100

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     &syntax.Element{},
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData := make([]int16, 1024)
	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}

	// First 8 samples should have noise (non-zero)
	hasNoise := false
	for i := 0; i < 8; i++ {
		if specData[i] != 0 {
			hasNoise = true
			break
		}
	}
	if !hasNoise {
		t.Error("noise band should have non-zero values")
	}
}
```

**Step 4.2: Run test**

Run: `go test ./internal/spectrum -run TestReconstructSingleChannel_WithNoiseBand -v`
Expected: PASS (need to add huffman import to test file)

**Step 4.3: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "test(spectrum): add PNS noise band test for ReconstructSingleChannel

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Add Tests for TNS Processing

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 5.1: Write test for TNS processing**

```go
func TestReconstructSingleChannel_WithTNS(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          4,
		NumSWB:          4,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		TNSDataPresent:  true,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 8
	ics.SWBOffset[2] = 16
	ics.SWBOffset[3] = 24
	ics.SWBOffset[4] = 32
	ics.SWBOffsetMax = 1024
	ics.SFBCB[0][0] = 1
	ics.SFBCB[0][1] = 1
	ics.SFBCB[0][2] = 1
	ics.SFBCB[0][3] = 1
	ics.ScaleFactors[0][0] = 100
	ics.ScaleFactors[0][1] = 100
	ics.ScaleFactors[0][2] = 100
	ics.ScaleFactors[0][3] = 100

	// Setup simple TNS data
	ics.TNS.NFilt[0] = 1
	ics.TNS.Length[0][0] = 4
	ics.TNS.Order[0][0] = 1
	ics.TNS.Direction[0][0] = 0
	ics.TNS.CoefRes[0] = 1
	ics.TNS.Coef[0][0][0] = 4

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     &syntax.Element{},
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData := make([]int16, 1024)
	for i := 0; i < 32; i++ {
		quantData[i] = 10
	}

	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}

	// TNS should have modified the spectrum
	// Just verify no error and non-zero output
	hasValue := false
	for i := 0; i < 32; i++ {
		if specData[i] != 0 {
			hasValue = true
			break
		}
	}
	if !hasValue {
		t.Error("spectrum should have non-zero values after TNS")
	}
}
```

**Step 5.2: Run test**

Run: `go test ./internal/spectrum -run TestReconstructSingleChannel_WithTNS -v`
Expected: PASS

**Step 5.3: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "test(spectrum): add TNS processing test for ReconstructSingleChannel

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Add Test for MAIN Profile IC Prediction

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 6.1: Write test for MAIN profile**

```go
func TestReconstructSingleChannel_MainProfile_ICPrediction(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups:       1,
		NumWindows:            1,
		MaxSFB:                4,
		NumSWB:                4,
		WindowSequence:        syntax.OnlyLongSequence,
		GlobalGain:            100,
		PredictorDataPresent:  true,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffset[3] = 12
	ics.SWBOffset[4] = 16
	ics.SWBOffsetMax = 1024
	ics.SFBCB[0][0] = 1
	ics.SFBCB[0][1] = 1
	ics.SFBCB[0][2] = 1
	ics.SFBCB[0][3] = 1
	ics.ScaleFactors[0][0] = 100
	ics.ScaleFactors[0][1] = 100
	ics.ScaleFactors[0][2] = 100
	ics.ScaleFactors[0][3] = 100

	// Enable prediction for first 2 bands
	ics.Pred.PredictionUsed[0] = true
	ics.Pred.PredictionUsed[1] = true

	// Create predictor state
	predState := make([]PredState, 1024)
	ResetAllPredictors(predState, 1024)

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     &syntax.Element{},
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeMain, // MAIN profile
		SRIndex:     4,
		PNSState:    NewPNSState(),
		PredState:   predState,
	}

	quantData := make([]int16, 1024)
	for i := 0; i < 16; i++ {
		quantData[i] = int16(i + 1)
	}

	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}

	// Predictor state should be updated
	stateUpdated := false
	for i := 0; i < 16; i++ {
		if predState[i].R[0] != 0 || predState[i].R[1] != 0 {
			stateUpdated = true
			break
		}
	}
	if !stateUpdated {
		t.Error("predictor state should be updated")
	}
}
```

**Step 6.2: Run test**

Run: `go test ./internal/spectrum -run TestReconstructSingleChannel_MainProfile -v`
Expected: PASS

**Step 6.3: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "test(spectrum): add MAIN profile IC prediction test

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Add Test for LTP Profile

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 7.1: Write test for LTP profile**

```go
func TestReconstructSingleChannel_LTPProfile(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          4,
		NumSWB:          4,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics.WindowGroupLength[0] = 1
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffset[3] = 12
	ics.SWBOffset[4] = 16
	ics.SWBOffsetMax = 1024
	ics.SFBCB[0][0] = 1
	ics.ScaleFactors[0][0] = 100

	// Enable LTP
	ics.LTP.DataPresent = true
	ics.LTP.Lag = 1024
	ics.LTP.Coef = 4
	ics.LTP.LastBand = 4
	ics.LTP.LongUsed[0] = true
	ics.LTP.LongUsed[1] = true

	// Create LTP state (empty for this test)
	ltpState := make([]int16, 4*1024)

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     &syntax.Element{},
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLTP, // LTP profile
		SRIndex:     4,
		PNSState:    NewPNSState(),
		LTPState:    ltpState,
		// LTPFilterBank is nil, so LTP will be skipped
	}

	quantData := make([]int16, 1024)
	specData := make([]float64, 1024)

	// Should succeed (LTP skipped due to no filterbank)
	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}
}
```

**Step 7.2: Run test**

Run: `go test ./internal/spectrum -run TestReconstructSingleChannel_LTPProfile -v`
Expected: PASS

**Step 7.3: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "test(spectrum): add LTP profile test for ReconstructSingleChannel

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Add Test for Short Blocks

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 8.1: Write test for short block processing**

```go
func TestReconstructSingleChannel_ShortBlocks(t *testing.T) {
	ics := &syntax.ICStream{
		NumWindowGroups: 2,
		NumWindows:      8,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.EightShortSequence,
		GlobalGain:      100,
	}
	ics.WindowGroupLength[0] = 4
	ics.WindowGroupLength[1] = 4
	ics.SWBOffset[0] = 0
	ics.SWBOffset[1] = 4
	ics.SWBOffset[2] = 8
	ics.SWBOffsetMax = 128 // 1024/8 for short blocks
	ics.SFBCB[0][0] = 1
	ics.SFBCB[0][1] = 1
	ics.SFBCB[1][0] = 1
	ics.SFBCB[1][1] = 1
	ics.ScaleFactors[0][0] = 100
	ics.ScaleFactors[0][1] = 100
	ics.ScaleFactors[1][0] = 100
	ics.ScaleFactors[1][1] = 100

	cfg := &ReconstructSingleChannelConfig{
		ICS:         ics,
		Element:     &syntax.Element{},
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData := make([]int16, 1024)
	for i := 0; i < 64; i++ {
		quantData[i] = 1
	}

	specData := make([]float64, 1024)

	err := ReconstructSingleChannel(quantData, specData, cfg)
	if err != nil {
		t.Fatalf("ReconstructSingleChannel failed: %v", err)
	}

	// Verify output has values
	hasValue := false
	for i := 0; i < 64; i++ {
		if specData[i] != 0 {
			hasValue = true
			break
		}
	}
	if !hasValue {
		t.Error("short block processing should produce non-zero values")
	}
}
```

**Step 8.2: Run test**

Run: `go test ./internal/spectrum -run TestReconstructSingleChannel_ShortBlocks -v`
Expected: PASS

**Step 8.3: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "test(spectrum): add short block test for ReconstructSingleChannel

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 9: Run Full Test Suite and Fix Any Issues

**Step 9.1: Run all tests**

Run: `go test ./internal/spectrum -v`
Expected: All tests pass

**Step 9.2: Run linter**

Run: `make lint`
Expected: No errors

**Step 9.3: Run all checks**

Run: `make check`
Expected: All checks pass

**Step 9.4: Final commit if any fixes needed**

```bash
git add -A
git commit -m "fix(spectrum): address any issues from full test suite

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Summary

This plan implements Step 4.11 (Spectral Reconstruction - Single Channel) by creating:

1. `ReconstructSingleChannelConfig` structure to hold all configuration
2. `ReconstructSingleChannel` function that orchestrates:
   - Pulse decode
   - Inverse quantization
   - Scale factor application
   - PNS decode
   - IC prediction (MAIN profile)
   - LTP prediction (LTP profile)
   - TNS decode

The function integrates all existing spectral tools into a single pipeline, matching FAAD2's `reconstruct_single_channel()` processing order.

**Dependencies:** All individual tools are already implemented in the spectrum package. This task only creates the orchestration layer.

**Next Steps:** After this is complete, proceed to Step 4.12 (Spectral Reconstruction - Channel Pair) which adds M/S stereo and intensity stereo processing for CPE elements.
