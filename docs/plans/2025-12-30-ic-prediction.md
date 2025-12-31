# IC Prediction (MAIN Profile) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement MPEG-2 style backward-adaptive prediction for AAC MAIN profile.

**Architecture:** IC (Intra-Channel) prediction uses a second-order linear predictor on each spectral coefficient. The predictor state is quantized to 16-bit values for efficiency. Prediction is only applied to long blocks and bands below the max prediction SFB for the sample rate.

**Tech Stack:** Pure Go, floating-point arithmetic, follows FAAD2's ic_predict.c implementation.

---

## Background

IC prediction is a coding tool used in AAC MAIN profile (not LC). It predicts spectral coefficients from previous frames using a backward-adaptive lattice predictor. The key functions are:

1. **ic_predict()** - Core prediction algorithm for one spectral bin
2. **ic_prediction()** - Applies prediction across all SFBs
3. **reset_all_predictors()** - Resets all predictor states
4. **pns_reset_pred_state()** - Resets predictors for PNS bands

The predictor state uses quantized 16-bit values to reduce memory and improve stability.

---

## Task 1: Add PredInfo Structure to Syntax Package

The syntax package needs a structure to hold MAIN profile prediction data parsed from the bitstream.

**Files:**
- Create: `internal/syntax/pred.go`
- Modify: `internal/syntax/ics.go` (add Pred field to ICStream)
- Modify: `internal/syntax/ics_info.go` (update parseMainPrediction to store data)

**Step 1.1: Write the PredInfo structure**

Create `internal/syntax/pred.go`:

```go
// internal/syntax/pred.go
package syntax

// PredInfo holds MAIN profile prediction data.
// This stores the prediction control information parsed from the bitstream.
//
// Ported from: pred_info in ~/dev/faad2/libfaad/structs.h:201-207
type PredInfo struct {
	// Limit is the maximum SFB for prediction (min of max_sfb and max_pred_sfb)
	Limit uint8

	// PredictorReset indicates if predictors should be reset
	PredictorReset bool

	// PredictorResetGroupNumber is the reset group (1-30), used with modulo 30
	PredictorResetGroupNumber uint8

	// PredictionUsed indicates which SFBs use prediction
	PredictionUsed [MaxSFB]bool
}
```

**Step 1.2: Add Pred field to ICStream**

In `internal/syntax/ics.go`, add to the ICStream struct after the LTP2 field:

```go
	// MAIN profile prediction data
	Pred PredInfo
```

**Step 1.3: Update parseMainPrediction to store data**

In `internal/syntax/ics_info.go`, replace the parseMainPrediction function:

```go
// parseMainPrediction parses MAIN profile prediction data.
// Ported from: ics_info() MAIN profile section in syntax.c:876-905
func parseMainPrediction(r *bits.Reader, ics *ICStream, sfIndex uint8) error {
	// Get max prediction SFB for this sample rate
	limit := maxPredSFB(sfIndex)
	if ics.MaxSFB < limit {
		limit = ics.MaxSFB
	}
	ics.Pred.Limit = limit

	// predictor_reset (1 bit)
	ics.Pred.PredictorReset = r.Get1Bit() != 0
	if ics.Pred.PredictorReset {
		ics.Pred.PredictorResetGroupNumber = uint8(r.GetBits(5))
	}

	// prediction_used flags for each SFB
	for sfb := uint8(0); sfb < limit; sfb++ {
		ics.Pred.PredictionUsed[sfb] = r.Get1Bit() != 0
	}

	return nil
}
```

**Step 1.4: Run tests to verify parsing still works**

Run: `go test ./internal/syntax/... -v`
Expected: All existing tests pass

**Step 1.5: Commit**

```bash
git add internal/syntax/pred.go internal/syntax/ics.go internal/syntax/ics_info.go
git commit -m "feat(syntax): add PredInfo structure for MAIN profile prediction"
```

---

## Task 2: Create Prediction Lookup Tables

The prediction algorithm uses two lookup tables for efficient division approximation.

**Files:**
- Create: `internal/spectrum/predict_tables.go`
- Create: `internal/spectrum/predict_tables_test.go`

**Step 2.1: Write the failing test**

Create `internal/spectrum/predict_tables_test.go`:

```go
package spectrum

import (
	"math"
	"testing"
)

func TestMntTableSize(t *testing.T) {
	if len(mntTable) != 128 {
		t.Errorf("mntTable length = %d, want 128", len(mntTable))
	}
}

func TestExpTableSize(t *testing.T) {
	if len(expTable) != 128 {
		t.Errorf("expTable length = %d, want 128", len(expTable))
	}
}

func TestMntTableFirstValue(t *testing.T) {
	// First value from FAAD2
	expected := float32(0.953125)
	if math.Abs(float64(mntTable[0]-expected)) > 1e-7 {
		t.Errorf("mntTable[0] = %v, want %v", mntTable[0], expected)
	}
}

func TestExpTableFirstValues(t *testing.T) {
	// First few values are powers of 0.5
	testCases := []struct {
		index    int
		expected float32
	}{
		{0, 0.5},
		{1, 0.25},
		{2, 0.125},
		{3, 0.0625},
	}
	for _, tc := range testCases {
		if math.Abs(float64(expTable[tc.index]-tc.expected)) > 1e-7 {
			t.Errorf("expTable[%d] = %v, want %v", tc.index, expTable[tc.index], tc.expected)
		}
	}
}
```

