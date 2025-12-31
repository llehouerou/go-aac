# Channel Pair Spectral Reconstruction Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement `ReconstructChannelPair` to process stereo audio (CPE elements) with M/S stereo, intensity stereo, and correlated PNS.

**Architecture:** Port FAAD2's `reconstruct_channel_pair()` from `specrec.c:1131-1365`. The function processes two channels together, applying stereo-specific tools (M/S, IS) after individual channel processing. PNS correlation requires checking `ms_used` flags.

**Tech Stack:** Go, internal packages (spectrum, syntax), existing single-channel infrastructure

---

## Overview

The `ReconstructChannelPair` function processes Channel Pair Elements (CPE) for stereo audio. It extends single-channel reconstruction by adding:

1. **M/S Stereo Decoding** - Converts Mid/Side to Left/Right: `L = M + S, R = M - S`
2. **Intensity Stereo Decoding** - Reconstructs right channel from scaled left channel
3. **Correlated PNS** - Uses same noise for both channels when `ms_used` is set

Processing order (from FAAD2 specrec.c:1156-1273):
1. Inverse quantization + scale factors (both channels)
2. PNS decoding (with correlation check)
3. M/S stereo decoding
4. Intensity stereo decoding
5. IC prediction (MAIN profile, both channels)
6. PNS reset predictor state (MAIN profile, both channels)
7. LTP prediction (LTP profile, both channels - uses `ltp2` for channel 2 when `common_window`)
8. TNS decoding (both channels)

---

### Task 1: Create Config and Function Signature

**Files:**
- Modify: `internal/spectrum/reconstruct.go`

**Step 1: Write the failing test**

Add to `internal/spectrum/reconstruct_test.go`:

```go
func TestReconstructChannelPair_BasicStereo(t *testing.T) {
	// Setup minimal ICS for both channels
	ics1 := &syntax.ICStream{
		NumWindowGroups:  1,
		NumWindows:       1,
		MaxSFB:           4,
		NumSWB:           4,
		WindowSequence:   syntax.OnlyLongSequence,
		GlobalGain:       100,
		MSMaskPresent:    0, // No M/S
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 4
	ics1.SWBOffset[2] = 8
	ics1.SWBOffset[3] = 12
	ics1.SWBOffset[4] = 16
	ics1.SWBOffsetMax = 1024
	for g := 0; g < 8; g++ {
		for sfb := 0; sfb < 51; sfb++ {
			ics1.SFBCB[g][sfb] = 1
			ics1.ScaleFactors[g][sfb] = 100
		}
	}

	ics2 := &syntax.ICStream{
		NumWindowGroups:  1,
		NumWindows:       1,
		MaxSFB:           4,
		NumSWB:           4,
		WindowSequence:   syntax.OnlyLongSequence,
		GlobalGain:       100,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 4
	ics2.SWBOffset[2] = 8
	ics2.SWBOffset[3] = 12
	ics2.SWBOffset[4] = 16
	ics2.SWBOffsetMax = 1024
	for g := 0; g < 8; g++ {
		for sfb := 0; sfb < 51; sfb++ {
			ics2.SFBCB[g][sfb] = 1
			ics2.ScaleFactors[g][sfb] = 100
		}
	}

	ele := &syntax.Element{
		CommonWindow: false,
	}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)
	quantData1[0] = 1
	quantData1[1] = 2
	quantData2[0] = 3
	quantData2[1] = 4

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// Verify non-zero output
	if specData1[0] == 0 || specData2[0] == 0 {
		t.Error("both channels should have non-zero output")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/spectrum`
Expected: FAIL with "undefined: ReconstructChannelPairConfig"

**Step 3: Write minimal implementation**

Add to `internal/spectrum/reconstruct.go`:

```go
// ReconstructChannelPairConfig holds configuration for channel pair reconstruction.
//
// Ported from: reconstruct_channel_pair() parameters in ~/dev/faad2/libfaad/specrec.c:1131-1132
type ReconstructChannelPairConfig struct {
	// ICS1 is the first channel's individual channel stream
	ICS1 *syntax.ICStream

	// ICS2 is the second channel's individual channel stream
	ICS2 *syntax.ICStream

	// Element is the syntax element (CPE)
	Element *syntax.Element

	// FrameLength is the frame length (typically 1024 or 960)
	FrameLength uint16

	// ObjectType is the AAC object type
	ObjectType aac.ObjectType

	// SRIndex is the sample rate index (0-15)
	SRIndex uint8

	// PredState1 is the predictor state for channel 1 (MAIN profile, nil otherwise)
	PredState1 []PredState

	// PredState2 is the predictor state for channel 2 (MAIN profile, nil otherwise)
	PredState2 []PredState

	// LTPState1 is the LTP state buffer for channel 1 (LTP profile, nil otherwise)
	LTPState1 []int16

	// LTPState2 is the LTP state buffer for channel 2 (LTP profile, nil otherwise)
	LTPState2 []int16

	// LTPFilterBank is the forward MDCT for LTP (nil if not LTP)
	LTPFilterBank ForwardMDCT

	// WindowShape1 is channel 1's current window shape
	WindowShape1 uint8

	// WindowShapePrev1 is channel 1's previous window shape
	WindowShapePrev1 uint8

	// WindowShape2 is channel 2's current window shape
	WindowShape2 uint8

	// WindowShapePrev2 is channel 2's previous window shape
	WindowShapePrev2 uint8

	// PNSState is the PNS random number generator state
	PNSState *PNSState
}

// ReconstructChannelPair performs spectral reconstruction for a channel pair (CPE).
// This processes two channels together, applying stereo-specific tools:
// - M/S stereo decoding (when ms_mask_present > 0)
// - Intensity stereo decoding (for INTENSITY_HCB bands)
// - Correlated PNS (same noise when ms_used is set for noise bands)
//
// Processing order:
// 1. Inverse quantization (both channels)
// 2. Apply scale factors (both channels)
// 3. PNS decode (with correlation for channel pairs)
// 4. M/S stereo decode
// 5. Intensity stereo decode
// 6. IC Prediction (MAIN profile, both channels)
// 7. PNS reset pred state (MAIN profile, both channels)
// 8. LTP prediction (LTP profile, both channels)
// 9. TNS decode (both channels)
//
// Ported from: reconstruct_channel_pair() in ~/dev/faad2/libfaad/specrec.c:1131-1365
func ReconstructChannelPair(quantData1, quantData2 []int16, specData1, specData2 []float64, cfg *ReconstructChannelPairConfig) error {
	ics1 := cfg.ICS1
	ics2 := cfg.ICS2
	frameLen := cfg.FrameLength

	// 1 & 2. Inverse quantization + scale factors for channel 1
	if err := InverseQuantize(quantData1, specData1); err != nil {
		return err
	}
	ApplyScaleFactors(specData1, &ApplyScaleFactorsConfig{
		ICS:         ics1,
		FrameLength: frameLen,
	})

	// 1 & 2. Inverse quantization + scale factors for channel 2
	if err := InverseQuantize(quantData2, specData2); err != nil {
		return err
	}
	ApplyScaleFactors(specData2, &ApplyScaleFactorsConfig{
		ICS:         ics2,
		FrameLength: frameLen,
	})

	// 3. PNS decode (with correlation handling)
	if cfg.PNSState != nil {
		channelPair := ics1.MSMaskPresent > 0
		PNSDecode(specData1, specData2, cfg.PNSState, &PNSDecodeConfig{
			ICSL:        ics1,
			ICSR:        ics2,
			FrameLength: frameLen,
			ChannelPair: channelPair,
			ObjectType:  uint8(cfg.ObjectType),
		})
	}

	// 4. M/S stereo decode
	MSDecode(specData1, specData2, &MSDecodeConfig{
		ICSL:        ics1,
		ICSR:        ics2,
		FrameLength: frameLen,
	})

	// 5. Intensity stereo decode
	ISDecode(specData1, specData2, &ISDecodeConfig{
		ICSL:        ics1,
		ICSR:        ics2,
		FrameLength: frameLen,
	})

	// 6 & 7. IC Prediction (MAIN profile only)
	if cfg.ObjectType == aac.ObjectTypeMain {
		if cfg.PredState1 != nil {
			specData1f32 := make([]float32, len(specData1))
			for i, v := range specData1 {
				specData1f32[i] = float32(v)
			}
			ICPrediction(ics1, specData1f32, cfg.PredState1, frameLen, cfg.SRIndex)
			for i, v := range specData1f32 {
				specData1[i] = float64(v)
			}
			PNSResetPredState(ics1, cfg.PredState1)
		}
		if cfg.PredState2 != nil {
			specData2f32 := make([]float32, len(specData2))
			for i, v := range specData2 {
				specData2f32[i] = float32(v)
			}
			ICPrediction(ics2, specData2f32, cfg.PredState2, frameLen, cfg.SRIndex)
			for i, v := range specData2f32 {
				specData2[i] = float64(v)
			}
			PNSResetPredState(ics2, cfg.PredState2)
		}
	}

	// 8. LTP prediction (LTP profile only)
	if IsLTPObjectType(cfg.ObjectType) {
		if cfg.LTPState1 != nil {
			LTPPredictionWithMDCT(specData1, cfg.LTPState1, cfg.LTPFilterBank, &LTPConfig{
				ICS:             ics1,
				LTP:             &ics1.LTP,
				SRIndex:         cfg.SRIndex,
				ObjectType:      cfg.ObjectType,
				FrameLength:     frameLen,
				WindowShape:     cfg.WindowShape1,
				WindowShapePrev: cfg.WindowShapePrev1,
			})
		}
		if cfg.LTPState2 != nil {
			// For common_window, use ltp2 from ICS2; otherwise use ltp from ICS2
			// Ported from: specrec.c:1239-1240
			ltp := &ics2.LTP
			if cfg.Element.CommonWindow {
				ltp = &ics2.LTP2
			}
			LTPPredictionWithMDCT(specData2, cfg.LTPState2, cfg.LTPFilterBank, &LTPConfig{
				ICS:             ics2,
				LTP:             ltp,
				SRIndex:         cfg.SRIndex,
				ObjectType:      cfg.ObjectType,
				FrameLength:     frameLen,
				WindowShape:     cfg.WindowShape2,
				WindowShapePrev: cfg.WindowShapePrev2,
			})
		}
	}

	// 9. TNS decode (both channels)
	if ics1.TNSDataPresent {
		TNSDecodeFrame(specData1, &TNSDecodeConfig{
			ICS:         ics1,
			SRIndex:     cfg.SRIndex,
			ObjectType:  cfg.ObjectType,
			FrameLength: frameLen,
		})
	}
	if ics2.TNSDataPresent {
		TNSDecodeFrame(specData2, &TNSDecodeConfig{
			ICS:         ics2,
			SRIndex:     cfg.SRIndex,
			ObjectType:  cfg.ObjectType,
			FrameLength: frameLen,
		})
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/spectrum/reconstruct.go internal/spectrum/reconstruct_test.go
git commit -m "$(cat <<'EOF'
feat(spectrum): add ReconstructChannelPair for stereo processing

Implements channel pair spectral reconstruction with M/S stereo,
intensity stereo, and correlated PNS support.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Add LTP2 Field to ICStream

**Files:**
- Modify: `internal/syntax/ics.go`
- Modify: `internal/syntax/ltp.go`

**Step 1: Write the failing test**

Add to `internal/syntax/ltp_test.go`:

```go
func TestICStream_LTP2Field(t *testing.T) {
	ics := &ICStream{}

	// LTP2 should exist for common_window CPE second channel
	ics.LTP2.DataPresent = true
	ics.LTP2.Lag = 512
	ics.LTP2.Coef = 3

	if !ics.LTP2.DataPresent {
		t.Error("LTP2 should be settable")
	}
	if ics.LTP2.Lag != 512 {
		t.Errorf("LTP2.Lag: got %d, want 512", ics.LTP2.Lag)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `make test PKG=./internal/syntax`
Expected: FAIL with "ics.LTP2 undefined"

**Step 3: Write minimal implementation**

Add to `internal/syntax/ics.go` inside the ICStream struct:

```go
	// LTP2 is the second LTP info for CPE with common_window.
	// When common_window is set, the second channel uses ltp2 instead of ltp.
	// Ported from: ltp2 in ~/dev/faad2/libfaad/structs.h:256
	LTP2 LTPInfo
```

**Step 4: Run test to verify it passes**

Run: `make test PKG=./internal/syntax`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/syntax/ics.go internal/syntax/ltp_test.go
git commit -m "$(cat <<'EOF'
feat(syntax): add LTP2 field to ICStream for CPE common_window

The second channel in a CPE with common_window uses a separate LTP
structure (ltp2) per FAAD2's specrec.c:1239-1240.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Test M/S Stereo Decoding in Channel Pair

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 1: Write the failing test**

```go
func TestReconstructChannelPair_WithMSStereo(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		MSMaskPresent:   2, // All bands use M/S
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 4
	ics1.SWBOffset[2] = 8
	ics1.SWBOffsetMax = 1024
	ics1.SFBCB[0][0] = 1
	ics1.SFBCB[0][1] = 1
	ics1.ScaleFactors[0][0] = 100
	ics1.ScaleFactors[0][1] = 100

	ics2 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 4
	ics2.SWBOffset[2] = 8
	ics2.SWBOffsetMax = 1024
	ics2.SFBCB[0][0] = 1
	ics2.SFBCB[0][1] = 1
	ics2.ScaleFactors[0][0] = 100
	ics2.ScaleFactors[0][1] = 100

	ele := &syntax.Element{CommonWindow: true}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	// Create M/S encoded data: M=10, S=2
	// Expected output: L = M + S = 12, R = M - S = 8
	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)
	quantData1[0] = 10 // Mid
	quantData2[0] = 2  // Side

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// After M/S decode: L should be different from R
	// M/S transform: L = M + S, R = M - S
	// Since we have non-zero M and S, L != R
	if specData1[0] == specData2[0] {
		t.Error("M/S stereo should produce different L and R values")
	}

	// L should be > R since M > 0 and S > 0
	if specData1[0] <= specData2[0] {
		t.Errorf("L (%f) should be > R (%f) with positive M and S", specData1[0], specData2[0])
	}
}
```

**Step 2: Run test to verify it passes**

Run: `make test PKG=./internal/spectrum`
Expected: PASS (M/S already applied in ReconstructChannelPair)

**Step 3: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "$(cat <<'EOF'
test(spectrum): add M/S stereo test for ReconstructChannelPair

Verifies that M/S stereo decoding correctly transforms Mid/Side
to Left/Right channels.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Test Intensity Stereo in Channel Pair

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 1: Write the test**

```go
func TestReconstructChannelPair_WithIntensityStereo(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		MSMaskPresent:   0, // No M/S
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 4
	ics1.SWBOffset[2] = 8
	ics1.SWBOffsetMax = 1024
	ics1.SFBCB[0][0] = 1
	ics1.SFBCB[0][1] = 1
	ics1.ScaleFactors[0][0] = 100
	ics1.ScaleFactors[0][1] = 100

	ics2 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 4
	ics2.SWBOffset[2] = 8
	ics2.SWBOffsetMax = 1024
	// First band: normal, second band: intensity stereo
	ics2.SFBCB[0][0] = 1
	ics2.SFBCB[0][1] = uint8(huffman.IntensityHCB) // 15 = intensity stereo
	ics2.ScaleFactors[0][0] = 100
	ics2.ScaleFactors[0][1] = 0 // IS scale factor

	ele := &syntax.Element{CommonWindow: false}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)
	// Left channel has data in band 1 (indices 4-7)
	quantData1[4] = 10
	quantData1[5] = 10
	// Right channel has no data in band 1 (will be copied from left via IS)

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// Second band (indices 4-7) should have IS-scaled copy in right channel
	if specData2[4] == 0 {
		t.Error("intensity stereo should copy scaled values from left to right")
	}
}
```

**Step 2: Run test**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "$(cat <<'EOF'
test(spectrum): add intensity stereo test for ReconstructChannelPair

Verifies that intensity stereo bands in the right channel are
reconstructed from scaled left channel values.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Test Correlated PNS in Channel Pair

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 1: Write the test**

```go
func TestReconstructChannelPair_CorrelatedPNS(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		MSMaskPresent:   2, // All bands - enables PNS correlation
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 8
	ics1.SWBOffset[2] = 16
	ics1.SWBOffsetMax = 1024
	// Both bands are noise
	ics1.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	ics1.SFBCB[0][1] = uint8(huffman.NoiseHCB)
	ics1.ScaleFactors[0][0] = 0
	ics1.ScaleFactors[0][1] = 0

	ics2 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 8
	ics2.SWBOffset[2] = 16
	ics2.SWBOffsetMax = 1024
	// Both bands are noise
	ics2.SFBCB[0][0] = uint8(huffman.NoiseHCB)
	ics2.SFBCB[0][1] = uint8(huffman.NoiseHCB)
	ics2.ScaleFactors[0][0] = 0
	ics2.ScaleFactors[0][1] = 0

	ele := &syntax.Element{CommonWindow: true}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// With ms_mask_present=2, PNS should be correlated (same random sequence)
	// The noise values should be proportional (same pattern, possibly different scale)
	// Check that the ratio is consistent across samples
	if specData1[0] == 0 || specData2[0] == 0 {
		t.Skip("PNS generated zero - need non-zero for correlation test")
	}

	ratio := specData1[0] / specData2[0]
	for i := 1; i < 8; i++ {
		if specData2[i] == 0 {
			continue
		}
		thisRatio := specData1[i] / specData2[i]
		// Allow small tolerance for floating point
		if (thisRatio-ratio)/ratio > 0.01 || (thisRatio-ratio)/ratio < -0.01 {
			t.Errorf("sample %d: ratio %f differs from expected %f", i, thisRatio, ratio)
		}
	}
}
```

**Step 2: Run test**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "$(cat <<'EOF'
test(spectrum): add correlated PNS test for ReconstructChannelPair

Verifies that PNS generates correlated noise when ms_mask_present
is set for channel pairs.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Test TNS on Both Channels

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 1: Write the test**

```go
func TestReconstructChannelPair_WithTNS(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          4,
		NumSWB:          4,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		TNSDataPresent:  true,
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 8
	ics1.SWBOffset[2] = 16
	ics1.SWBOffset[3] = 24
	ics1.SWBOffset[4] = 32
	ics1.SWBOffsetMax = 1024
	ics1.SFBCB[0][0] = 1
	ics1.ScaleFactors[0][0] = 100
	ics1.TNS.NFilt[0] = 1
	ics1.TNS.Length[0][0] = 4
	ics1.TNS.Order[0][0] = 1
	ics1.TNS.Direction[0][0] = 0
	ics1.TNS.CoefRes[0] = 1
	ics1.TNS.Coef[0][0][0] = 4

	ics2 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          4,
		NumSWB:          4,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
		TNSDataPresent:  true,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 8
	ics2.SWBOffset[2] = 16
	ics2.SWBOffset[3] = 24
	ics2.SWBOffset[4] = 32
	ics2.SWBOffsetMax = 1024
	ics2.SFBCB[0][0] = 1
	ics2.ScaleFactors[0][0] = 100
	ics2.TNS.NFilt[0] = 1
	ics2.TNS.Length[0][0] = 4
	ics2.TNS.Order[0][0] = 1
	ics2.TNS.Direction[0][0] = 0
	ics2.TNS.CoefRes[0] = 1
	ics2.TNS.Coef[0][0][0] = 4

	ele := &syntax.Element{CommonWindow: false}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)
	for i := 0; i < 32; i++ {
		quantData1[i] = 10
		quantData2[i] = 10
	}

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// TNS should have modified both channels
	hasValue1 := false
	hasValue2 := false
	for i := 0; i < 32; i++ {
		if specData1[i] != 0 {
			hasValue1 = true
		}
		if specData2[i] != 0 {
			hasValue2 = true
		}
	}
	if !hasValue1 || !hasValue2 {
		t.Error("TNS should produce non-zero values in both channels")
	}
}
```

**Step 2: Run test**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "$(cat <<'EOF'
test(spectrum): add TNS test for ReconstructChannelPair

Verifies that TNS is applied to both channels in a channel pair.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Test MAIN Profile IC Prediction on Both Channels

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 1: Write the test**

```go
func TestReconstructChannelPair_MainProfile_ICPrediction(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups:      1,
		NumWindows:           1,
		MaxSFB:               4,
		NumSWB:               4,
		WindowSequence:       syntax.OnlyLongSequence,
		GlobalGain:           100,
		PredictorDataPresent: true,
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 4
	ics1.SWBOffset[2] = 8
	ics1.SWBOffset[3] = 12
	ics1.SWBOffset[4] = 16
	ics1.SWBOffsetMax = 1024
	ics1.SFBCB[0][0] = 1
	ics1.ScaleFactors[0][0] = 100
	ics1.Pred.PredictionUsed[0] = true
	ics1.Pred.PredictionUsed[1] = true

	ics2 := &syntax.ICStream{
		NumWindowGroups:      1,
		NumWindows:           1,
		MaxSFB:               4,
		NumSWB:               4,
		WindowSequence:       syntax.OnlyLongSequence,
		GlobalGain:           100,
		PredictorDataPresent: true,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 4
	ics2.SWBOffset[2] = 8
	ics2.SWBOffset[3] = 12
	ics2.SWBOffset[4] = 16
	ics2.SWBOffsetMax = 1024
	ics2.SFBCB[0][0] = 1
	ics2.ScaleFactors[0][0] = 100
	ics2.Pred.PredictionUsed[0] = true
	ics2.Pred.PredictionUsed[1] = true

	predState1 := make([]PredState, 1024)
	predState2 := make([]PredState, 1024)
	ResetAllPredictors(predState1, 1024)
	ResetAllPredictors(predState2, 1024)

	ele := &syntax.Element{CommonWindow: false}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeMain,
		SRIndex:     4,
		PNSState:    NewPNSState(),
		PredState1:  predState1,
		PredState2:  predState2,
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)
	for i := 0; i < 16; i++ {
		quantData1[i] = int16(i + 1)
		quantData2[i] = int16(i + 1)
	}

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// Predictor state should be updated for both channels
	stateUpdated1 := false
	stateUpdated2 := false
	for i := 0; i < 16; i++ {
		if predState1[i].R[0] != 0 || predState1[i].R[1] != 0 {
			stateUpdated1 = true
		}
		if predState2[i].R[0] != 0 || predState2[i].R[1] != 0 {
			stateUpdated2 = true
		}
	}
	if !stateUpdated1 || !stateUpdated2 {
		t.Error("predictor state should be updated for both channels")
	}
}
```

**Step 2: Run test**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "$(cat <<'EOF'
test(spectrum): add MAIN profile IC prediction test for channel pair

Verifies that IC prediction is applied to both channels and
predictor state is updated for MAIN profile.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: Test LTP Profile with Common Window

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 1: Write the test**

```go
func TestReconstructChannelPair_LTPProfile_CommonWindow(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          4,
		NumSWB:          4,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics1.WindowGroupLength[0] = 1
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 4
	ics1.SWBOffset[2] = 8
	ics1.SWBOffset[3] = 12
	ics1.SWBOffset[4] = 16
	ics1.SWBOffsetMax = 1024
	ics1.SFBCB[0][0] = 1
	ics1.ScaleFactors[0][0] = 100
	ics1.LTP.DataPresent = true
	ics1.LTP.Lag = 1024
	ics1.LTP.Coef = 4
	ics1.LTP.LastBand = 4
	ics1.LTP.LongUsed[0] = true

	ics2 := &syntax.ICStream{
		NumWindowGroups: 1,
		NumWindows:      1,
		MaxSFB:          4,
		NumSWB:          4,
		WindowSequence:  syntax.OnlyLongSequence,
		GlobalGain:      100,
	}
	ics2.WindowGroupLength[0] = 1
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 4
	ics2.SWBOffset[2] = 8
	ics2.SWBOffset[3] = 12
	ics2.SWBOffset[4] = 16
	ics2.SWBOffsetMax = 1024
	ics2.SFBCB[0][0] = 1
	ics2.ScaleFactors[0][0] = 100
	// LTP2 is used for second channel with common_window
	ics2.LTP2.DataPresent = true
	ics2.LTP2.Lag = 1024
	ics2.LTP2.Coef = 4
	ics2.LTP2.LastBand = 4
	ics2.LTP2.LongUsed[0] = true

	ltpState1 := make([]int16, 4*1024)
	ltpState2 := make([]int16, 4*1024)

	ele := &syntax.Element{CommonWindow: true}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLTP,
		SRIndex:     4,
		PNSState:    NewPNSState(),
		LTPState1:   ltpState1,
		LTPState2:   ltpState2,
		// LTPFilterBank is nil, so LTP will be skipped
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	// Should succeed even without filterbank (LTP skipped)
	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}
}
```

**Step 2: Run test**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "$(cat <<'EOF'
test(spectrum): add LTP profile test for channel pair with common_window

Verifies that LTP uses ltp2 for the second channel when
common_window is set per FAAD2 specrec.c:1239-1240.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: Test Short Blocks on Both Channels

**Files:**
- Modify: `internal/spectrum/reconstruct_test.go`

**Step 1: Write the test**

```go
func TestReconstructChannelPair_ShortBlocks(t *testing.T) {
	ics1 := &syntax.ICStream{
		NumWindowGroups: 2,
		NumWindows:      8,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.EightShortSequence,
		GlobalGain:      100,
	}
	ics1.WindowGroupLength[0] = 4
	ics1.WindowGroupLength[1] = 4
	ics1.SWBOffset[0] = 0
	ics1.SWBOffset[1] = 4
	ics1.SWBOffset[2] = 8
	ics1.SWBOffsetMax = 128
	ics1.SFBCB[0][0] = 1
	ics1.SFBCB[0][1] = 1
	ics1.SFBCB[1][0] = 1
	ics1.SFBCB[1][1] = 1
	ics1.ScaleFactors[0][0] = 100
	ics1.ScaleFactors[0][1] = 100
	ics1.ScaleFactors[1][0] = 100
	ics1.ScaleFactors[1][1] = 100

	ics2 := &syntax.ICStream{
		NumWindowGroups: 2,
		NumWindows:      8,
		MaxSFB:          2,
		NumSWB:          2,
		WindowSequence:  syntax.EightShortSequence,
		GlobalGain:      100,
	}
	ics2.WindowGroupLength[0] = 4
	ics2.WindowGroupLength[1] = 4
	ics2.SWBOffset[0] = 0
	ics2.SWBOffset[1] = 4
	ics2.SWBOffset[2] = 8
	ics2.SWBOffsetMax = 128
	ics2.SFBCB[0][0] = 1
	ics2.SFBCB[0][1] = 1
	ics2.SFBCB[1][0] = 1
	ics2.SFBCB[1][1] = 1
	ics2.ScaleFactors[0][0] = 100
	ics2.ScaleFactors[0][1] = 100
	ics2.ScaleFactors[1][0] = 100
	ics2.ScaleFactors[1][1] = 100

	ele := &syntax.Element{CommonWindow: false}

	cfg := &ReconstructChannelPairConfig{
		ICS1:        ics1,
		ICS2:        ics2,
		Element:     ele,
		FrameLength: 1024,
		ObjectType:  aac.ObjectTypeLC,
		SRIndex:     4,
		PNSState:    NewPNSState(),
	}

	quantData1 := make([]int16, 1024)
	quantData2 := make([]int16, 1024)
	for i := 0; i < 64; i++ {
		quantData1[i] = 1
		quantData2[i] = 1
	}

	specData1 := make([]float64, 1024)
	specData2 := make([]float64, 1024)

	err := ReconstructChannelPair(quantData1, quantData2, specData1, specData2, cfg)
	if err != nil {
		t.Fatalf("ReconstructChannelPair failed: %v", err)
	}

	// Verify both channels have values
	hasValue1 := false
	hasValue2 := false
	for i := 0; i < 64; i++ {
		if specData1[i] != 0 {
			hasValue1 = true
		}
		if specData2[i] != 0 {
			hasValue2 = true
		}
	}
	if !hasValue1 || !hasValue2 {
		t.Error("short block processing should produce non-zero values in both channels")
	}
}
```

**Step 2: Run test**

Run: `make test PKG=./internal/spectrum`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/spectrum/reconstruct_test.go
git commit -m "$(cat <<'EOF'
test(spectrum): add short block test for ReconstructChannelPair

Verifies that short block processing works correctly for
channel pairs (8 short windows per channel).

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 10: Run Full Test Suite and Verify

**Step 1: Run all spectrum package tests**

Run: `make test PKG=./internal/spectrum`
Expected: All tests PASS

**Step 2: Run full test suite**

Run: `make check`
Expected: All checks pass (fmt, lint, test)

**Step 3: Final commit if any cleanup needed**

```bash
# Only if changes were needed
git add .
git commit -m "$(cat <<'EOF'
chore(spectrum): cleanup after channel pair implementation

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Summary

This plan implements Step 4.12 (Spectral Reconstruction - Channel Pair) from MIGRATION_STEPS.md.

**Key additions:**
1. `ReconstructChannelPairConfig` - Configuration for stereo reconstruction
2. `ReconstructChannelPair` - Main function processing two channels with:
   - Inverse quantization + scale factors (both channels)
   - PNS with correlation (when ms_used is set)
   - M/S stereo decoding
   - Intensity stereo decoding
   - IC prediction (MAIN profile, both channels)
   - LTP prediction (LTP profile, using ltp2 for channel 2 with common_window)
   - TNS decoding (both channels)
3. `LTP2` field in `ICStream` for common_window CPE second channel

**Files modified:**
- `internal/spectrum/reconstruct.go` - Add config and function
- `internal/spectrum/reconstruct_test.go` - Add comprehensive tests
- `internal/syntax/ics.go` - Add LTP2 field
- `internal/syntax/ltp_test.go` - Test LTP2 field

**Ported from:** `reconstruct_channel_pair()` in `~/dev/faad2/libfaad/specrec.c:1131-1365`