**Step 2.2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run "TestMntTable|TestExpTable" -v`
Expected: FAIL with "undefined: mntTable" and "undefined: expTable"

**Step 2.3: Write the tables**

Create `internal/spectrum/predict_tables.go`:

```go
// internal/spectrum/predict_tables.go
package spectrum

// Prediction constants.
// Ported from: ~/dev/faad2/libfaad/ic_predict.h:40-41
const (
	// predAlpha is the adaptation constant for variance/correlation update.
	predAlpha = float32(0.90625)

	// predA is the leakage factor for predictor state update.
	predA = float32(0.953125)
)

// mntTable is the mantissa lookup table for fast division approximation.
// Used to compute k1 and k2 coefficients in the predictor.
//
// Copied from: ~/dev/faad2/libfaad/ic_predict.h:49-114
var mntTable = [128]float32{
	0.9531250000, 0.9453125000, 0.9375000000, 0.9296875000,
	0.9257812500, 0.9179687500, 0.9101562500, 0.9023437500,
	0.8984375000, 0.8906250000, 0.8828125000, 0.8789062500,
	0.8710937500, 0.8671875000, 0.8593750000, 0.8515625000,
	0.8476562500, 0.8398437500, 0.8359375000, 0.8281250000,
	0.8242187500, 0.8203125000, 0.8125000000, 0.8085937500,
	0.8007812500, 0.7968750000, 0.7929687500, 0.7851562500,
	0.7812500000, 0.7773437500, 0.7734375000, 0.7656250000,
	0.7617187500, 0.7578125000, 0.7539062500, 0.7500000000,
	0.7421875000, 0.7382812500, 0.7343750000, 0.7304687500,
	0.7265625000, 0.7226562500, 0.7187500000, 0.7148437500,
	0.7109375000, 0.7070312500, 0.6992187500, 0.6953125000,
	0.6914062500, 0.6875000000, 0.6835937500, 0.6796875000,
	0.6796875000, 0.6757812500, 0.6718750000, 0.6679687500,
	0.6640625000, 0.6601562500, 0.6562500000, 0.6523437500,
	0.6484375000, 0.6445312500, 0.6406250000, 0.6406250000,
	0.6367187500, 0.6328125000, 0.6289062500, 0.6250000000,
	0.6210937500, 0.6210937500, 0.6171875000, 0.6132812500,
	0.6093750000, 0.6054687500, 0.6054687500, 0.6015625000,
	0.5976562500, 0.5937500000, 0.5937500000, 0.5898437500,
	0.5859375000, 0.5820312500, 0.5820312500, 0.5781250000,
	0.5742187500, 0.5742187500, 0.5703125000, 0.5664062500,
	0.5664062500, 0.5625000000, 0.5585937500, 0.5585937500,
	0.5546875000, 0.5507812500, 0.5507812500, 0.5468750000,
	0.5429687500, 0.5429687500, 0.5390625000, 0.5390625000,
	0.5351562500, 0.5312500000, 0.5312500000, 0.5273437500,
	0.5273437500, 0.5234375000, 0.5195312500, 0.5195312500,
	0.5156250000, 0.5156250000, 0.5117187500, 0.5117187500,
	0.5078125000, 0.5078125000, 0.5039062500, 0.5039062500,
	0.5000000000, 0.4980468750, 0.4960937500, 0.4941406250,
	0.4921875000, 0.4902343750, 0.4882812500, 0.4863281250,
	0.4843750000, 0.4824218750, 0.4804687500, 0.4785156250,
}

// expTable is the exponent lookup table for fast division approximation.
// Contains powers of 0.5: expTable[i] = 0.5^(i+1)
//
// Copied from: ~/dev/faad2/libfaad/ic_predict.h:116-245
var expTable = [128]float32{
	0.50000000000000000000000000000000,
	0.25000000000000000000000000000000,
	0.12500000000000000000000000000000,
	0.06250000000000000000000000000000,
	0.03125000000000000000000000000000,
	0.01562500000000000000000000000000,
	0.00781250000000000000000000000000,
	0.00390625000000000000000000000000,
	0.00195312500000000000000000000000,
	0.00097656250000000000000000000000,
	0.00048828125000000000000000000000,
	0.00024414062500000000000000000000,
	0.00012207031250000000000000000000,
	0.00006103515625000000000000000000,
	0.00003051757812500000000000000000,
	0.00001525878906250000000000000000,
	0.00000762939453125000000000000000,
	0.00000381469726562500000000000000,
	0.00000190734863281250000000000000,
	0.00000095367431640625000000000000,
	0.00000047683715820312500000000000,
	0.00000023841857910156250000000000,
	0.00000011920928955078125000000000,
	0.00000005960464477539062500000000,
	0.00000002980232238769531300000000,
	0.00000001490116119384765600000000,
	0.00000000745058059692382810000000,
	0.00000000372529029846191410000000,
	0.00000000186264514923095700000000,
	0.00000000093132257461547852000000,
	0.00000000046566128730773926000000,
	0.00000000023283064365386963000000,
	0.00000000011641532182693481000000,
	0.00000000005820766091346740700000,
	0.00000000002910383045673370400000,
	0.00000000001455191522836685200000,
	0.00000000000727595761418342590000,
	0.00000000000363797880709171300000,
	0.00000000000181898940354585650000,
	0.00000000000090949470177292824000,
	0.00000000000045474735088646412000,
	0.00000000000022737367544323206000,
	0.00000000000011368683772161603000,
	0.00000000000005684341886080801500,
	0.00000000000002842170943040400700,
	0.00000000000001421085471520200400,
	0.00000000000000710542735760100190,
	0.00000000000000355271367880050090,
	0.00000000000000177635683940025050,
	0.00000000000000088817841970012523,
	0.00000000000000044408920985006262,
	0.00000000000000022204460492503131,
	0.00000000000000011102230246251565,
	0.00000000000000005551115123125782700,
	0.00000000000000002775557561562891400,
	0.00000000000000001387778780781445700,
	0.00000000000000000693889390390722840,
	0.00000000000000000346944695195361420,
	0.00000000000000000173472347597680710,
	0.00000000000000000086736173798840355,
	0.00000000000000000043368086899420177,
	0.00000000000000000021684043449710089,
	0.00000000000000000010842021724855044,
	0.00000000000000000005421010862427522200,
	0.00000000000000000002710505431213761100,
	0.00000000000000000001355252715606880500,
	0.00000000000000000000677626357803440270,
	0.00000000000000000000338813178901720140,
	0.00000000000000000000169406589450860070,
	0.00000000000000000000084703294725430034,
	0.00000000000000000000042351647362715017,
	0.00000000000000000000021175823681357508,
	0.00000000000000000000010587911840678754,
	0.00000000000000000000005293955920339377100,
	0.00000000000000000000002646977960169688600,
	0.00000000000000000000001323488980084844300,
	0.00000000000000000000000661744490042422140,
	0.00000000000000000000000330872245021211070,
	0.00000000000000000000000165436122510605530,
	0.00000000000000000000000082718061255302767,
	0.00000000000000000000000041359030627651384,
	0.00000000000000000000000020679515313825692,
	0.00000000000000000000000010339757656912846,
	0.00000000000000000000000005169878828456423,
	0.00000000000000000000000002584939414228211500,
	0.00000000000000000000000001292469707114105700,
	0.00000000000000000000000000646234853557052870,
	0.00000000000000000000000000323117426778526440,
	0.00000000000000000000000000161558713389263220,
	0.00000000000000000000000000080779356694631609,
	0.00000000000000000000000000040389678347315804,
	0.00000000000000000000000000020194839173657902,
	0.00000000000000000000000000010097419586828951,
	0.00000000000000000000000000005048709793414475600,
	0.00000000000000000000000000002524354896707237800,
	0.00000000000000000000000000001262177448353618900,
	0.00000000000000000000000000000631088724176809440,
	0.00000000000000000000000000000315544362088404720,
	0.00000000000000000000000000000157772181044202360,
	0.00000000000000000000000000000078886090522101181,
	0.00000000000000000000000000000039443045261050590,
	0.00000000000000000000000000000019721522630525295,
	0.00000000000000000000000000000009860761315262647600,
	0.00000000000000000000000000000004930380657631323800,
	0.00000000000000000000000000000002465190328815661900,
	0.00000000000000000000000000000001232595164407830900,
	0.00000000000000000000000000000000616297582203915470,
	0.00000000000000000000000000000000308148791101957740,
	0.00000000000000000000000000000000154074395550978870,
	0.00000000000000000000000000000000077037197775489434,
	0.00000000000000000000000000000000038518598887744717,
	0.00000000000000000000000000000000019259299443872359,
	0.00000000000000000000000000000000009629649721936179,
	0.00000000000000000000000000000000004814824860968090,
	0.00000000000000000000000000000000002407412430484045,
	0.00000000000000000000000000000000001203706215242022,
	0.00000000000000000000000000000000000601853107621011,
	0.00000000000000000000000000000000000300926553810506,
	0.00000000000000000000000000000000000150463276905253,
	0.00000000000000000000000000000000000075231638452626,
	0.00000000000000000000000000000000000037615819226313,
	0.00000000000000000000000000000000000018807909613157,
	0.00000000000000000000000000000000000009403954806578,
	0.00000000000000000000000000000000000004701977403289,
	0.00000000000000000000000000000000000002350988701645,
	0.00000000000000000000000000000000000001175494350822,
	0.0, // Underflow
	0.0, // Underflow
}
```

**Step 2.4: Run tests to verify they pass**

Run: `go test ./internal/spectrum -run "TestMntTable|TestExpTable" -v`
Expected: PASS

**Step 2.5: Commit**

```bash
git add internal/spectrum/predict_tables.go internal/spectrum/predict_tables_test.go
git commit -m "feat(spectrum): add prediction lookup tables for MAIN profile"
```

---

## Task 3: Create PredState Structure and Reset Functions

The predictor state structure holds quantized predictor coefficients.

**Files:**
- Create: `internal/spectrum/predict.go`
- Create: `internal/spectrum/predict_test.go`

**Step 3.1: Write the failing test for PredState**

Create `internal/spectrum/predict_test.go`:

```go
package spectrum

import (
	"testing"
)

func TestNewPredState(t *testing.T) {
	state := NewPredState()
	// After reset, VAR should be 0x3F80 (1.0 in quantized form)
	if state.VAR[0] != 0x3F80 {
		t.Errorf("VAR[0] = %#x, want 0x3F80", state.VAR[0])
	}
	if state.VAR[1] != 0x3F80 {
		t.Errorf("VAR[1] = %#x, want 0x3F80", state.VAR[1])
	}
	// R and COR should be zero
	if state.R[0] != 0 || state.R[1] != 0 {
		t.Errorf("R = %v, want [0, 0]", state.R)
	}
	if state.COR[0] != 0 || state.COR[1] != 0 {
		t.Errorf("COR = %v, want [0, 0]", state.COR)
	}
}

func TestResetPredState(t *testing.T) {
	state := &PredState{
		R:   [2]int16{100, 200},
		COR: [2]int16{300, 400},
		VAR: [2]int16{500, 600},
	}
	ResetPredState(state)

	if state.R[0] != 0 || state.R[1] != 0 {
		t.Errorf("after reset, R = %v, want [0, 0]", state.R)
	}
	if state.COR[0] != 0 || state.COR[1] != 0 {
		t.Errorf("after reset, COR = %v, want [0, 0]", state.COR)
	}
	if state.VAR[0] != 0x3F80 || state.VAR[1] != 0x3F80 {
		t.Errorf("after reset, VAR = %v, want [0x3F80, 0x3F80]", state.VAR)
	}
}

func TestResetAllPredictors(t *testing.T) {
	frameLen := uint16(1024)
	states := make([]PredState, frameLen)

	// Set some non-zero values
	for i := range states {
		states[i].R[0] = int16(i)
		states[i].VAR[0] = int16(i + 100)
	}

	ResetAllPredictors(states, frameLen)

	// Check all are reset
	for i := uint16(0); i < frameLen; i++ {
		if states[i].R[0] != 0 {
			t.Errorf("states[%d].R[0] = %d, want 0", i, states[i].R[0])
			break
		}
		if states[i].VAR[0] != 0x3F80 {
			t.Errorf("states[%d].VAR[0] = %#x, want 0x3F80", i, states[i].VAR[0])
			break
		}
	}
}
```

**Step 3.2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run "TestNewPredState|TestResetPredState|TestResetAllPredictors" -v`
Expected: FAIL with "undefined: PredState"

**Step 3.3: Write the PredState structure and reset functions**

Create `internal/spectrum/predict.go`:

```go
// internal/spectrum/predict.go
package spectrum

import (
	"math"

	"github.com/llehouerou/go-aac/internal/syntax"
	"github.com/llehouerou/go-aac/internal/tables"
)

// PredState holds the state for one spectral coefficient's predictor.
// The values are quantized to 16-bit for memory efficiency and stability.
//
// Ported from: pred_state in ~/dev/faad2/libfaad/structs.h:51-55
type PredState struct {
	R   [2]int16 // Predictor state (past output)
	COR [2]int16 // Correlation accumulators
	VAR [2]int16 // Variance accumulators
}

// NewPredState creates a new predictor state with initial values.
func NewPredState() *PredState {
	s := &PredState{}
	ResetPredState(s)
	return s
}

// ResetPredState resets a single predictor state to initial values.
// After reset, the predictor will output zero prediction.
//
// Ported from: reset_pred_state() in ~/dev/faad2/libfaad/ic_predict.c:198-206
func ResetPredState(state *PredState) {
	state.R[0] = 0
	state.R[1] = 0
	state.COR[0] = 0
	state.COR[1] = 0
	state.VAR[0] = 0x3F80 // 1.0 in quantized form
	state.VAR[1] = 0x3F80 // 1.0 in quantized form
}

// ResetAllPredictors resets all predictor states in the array.
//
// Ported from: reset_all_predictors() in ~/dev/faad2/libfaad/ic_predict.c:236-241
func ResetAllPredictors(states []PredState, frameLen uint16) {
	for i := uint16(0); i < frameLen && int(i) < len(states); i++ {
		ResetPredState(&states[i])
	}
}
```

**Step 3.4: Run tests to verify they pass**

Run: `go test ./internal/spectrum -run "TestNewPredState|TestResetPredState|TestResetAllPredictors" -v`
Expected: PASS

**Step 3.5: Commit**

```bash
git add internal/spectrum/predict.go internal/spectrum/predict_test.go
git commit -m "feat(spectrum): add PredState structure and reset functions"
```

---

## Task 4: Implement Quantization Helper Functions

The predictor uses 16-bit quantized values stored as the upper 16 bits of a float32.

**Files:**
- Modify: `internal/spectrum/predict.go` (add functions)
- Modify: `internal/spectrum/predict_test.go` (add tests)

**Step 4.1: Write the failing test**

Add to `internal/spectrum/predict_test.go`:

```go
func TestQuantPred(t *testing.T) {
	testCases := []struct {
		input    float32
		expected int16
	}{
		{0.0, 0},
		{1.0, 0x3F80},       // IEEE 754: 0x3F800000
		{-1.0, -16512},      // IEEE 754: 0xBF800000 -> 0xBF80 as int16
		{0.5, 0x3F00},       // IEEE 754: 0x3F000000
	}
	for _, tc := range testCases {
		got := quantPred(tc.input)
		if got != tc.expected {
			t.Errorf("quantPred(%v) = %#x, want %#x", tc.input, got, tc.expected)
		}
	}
}

func TestInvQuantPred(t *testing.T) {
	testCases := []struct {
		input    int16
		expected float32
	}{
		{0, 0.0},
		{0x3F80, 1.0},
		{0x3F00, 0.5},
	}
	for _, tc := range testCases {
		got := invQuantPred(tc.input)
		if math.Abs(float64(got-tc.expected)) > 1e-6 {
			t.Errorf("invQuantPred(%#x) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestFltRound(t *testing.T) {
	// Test that fltRound rounds to 16-bit precision
	testCases := []struct {
		input    float32
		expected float32
	}{
		{1.0, 1.0},           // Already aligned
		{0.5, 0.5},           // Already aligned
		{1.0000001, 1.0},     // Should round
	}
	for _, tc := range testCases {
		got := fltRound(tc.input)
		// Check that result has same upper 16 bits as expected
		gotQ := quantPred(got)
		expQ := quantPred(tc.expected)
		if gotQ != expQ {
			t.Errorf("fltRound(%v): quantized = %#x, want %#x", tc.input, gotQ, expQ)
		}
	}
}
```

**Step 4.2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run "TestQuantPred|TestInvQuantPred|TestFltRound" -v`
Expected: FAIL with "undefined: quantPred"

**Step 4.3: Implement the quantization functions**

Add to `internal/spectrum/predict.go`:

```go
// floatToBits converts a float32 to its IEEE 754 bit representation.
func floatToBits(f float32) uint32 {
	return math.Float32bits(f)
}

// bitsToFloat converts IEEE 754 bits to a float32.
func bitsToFloat(u uint32) float32 {
	return math.Float32frombits(u)
}

// fltRound rounds a float32 to 16-bit mantissa precision.
// This matches FAAD2's flt_round() which rounds 0.5 LSB toward infinity.
//
// Ported from: flt_round() in ~/dev/faad2/libfaad/ic_predict.c:53-74
func fltRound(pf float32) float32 {
	tmp := floatToBits(pf)
	flg := tmp & 0x00008000

	tmp &= 0xffff0000
	tmp1 := tmp

	// Round 0.5 LSB toward infinity
	if flg != 0 {
		tmp &= 0xff800000        // Extract exponent and sign
		tmp |= 0x00010000        // Insert 1 LSB
		tmp2 := tmp              // Add 1 LSB and elided one
		tmp &= 0xff800000        // Extract exponent and sign

		return bitsToFloat(tmp1) + bitsToFloat(tmp2) - bitsToFloat(tmp)
	}
	return bitsToFloat(tmp)
}

// quantPred quantizes a float32 to 16-bit by taking the upper 16 bits.
//
// Ported from: quant_pred() in ~/dev/faad2/libfaad/ic_predict.c:76-79
func quantPred(x float32) int16 {
	return int16(floatToBits(x) >> 16)
}

// invQuantPred dequantizes a 16-bit value back to float32.
//
// Ported from: inv_quant_pred() in ~/dev/faad2/libfaad/ic_predict.c:81-85
func invQuantPred(q int16) float32 {
	u16 := uint16(q)
	return bitsToFloat(uint32(u16) << 16)
}
```

**Step 4.4: Run tests to verify they pass**

Run: `go test ./internal/spectrum -run "TestQuantPred|TestInvQuantPred|TestFltRound" -v`
Expected: PASS

**Step 4.5: Commit**

```bash
git add internal/spectrum/predict.go internal/spectrum/predict_test.go
git commit -m "feat(spectrum): add quantization functions for IC prediction"
```

---

## Task 5: Implement Core IC Predict Function

The core prediction function applies the backward-adaptive predictor to a single spectral coefficient.

**Files:**
- Modify: `internal/spectrum/predict.go`
- Modify: `internal/spectrum/predict_test.go`

**Step 5.1: Write the failing test**

Add to `internal/spectrum/predict_test.go`:

```go
func TestICPredict_NoPrediction(t *testing.T) {
	// When pred=false, output should equal input and state should still update
	state := NewPredState()
	input := float32(0.5)

	output := icPredict(state, input, false)

	// Output should be unchanged (no prediction applied)
	if output != input {
		t.Errorf("icPredict with pred=false: output = %v, want %v", output, input)
	}
}

func TestICPredict_WithPrediction(t *testing.T) {
	// When pred=true, output should be input + predicted value
	state := NewPredState()

	// First sample: no prediction yet (k1, k2 = 0)
	output1 := icPredict(state, 1.0, true)
	// With fresh state, prediction should be 0, so output = input
	if math.Abs(float64(output1-1.0)) > 0.001 {
		t.Errorf("first sample: output = %v, want ~1.0", output1)
	}

	// Second sample: state has been updated, prediction should be non-zero
	output2 := icPredict(state, 1.0, true)
	// After one update, there should be some prediction
	// The exact value depends on the predictor coefficients
	if output2 == 1.0 {
		t.Logf("second sample: output = %v (prediction may be small)", output2)
	}
}

func TestICPredict_StateUpdate(t *testing.T) {
	// Verify that state is updated after each call
	state := NewPredState()

	_ = icPredict(state, 1.0, true)

	// State should have been updated
	if state.R[0] == 0 && state.R[1] == 0 {
		t.Error("state.R was not updated")
	}
}
```

**Step 5.2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run "TestICPredict" -v`
Expected: FAIL with "undefined: icPredict"

**Step 5.3: Implement icPredict**

Add to `internal/spectrum/predict.go`:

```go
// icPredict applies backward-adaptive prediction to one spectral coefficient.
// If pred is true, the prediction is added to the input.
// The state is always updated regardless of pred.
//
// Ported from: ic_predict() in ~/dev/faad2/libfaad/ic_predict.c:87-196
func icPredict(state *PredState, input float32, pred bool) float32 {
	// Dequantize state
	r0 := invQuantPred(state.R[0])
	r1 := invQuantPred(state.R[1])
	cor0 := invQuantPred(state.COR[0])
	cor1 := invQuantPred(state.COR[1])
	var0 := invQuantPred(state.VAR[0])
	var1 := invQuantPred(state.VAR[1])

	// Calculate k1 coefficient using table lookup
	var k1 float32
	tmp := uint16(state.VAR[0])
	j := int(tmp >> 7)
	i := int(tmp & 0x7f)
	if j >= 128 {
		j -= 128
		k1 = cor0 * expTable[j] * mntTable[i]
	} else {
		k1 = 0
	}

	var output float32
	if pred {
		// Calculate k2 coefficient
		var k2 float32
		tmp = uint16(state.VAR[1])
		j = int(tmp >> 7)
		i = int(tmp & 0x7f)
		if j >= 128 {
			j -= 128
			k2 = cor1 * expTable[j] * mntTable[i]
		} else {
			k2 = 0
		}

		// Calculate predicted value
		predictedValue := k1*r0 + k2*r1
		predictedValue = fltRound(predictedValue)
		output = input + predictedValue
	} else {
		output = input
	}

	// Calculate new state data
	e0 := output
	e1 := e0 - k1*r0
	dr1 := k1 * e0

	// Update variance and correlation
	var0 = predAlpha*var0 + 0.5*(r0*r0+e0*e0)
	cor0 = predAlpha*cor0 + r0*e0
	var1 = predAlpha*var1 + 0.5*(r1*r1+e1*e1)
	cor1 = predAlpha*cor1 + r1*e1

	// Update predictor state
	r1 = predA * (r0 - dr1)
	r0 = predA * e0

	// Quantize and store state
	state.R[0] = quantPred(r0)
	state.R[1] = quantPred(r1)
	state.COR[0] = quantPred(cor0)
	state.COR[1] = quantPred(cor1)
	state.VAR[0] = quantPred(var0)
	state.VAR[1] = quantPred(var1)

	return output
}
```

**Step 5.4: Run tests to verify they pass**

Run: `go test ./internal/spectrum -run "TestICPredict" -v`
Expected: PASS

**Step 5.5: Commit**

```bash
git add internal/spectrum/predict.go internal/spectrum/predict_test.go
git commit -m "feat(spectrum): implement icPredict core function"
```

---

## Task 6: Implement ICPrediction Function

This is the main entry point that applies prediction across all SFBs.

**Files:**
- Modify: `internal/spectrum/predict.go`
- Modify: `internal/spectrum/predict_test.go`

**Step 6.1: Write the failing test**

Add to `internal/spectrum/predict_test.go`:

```go
func TestICPrediction_ShortSequence(t *testing.T) {
	// For short sequences, all predictors should be reset
	frameLen := uint16(1024)
	states := make([]PredState, frameLen)

	// Set non-zero values
	for i := range states {
		states[i].R[0] = 100
	}

	ics := &syntax.ICStream{
		WindowSequence: syntax.EightShortSequence,
	}
	spec := make([]float32, frameLen)

	ICPrediction(ics, spec, states, frameLen, 3) // sfIndex=3 (48kHz)

	// All states should be reset
	for i := uint16(0); i < frameLen; i++ {
		if states[i].R[0] != 0 {
			t.Errorf("states[%d].R[0] = %d, want 0 (should be reset)", i, states[i].R[0])
			break
		}
	}
}

func TestICPrediction_LongSequence(t *testing.T) {
	// For long sequences, prediction should be applied
	frameLen := uint16(1024)
	states := make([]PredState, frameLen)
	for i := range states {
		ResetPredState(&states[i])
	}

	ics := &syntax.ICStream{
		WindowSequence:       syntax.OnlyLongSequence,
		MaxSFB:               10,
		PredictorDataPresent: true,
	}
	ics.Pred.Limit = 10
	for i := uint8(0); i < 10; i++ {
		ics.Pred.PredictionUsed[i] = true
	}

	// Set up SWB offsets (simplified)
	for i := 0; i <= 10; i++ {
		ics.SWBOffset[i] = uint16(i * 10)
	}
	ics.SWBOffsetMax = 100

	spec := make([]float32, frameLen)
	for i := range spec {
		spec[i] = 1.0
	}

	ICPrediction(ics, spec, states, frameLen, 3)

	// After prediction with fresh states, spec should be mostly unchanged
	// (prediction is zero for fresh states)
	// This is a basic sanity check
	if spec[0] != 1.0 {
		t.Logf("spec[0] = %v after prediction (expected ~1.0 for fresh state)", spec[0])
	}
}
```

**Step 6.2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run "TestICPrediction" -v`
Expected: FAIL with "undefined: ICPrediction"

**Step 6.3: Implement ICPrediction**

Add to `internal/spectrum/predict.go`:

```go
// ICPrediction applies intra-channel prediction to the spectral coefficients.
// For short sequences, all predictors are reset.
// For long sequences, prediction is applied per SFB based on prediction_used flags.
//
// Ported from: ic_prediction() in ~/dev/faad2/libfaad/ic_predict.c:245-279
func ICPrediction(ics *syntax.ICStream, spec []float32, states []PredState, frameLen uint16, sfIndex uint8) {
	if ics.WindowSequence == syntax.EightShortSequence {
		// Short sequence: reset all predictors
		ResetAllPredictors(states, frameLen)
		return
	}

	// Long sequence: apply prediction per SFB
	maxPredSfb := tables.MaxPredSFB(sfIndex)

	for sfb := uint8(0); sfb < maxPredSfb; sfb++ {
		low := ics.SWBOffset[sfb]
		high := ics.SWBOffset[sfb+1]
		if high > ics.SWBOffsetMax {
			high = ics.SWBOffsetMax
		}

		// Determine if prediction is used for this SFB
		usePred := ics.PredictorDataPresent && ics.Pred.PredictionUsed[sfb]

		for bin := low; bin < high && int(bin) < len(spec) && int(bin) < len(states); bin++ {
			spec[bin] = icPredict(&states[bin], spec[bin], usePred)
		}
	}

	// Handle predictor reset groups
	if ics.PredictorDataPresent && ics.Pred.PredictorReset {
		resetGroup := ics.Pred.PredictorResetGroupNumber
		if resetGroup > 0 {
			// Reset every 30th predictor starting from (resetGroup - 1)
			for bin := uint16(resetGroup - 1); bin < frameLen && int(bin) < len(states); bin += 30 {
				ResetPredState(&states[bin])
			}
		}
	}
}
```

**Step 6.4: Run tests to verify they pass**

Run: `go test ./internal/spectrum -run "TestICPrediction" -v`
Expected: PASS

**Step 6.5: Commit**

```bash
git add internal/spectrum/predict.go internal/spectrum/predict_test.go
git commit -m "feat(spectrum): implement ICPrediction main function"
```

---

## Task 7: Implement PNSResetPredState Function

This function resets predictor states for bands that use PNS coding.

**Files:**
- Modify: `internal/spectrum/predict.go`
- Modify: `internal/spectrum/predict_test.go`

**Step 7.1: Write the failing test**

Add to `internal/spectrum/predict_test.go`:

```go
func TestPNSResetPredState(t *testing.T) {
	frameLen := uint16(1024)
	states := make([]PredState, frameLen)

	// Set non-zero values
	for i := range states {
		states[i].R[0] = 100
	}

	ics := &syntax.ICStream{
		WindowSequence:    syntax.OnlyLongSequence,
		NumWindowGroups:   1,
		WindowGroupLength: [8]uint8{1},
		MaxSFB:            5,
		SWBOffsetMax:      100,
	}
	// Set up SWB offsets
	for i := 0; i <= 5; i++ {
		ics.SWBOffset[i] = uint16(i * 20)
	}
	// Set SFB 2 to use noise codebook
	ics.SFBCB[0][2] = uint8(huffman.NoiseHCB)

	PNSResetPredState(ics, states)

	// States in SFB 2 (bins 40-59) should be reset
	for bin := 40; bin < 60; bin++ {
		if states[bin].R[0] != 0 {
			t.Errorf("states[%d].R[0] = %d, want 0 (noise band)", bin, states[bin].R[0])
		}
	}

	// States in other SFBs should not be reset
	if states[0].R[0] != 100 {
		t.Errorf("states[0].R[0] = %d, want 100 (non-noise band)", states[0].R[0])
	}
}

func TestPNSResetPredState_ShortSequence(t *testing.T) {
	// Short sequences should return early without doing anything
	states := make([]PredState, 1024)
	for i := range states {
		states[i].R[0] = 100
	}

	ics := &syntax.ICStream{
		WindowSequence: syntax.EightShortSequence,
	}

	PNSResetPredState(ics, states)

	// States should be unchanged
	if states[0].R[0] != 100 {
		t.Errorf("states[0].R[0] = %d, want 100 (short sequence, no reset)", states[0].R[0])
	}
}
```

**Step 7.2: Run test to verify it fails**

Run: `go test ./internal/spectrum -run "TestPNSResetPredState" -v`
Expected: FAIL with "undefined: PNSResetPredState"

**Step 7.3: Implement PNSResetPredState**

Add to `internal/spectrum/predict.go`:

```go
// PNSResetPredState resets predictor states for bands that use PNS (noise) coding.
// This is called after PNS decoding to prevent prediction from affecting noise bands.
// Only applies to long blocks.
//
// Ported from: pns_reset_pred_state() in ~/dev/faad2/libfaad/ic_predict.c:208-234
func PNSResetPredState(ics *syntax.ICStream, states []PredState) {
	// Prediction only for long blocks
	if ics.WindowSequence == syntax.EightShortSequence {
		return
	}

	for g := uint8(0); g < ics.NumWindowGroups; g++ {
		for b := uint8(0); b < ics.WindowGroupLength[g]; b++ {
			for sfb := uint8(0); sfb < ics.MaxSFB; sfb++ {
				if IsNoiseICS(ics, g, sfb) {
					offs := ics.SWBOffset[sfb]
					offs2 := ics.SWBOffset[sfb+1]
					if offs2 > ics.SWBOffsetMax {
						offs2 = ics.SWBOffsetMax
					}

					for i := offs; i < offs2 && int(i) < len(states); i++ {
						ResetPredState(&states[i])
					}
				}
			}
		}
	}
}
```

**Step 7.4: Add missing import**

Make sure the import for huffman is present at the top of the test file:

```go
import (
	"math"
	"testing"

	"github.com/llehouerou/go-aac/internal/huffman"
	"github.com/llehouerou/go-aac/internal/syntax"
)
```

**Step 7.5: Run tests to verify they pass**

Run: `go test ./internal/spectrum -run "TestPNSResetPredState" -v`
Expected: PASS

**Step 7.6: Commit**

```bash
git add internal/spectrum/predict.go internal/spectrum/predict_test.go
git commit -m "feat(spectrum): implement PNSResetPredState for noise bands"
```

---

## Task 8: Run Full Test Suite and Final Verification

Verify all tests pass and the implementation integrates correctly.

**Step 8.1: Run all spectrum tests**

Run: `go test ./internal/spectrum/... -v`
Expected: All tests pass

**Step 8.2: Run all syntax tests**

Run: `go test ./internal/syntax/... -v`
Expected: All tests pass

**Step 8.3: Run full test suite**

Run: `make check`
Expected: All checks pass (fmt, lint, test)

**Step 8.4: Create final commit if needed**

If any cleanup was needed:

```bash
git add -A
git commit -m "chore: cleanup and finalize IC prediction implementation"
```

---

## Summary

**Files Created:**
- `internal/syntax/pred.go` - PredInfo structure
- `internal/spectrum/predict.go` - Main prediction implementation
- `internal/spectrum/predict_tables.go` - Lookup tables
- `internal/spectrum/predict_test.go` - Tests
- `internal/spectrum/predict_tables_test.go` - Table tests

**Files Modified:**
- `internal/syntax/ics.go` - Added Pred field to ICStream
- `internal/syntax/ics_info.go` - Updated parseMainPrediction to store data

**Functions Implemented:**
- `PredState` structure and `NewPredState()`
- `ResetPredState(state)` - Reset single predictor
- `ResetAllPredictors(states, frameLen)` - Reset all predictors
- `ICPrediction(ics, spec, states, frameLen, sfIndex)` - Main prediction entry point
- `PNSResetPredState(ics, states)` - Reset predictors for noise bands
- `icPredict(state, input, pred)` - Core prediction function (internal)
- `quantPred(x)`, `invQuantPred(q)`, `fltRound(pf)` - Quantization helpers (internal)

**Tables Added:**
- `mntTable[128]` - Mantissa lookup table
- `expTable[128]` - Exponent lookup table (powers of 0.5)
- `predAlpha`, `predA` - Prediction constants
