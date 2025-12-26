# FAAD2 to Go Migration Steps

This document details the complete migration path from FAAD2 (C) to a pure Go AAC decoder.
Each step is designed to be independently testable and incrementally buildable.

## Overview

**Source**: FAAD2 (`~/dev/faad2/libfaad/`) - ~60,000 lines of C code
**Target**: Pure Go AAC decoder with no CGO dependencies

---

## Testing Strategy (MANDATORY)

**Every step MUST be validated against FAAD2 reference output before moving to the next step.**

### The FAAD2 Debug Tool

We have a custom debug tool (`scripts/faad2_debug`) that decodes AAC files using FAAD2 and dumps intermediate values. This is the authoritative reference for testing.

```bash
# Build and test the debug tool
cd scripts && make test

# Generate reference data for any AAC file
./check_faad2 input.aac
# Output: /tmp/faad2_ref_<name>/
```

### Reference Data Format

Per-frame output files:
| File | Contents | Format |
|------|----------|--------|
| `frame_NNNN_adts.bin` | Parsed ADTS header | 16 bytes (see below) |
| `frame_NNNN_pcm.bin` | Final PCM output | int16 LE, interleaved |
| `info.json` | Stream metadata | JSON |

ADTS header binary format (16 bytes):
```
[0-1]  syncword (0x0FFF)
[2]    id (MPEG-2/4)
[3]    layer
[4]    protection_absent
[5]    profile
[6]    sf_index (sample rate index)
[7]    private_bit
[8]    channel_config
[9]    original
[10]   home
[11-12] frame_length (big-endian)
[13-14] buffer_fullness (big-endian)
[15]   num_raw_blocks
```

### Testing Requirements Per Step

Each implementation step has three testing levels:

1. **Unit Tests**: Test the Go code in isolation with known inputs
2. **FAAD2 Comparison**: Compare output against `faad2_debug` reference data
3. **Integration Tests**: Test the component works with upstream/downstream components

### Test File Generation

```bash
# Generate comprehensive test files
go run testdata/generate.go

# Quick test file with FFmpeg
ffmpeg -f lavfi -i "sine=frequency=1000:duration=0.5" -c:a aac -b:a 128k test.aac
```

### Acceptance Criteria Template

Every step's acceptance criteria MUST include:
- [ ] Unit tests pass (`make test PKG=./internal/<package>`)
- [ ] Output matches FAAD2 reference for test files
- [ ] No regressions in existing tests (`make check`)

---

## Code Organization Rules (MANDATORY)

These rules MUST be followed during migration. Do NOT replicate FAAD2's flat C structure.

### Rule 1: Use Thematic Packages

Organize code into logical packages by function, not by source file:

```
go-aac/
├── aac.go                    # Public API only
├── decoder.go                # Main decoder implementation
├── internal/
│   ├── bits/                 # Bitstream reading
│   ├── huffman/              # Huffman decoding
│   ├── syntax/               # Bitstream syntax parsing
│   ├── spectrum/             # Spectral processing (requant, TNS, PNS, etc.)
│   ├── filterbank/           # Filter bank (IMDCT + windowing)
│   ├── mdct/                 # MDCT implementation
│   ├── fft/                  # FFT implementation
│   ├── tables/               # Lookup tables
│   ├── output/               # PCM output conversion
│   ├── sbr/                  # HE-AAC (SBR) - optional
│   └── ps/                   # HE-AACv2 (PS) - optional
└── testdata/                 # Test files
```

### Rule 2: Maximum ~300 Lines Per File

Split large logical units into multiple files:

| C Source | Go Files |
|----------|----------|
| `codebook/hcb_*.h` (12 files) | `huffman/codebook_1.go`, `codebook_2.go`, ... `codebook_sf.go` |
| `syntax.c` (2700 lines) | `syntax/adts.go`, `adif.go`, `pce.go`, `ics.go`, `sce.go`, `cpe.go`, `fill.go`, `raw_data_block.go` |
| `specrec.c` (1400 lines) | `spectrum/requant.go`, `scalefac.go`, `window.go`, `reconstruct.go` |
| `iq_table.h` (16000 lines) | `tables/iq_table.go` (generated, exception to size rule) |
| `kbd_win.h` + `sine_win.h` | `filterbank/window_sine.go`, `window_kbd.go` |

### Rule 3: One Concept Per File

Each file should have a single responsibility:

```
internal/huffman/
├── codebook.go        # Codebook types and lookup logic
├── codebook_1.go      # HCB_1 table data only
├── codebook_2.go      # HCB_2 table data only
├── ...
├── codebook_11.go     # HCB_11 table data only
├── codebook_sf.go     # Scale factor codebook data only
├── decoder.go         # Decoding functions
└── decoder_test.go    # Tests
```

```
internal/syntax/
├── types.go           # Shared types (ic_stream, element, etc.)
├── adts.go            # ADTS header parsing
├── adif.go            # ADIF header parsing
├── pce.go             # Program Config Element
├── ics.go             # Individual Channel Stream
├── section.go         # Section data parsing
├── scalefactor.go     # Scale factor parsing
├── spectral.go        # Spectral data parsing
├── sce.go             # Single Channel Element
├── cpe.go             # Channel Pair Element
├── cce.go             # Coupling Channel Element
├── fill.go            # Fill elements and extensions
├── raw_data_block.go  # Main entry point
└── *_test.go          # Tests for each
```

### Rule 4: Source Reference Comments

Add comments linking Go code to FAAD2 source:

```go
// Package bits implements AAC bitstream reading.
//
// Ported from: ~/dev/faad2/libfaad/bits.c, bits.h
package bits

// Reader reads bits from a byte buffer.
//
// Ported from: bitfile struct in bits.h:48-60
type Reader struct {
    buffer    []byte
    pos       int
    bitsLeft  uint
    // ...
}

// GetBits reads n bits from the stream.
//
// Ported from: faad_getbits() in bits.h:130-146
func (r *Reader) GetBits(n uint) uint32 {
    // ...
}
```

### Rule 5: Package Dependency Graph

Maintain clean dependencies (no cycles):

```
aac (public API)
 └── decoder
      ├── syntax ──────┬── bits
      │                └── huffman
      ├── spectrum ──── tables
      ├── filterbank ── mdct ── fft
      └── output
```

### Rule 6: Generated Tables

For large lookup tables (IQ table, window coefficients):
1. Create a generator in `scripts/generate_*.go`
2. Generate the Go file with `//go:generate` directive
3. Tables are the only exception to the 300-line rule
4. Add a header comment: `// Code generated by generate_*.go. DO NOT EDIT.`

---

### Architecture Summary

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              AAC Decoder Pipeline                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐              │
│  │ Bitstream│───▶│ Huffman  │───▶│ Spectral │───▶│  Filter  │───▶ PCM     │
│  │  Reader  │    │ Decoding │    │  Recon   │    │   Bank   │              │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘              │
│       │               │               │               │                     │
│    bits.c         huffman.c       specrec.c      filtbank.c                │
│                                       │            mdct.c                   │
│                                       │            cfft.c                   │
│                              ┌────────┴────────┐                           │
│                              │                 │                            │
│                           ┌──┴──┐          ┌───┴───┐                       │
│                           │ TNS │          │  PNS  │                       │
│                           └─────┘          └───────┘                       │
│                           tns.c            pns.c                           │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Foundation Layer (No Dependencies)

### Step 1.1: Project Structure and Build System
**Source**: N/A (new)
**Lines**: ~50
**Files to create**:
- `go.mod` - Module definition
- `Makefile` - Build, test, lint commands
- `internal/` - Internal packages

**Acceptance criteria**:
- `make check` passes (fmt, lint, test)
- Project compiles with `go build ./...`

---

### Step 1.2: Constants and Types (`common.go`)
**Source**: `common.h` (437 lines), `neaacdec.h` (200 lines)
**Lines**: ~200
**Files to create**:
- `aac.go` - Public types and constants
- `internal/types/types.go` - Internal type definitions

**Port these definitions**:
```c
// Object types (common.h, neaacdec.h)
#define MAIN       1
#define LC         2
#define SSR        3
#define LTP        4
#define HE_AAC     5
#define LD        23
#define ER_LC     17
#define ER_LTP    19
#define DRM_ER_LC 27

// Header types
#define RAW        0
#define ADIF       1
#define ADTS       2
#define LATM       3

// Output formats
#define FAAD_FMT_16BIT  1
#define FAAD_FMT_24BIT  2
#define FAAD_FMT_32BIT  3
#define FAAD_FMT_FLOAT  4
#define FAAD_FMT_DOUBLE 5

// Channel positions
#define FRONT_CHANNEL_CENTER (1)
#define FRONT_CHANNEL_LEFT   (2)
// ... etc
```

**Acceptance criteria**:
- All AAC object types defined as Go constants
- All header types defined
- All output format types defined
- Channel position constants defined

---

### Step 1.3: Error Definitions (`error.go`)
**Source**: `error.c` (70 lines), `error.h` (44 lines)
**Lines**: ~100
**Files to create**:
- `internal/errors/errors.go`

**Port these error messages**:
```c
char *err_msg[] = {
    "No error",
    "Gain control not yet implemented",
    "Pulse coding not allowed in short blocks",
    // ... 34 total error messages
};
```

**Acceptance criteria**:
- All 34 error codes defined
- Error type implements `error` interface
- `Error()` method returns descriptive message

---

### Step 1.4: Bit Reader (`bits.go`)
**Source**: `bits.c` (292 lines), `bits.h` (422 lines)
**Lines**: ~300
**Files to create**:
- `internal/bits/reader.go`
- `internal/bits/reader_test.go`

**Key functions to port**:
```c
void faad_initbits(bitfile *ld, const void *buffer, const uint32_t buffer_size);
void faad_endbits(bitfile *ld);
uint8_t faad_byte_align(bitfile *ld);
uint32_t faad_get_processed_bits(bitfile *ld);
void faad_flushbits(bitfile *ld, uint32_t bits);
uint32_t faad_showbits(bitfile *ld, uint32_t bits);
uint32_t faad_getbits(bitfile *ld, uint32_t n);
uint8_t faad_get1bit(bitfile *ld);
uint8_t *faad_getbitbuffer(bitfile *ld, uint32_t bits);
```

**Data structure**:
```c
typedef struct _bitfile {
    const void *buffer;
    uint32_t *tail;
    uint32_t *start;
    uint32_t bufa;
    uint32_t bufb;
    uint32_t bits_left;
    uint32_t buffer_size;
    uint32_t bytes_left;
    uint8_t error;
} bitfile;
```

**Acceptance criteria**:
- Can read arbitrary number of bits (1-32)
- Can peek bits without consuming
- Proper byte alignment
- Error handling for buffer overrun
- Unit tests with known bit patterns
- **Validation**: Read first 56 bits of ADTS frame and verify against `frame_NNNN_adts.bin`

**Test procedure**:
```go
func TestBitReader_ADTSHeader(t *testing.T) {
    // Load real AAC file
    data, _ := os.ReadFile("testdata/test.aac")
    r := bits.NewReader(data)

    // Read ADTS header fields (matches FAAD2's adts_frame parsing)
    syncword := r.GetBits(12)       // Should be 0xFFF
    id := r.GetBits(1)
    layer := r.GetBits(2)
    protection := r.GetBits(1)
    profile := r.GetBits(2)
    sfIndex := r.GetBits(4)
    // ... compare against reference
}
```

---

## Phase 2: Huffman Decoding Layer

### Step 2.1: Huffman Codebook Tables
**Source**: `codebook/hcb_*.h` (12 files, ~3000 lines total)
**Lines**: ~500 (Go representation, split across 13 files)
**Files to create**:
- `internal/huffman/codebook.go` - Types and lookup logic
- `internal/huffman/codebook_1.go` - HCB_1 table (2-step)
- `internal/huffman/codebook_2.go` - HCB_2 table (2-step)
- `internal/huffman/codebook_3.go` - HCB_3 table (binary)
- `internal/huffman/codebook_4.go` - HCB_4 table (2-step)
- `internal/huffman/codebook_5.go` - HCB_5 table (binary)
- `internal/huffman/codebook_6.go` - HCB_6 table (2-step)
- `internal/huffman/codebook_7.go` - HCB_7 table (binary)
- `internal/huffman/codebook_8.go` - HCB_8 table (2-step)
- `internal/huffman/codebook_9.go` - HCB_9 table (binary)
- `internal/huffman/codebook_10.go` - HCB_10 table (2-step)
- `internal/huffman/codebook_11.go` - HCB_11 table (2-step)
- `internal/huffman/codebook_sf.go` - Scale factor codebook (binary)

**Tables to port**:
| Codebook | Method | Source File |
|----------|--------|-------------|
| HCB_1 | 2-Step | hcb_1.h |
| HCB_2 | 2-Step | hcb_2.h |
| HCB_3 | Binary | hcb_3.h |
| HCB_4 | 2-Step | hcb_4.h |
| HCB_5 | Binary | hcb_5.h |
| HCB_6 | 2-Step | hcb_6.h |
| HCB_7 | Binary | hcb_7.h |
| HCB_8 | 2-Step | hcb_8.h |
| HCB_9 | Binary | hcb_9.h |
| HCB_10 | 2-Step | hcb_10.h |
| HCB_11 | 2-Step | hcb_11.h |
| HCB_SF | Binary | hcb_sf.h |

**Data structures**:
```c
// 1st step table
typedef struct { uint8_t offset; uint8_t extra_bits; } hcb;

// 2nd step table (pairs)
typedef struct { uint8_t bits; int8_t x; int8_t y; } hcb_2_pair;

// 2nd step table (quads)
typedef struct { uint8_t bits; int8_t x; int8_t y; int8_t v; int8_t w; } hcb_2_quad;

// Binary search table
typedef struct { uint8_t is_leaf; int8_t data[4]; } hcb_bin_quad;
typedef struct { uint8_t is_leaf; int8_t data[2]; } hcb_bin_pair;
```

**Acceptance criteria**:
- All 12 codebook tables ported
- Go structs mirror C structs
- Table sizes match original

---

### Step 2.2: Huffman Decoder Functions (`huffman.go`)
**Source**: `huffman.c` (594 lines), `huffman.h` (47 lines)
**Lines**: ~400
**Files to create**:
- `internal/huffman/decoder.go`
- `internal/huffman/decoder_test.go`

**Key functions to port**:
```c
int8_t huffman_scale_factor(bitfile *ld);
uint8_t huffman_spectral_data(uint8_t cb, bitfile *ld, int16_t *sp);

// Internal helpers
static INLINE int8_t huffman_2step_quad(uint8_t cb, bitfile *ld, int8_t *sp);
static INLINE int8_t huffman_2step_pair(uint8_t cb, bitfile *ld, int8_t *sp);
static INLINE int8_t huffman_binary_quad(uint8_t cb, bitfile *ld, int8_t *sp);
static INLINE int8_t huffman_binary_pair(uint8_t cb, bitfile *ld, int8_t *sp);
static INLINE int16_t huffman_getescape(bitfile *ld, int16_t sp);
static INLINE int8_t huffman_2step_quad_sign(uint8_t cb, bitfile *ld, int8_t *sp);
static INLINE int8_t huffman_2step_pair_sign(uint8_t cb, bitfile *ld, int8_t *sp);
static INLINE int8_t huffman_binary_quad_sign(uint8_t cb, bitfile *ld, int8_t *sp);
static INLINE int8_t huffman_binary_pair_sign(uint8_t cb, bitfile *ld, int8_t *sp);
```

**Acceptance criteria**:
- Scale factor decoding works
- All 11 spectral codebooks work
- Escape code handling correct
- Sign bit handling correct
- Unit tests with known encoded data

---

### Step 2.3: Inverse Quantization Table
**Source**: `iq_table.h` (16,458 lines)
**Lines**: ~100 (generated)
**Files to create**:
- `internal/tables/iq_table.go`
- `internal/tables/generate_iq.go` (generator script)

**Table structure**:
```c
// Table of 2^(i/4) for i in 0..8191
static const real_t iq_table[8192] = { ... };
```

**Acceptance criteria**:
- Table generated programmatically (not copied)
- Values match FAAD2 within floating-point tolerance
- Generator script can regenerate table

---

## Phase 3: Syntax Parsing Layer

### Step 3.1: Core Data Structures (`structs.go`)
**Source**: `structs.h` (446 lines)
**Lines**: ~300
**Files to create**:
- `internal/syntax/structs.go`

**Key structures to port**:
```c
// Individual Channel Stream
typedef struct {
    uint8_t max_sfb;
    uint8_t global_gain;
    uint8_t num_swb;
    uint8_t num_window_groups;
    uint8_t num_windows;
    uint8_t window_sequence;
    uint8_t window_group_length[8];
    uint8_t window_shape;
    uint8_t scale_factor_grouping;
    uint16_t sect_sfb_offset[8][15*8];
    uint16_t swb_offset[52];
    // ... more fields
    int16_t scale_factors[8][51];
    uint8_t ms_mask_present;
    uint8_t ms_used[8][51];
    pulse_info pul;
    tns_info tns;
} ic_stream;

// Syntax element
typedef struct {
    uint8_t channel;
    int16_t paired_channel;
    uint8_t element_instance_tag;
    uint8_t common_window;
    ic_stream ics1;
    ic_stream ics2;
} element;

// Program Config Element
typedef struct {
    uint8_t element_instance_tag;
    uint8_t object_type;
    uint8_t sf_index;
    // ... channel configuration
} program_config;

// ADTS header
typedef struct {
    uint16_t syncword;
    uint8_t id;
    uint8_t layer;
    uint8_t profile;
    // ... more fields
} adts_header;

// ADIF header
typedef struct {
    uint8_t copyright_id_present;
    uint8_t original_copy;
    uint32_t bitrate;
    // ... more fields
} adif_header;
```

**Acceptance criteria**:
- All structures match C layout
- Proper Go idioms (no pointer arithmetic)
- All field sizes correct

---

### Step 3.2: Sample Rate and Scalefactor Band Tables
**Source**: `common.c` (492 lines)
**Lines**: ~200 (split across 4 files)
**Files to create**:
- `internal/tables/sample_rates.go` - Sample rate lookup
- `internal/tables/sfb_long.go` - SFB offsets for long windows (all sample rates)
- `internal/tables/sfb_short.go` - SFB offsets for short windows (all sample rates)
- `internal/tables/sfb.go` - SFB lookup functions

**Tables to port**:
```c
// Sample rate table
static const uint32_t sample_rates[] = {
    96000, 88200, 64000, 48000, 44100, 32000,
    24000, 22050, 16000, 12000, 11025, 8000, 7350, 0, 0, 0
};

// Scalefactor band offsets for long windows
static const uint16_t swb_offset_1024_96[] = { ... };
static const uint16_t swb_offset_1024_64[] = { ... };
// ... one for each sample rate

// Scalefactor band offsets for short windows
static const uint16_t swb_offset_128_96[] = { ... };
// ... one for each sample rate
```

**Functions to port**:
```c
uint8_t get_sr_index(const uint32_t samplerate);
uint32_t get_sample_rate(const uint8_t sr_index);
uint8_t max_pred_sfb(const uint8_t sr_index);
uint8_t max_tns_sfb(const uint8_t sr_index, const uint8_t object_type, const uint8_t is_short);
int8_t can_decode_ot(const uint8_t object_type);
```

**Acceptance criteria**:
- All 12 sample rates supported
- SFB tables for all sample rates
- Long and short window variants

---

### Step 3.3: ADTS Header Parser
**Source**: `syntax.c` (adts_frame function, ~100 lines)
**Lines**: ~100
**Files to create**:
- `internal/syntax/adts.go`
- `internal/syntax/adts_test.go`

**Function to port**:
```c
uint8_t adts_frame(adts_header *adts, bitfile *ld);
```

**ADTS frame structure** (56 bits fixed + variable):
```
syncword                 12 bits  (0xFFF)
id                        1 bit   (MPEG-2/4)
layer                     2 bits  (always 0)
protection_absent         1 bit
profile                   2 bits
sampling_frequency_index  4 bits
private_bit               1 bit
channel_configuration     3 bits
original                  1 bit
home                      1 bit
copyright_id_bit          1 bit
copyright_id_start        1 bit
frame_length             13 bits
buffer_fullness          11 bits
no_raw_data_blocks        2 bits
```

**Acceptance criteria**:
- Parse ADTS sync word (0xFFF)
- Extract sample rate index
- Extract channel configuration
- Handle CRC when present
- Unit tests with real ADTS frames
- **FAAD2 Validation**: Compare parsed fields against `frame_NNNN_adts.bin` reference data

**Test procedure**:
```bash
# Generate reference
./scripts/check_faad2 testdata/test.aac

# Compare in Go test
func TestADTSParser_FAAD2Reference(t *testing.T) {
    ref, _ := os.ReadFile("/tmp/faad2_ref_test/frame_0001_adts.bin")
    // ref[5] = profile, ref[6] = sf_index, ref[8] = channel_config
    // Compare against parsed Go values
}
```

---

### Step 3.4: ADIF Header Parser
**Source**: `syntax.c` (get_adif_header function, ~50 lines)
**Lines**: ~80
**Files to create**:
- `internal/syntax/adif.go`
- `internal/syntax/adif_test.go`

**Function to port**:
```c
void get_adif_header(adif_header *adif, bitfile *ld);
```

**Acceptance criteria**:
- Detect "ADIF" magic bytes
- Parse bitrate info
- Parse program config elements
- Unit tests with ADIF samples

---

### Step 3.5: Program Config Element Parser
**Source**: `syntax.c` (program_config_element, ~100 lines)
**Lines**: ~120
**Files to create**:
- `internal/syntax/pce.go`
- `internal/syntax/pce_test.go`

**Function to port**:
```c
static uint8_t program_config_element(program_config *pce, bitfile *ld);
```

**Acceptance criteria**:
- Parse front/side/back/LFE channel elements
- Parse coupling channel elements
- Parse comment field
- Compute total channel count

---

### Step 3.6: MP4 AudioSpecificConfig Parser
**Source**: `mp4.c` (313 lines), `mp4.h` (53 lines)
**Lines**: ~200
**Files to create**:
- `internal/mp4/asc.go`
- `internal/mp4/asc_test.go`

**Functions to port**:
```c
int8_t AudioSpecificConfig2(uint8_t *pBuffer, uint32_t buffer_size,
                            mp4AudioSpecificConfig *mp4ASC,
                            program_config *pce, uint8_t short_form);
int8_t AudioSpecificConfigFromBitfile(bitfile *ld, mp4AudioSpecificConfig *mp4ASC,
                                      program_config *pce, uint32_t bsize, uint8_t short_form);
int8_t GASpecificConfig(bitfile *ld, mp4AudioSpecificConfig *mp4ASC, program_config *pce);
```

**Acceptance criteria**:
- Parse objectTypeIndex
- Parse samplingFrequencyIndex
- Parse channelsConfiguration
- Parse GA specific config
- Detect SBR/PS signaling

---

### Step 3.7: Individual Channel Stream Parser
**Source**: `syntax.c` (individual_channel_stream, ~300 lines)
**Lines**: ~300 (split across 5 files)
**Files to create**:
- `internal/syntax/ics.go` - Main ICS parsing and ics_info
- `internal/syntax/section.go` - Section data parsing
- `internal/syntax/scalefactor.go` - Scale factor parsing
- `internal/syntax/spectral.go` - Spectral data parsing
- `internal/syntax/ics_test.go`

**Functions to port**:
```c
// ics.go
static uint8_t individual_channel_stream(NeAACDecStruct *hDecoder, element *ele,
                                         bitfile *ld, ic_stream *ics, uint8_t scal_flag,
                                         int16_t *spec_data);
static uint8_t ics_info(NeAACDecStruct *hDecoder, ic_stream *ics, bitfile *ld,
                        uint8_t common_window);

// section.go
static uint8_t section_data(NeAACDecStruct *hDecoder, ic_stream *ics, bitfile *ld);

// scalefactor.go
static uint8_t scale_factor_data(NeAACDecStruct *hDecoder, ic_stream *ics, bitfile *ld);

// spectral.go
static uint8_t spectral_data(NeAACDecStruct *hDecoder, ic_stream *ics, bitfile *ld,
                             int16_t *spectral_data);
```

**Acceptance criteria**:
- Parse window sequence
- Parse window shape
- Parse scale factor grouping
- Parse section data
- Parse scale factors
- Parse spectral data using Huffman decoder

---

### Step 3.8: Single Channel Element (SCE) Parser
**Source**: `syntax.c` (single_lfe_channel_element, ~50 lines)
**Lines**: ~60
**Files to create**:
- `internal/syntax/sce.go`

**Function to port**:
```c
static void single_lfe_channel_element(NeAACDecStruct *hDecoder, bitfile *ld,
                                       uint8_t channel, uint8_t *tag);
```

**Acceptance criteria**:
- Parse element instance tag
- Call ICS parser
- Store spectral data

---

### Step 3.9: Channel Pair Element (CPE) Parser
**Source**: `syntax.c` (channel_pair_element, ~100 lines)
**Lines**: ~120
**Files to create**:
- `internal/syntax/cpe.go`

**Function to port**:
```c
static void channel_pair_element(NeAACDecStruct *hDecoder, bitfile *ld,
                                 uint8_t channel, uint8_t *tag);
```

**Acceptance criteria**:
- Parse element instance tag
- Parse common_window flag
- Parse M/S mask if common_window
- Parse both ICS
- Handle stereo coupling

---

### Step 3.10: Coupling Channel Element (CCE) Parser
**Source**: `syntax.c` (coupling_channel_element, ~150 lines)
**Lines**: ~150
**Files to create**:
- `internal/syntax/cce.go`

**Function to port**:
```c
static void coupling_channel_element(NeAACDecStruct *hDecoder, bitfile *ld);
```

**Acceptance criteria**:
- Parse coupling parameters
- Parse gain element lists
- (Lower priority - CCE rarely used)

---

### Step 3.11: Fill Element & Extension Payload Parser
**Source**: `syntax.c` (fill_element, ~80 lines)
**Lines**: ~100
**Files to create**:
- `internal/syntax/fill.go`

**Functions to port**:
```c
static void fill_element(NeAACDecStruct *hDecoder, bitfile *ld, drc_info *drc, uint8_t sbr_ele);
static uint8_t extension_payload(bitfile *ld, drc_info *drc, uint8_t *sbr_ele, uint16_t cnt);
static uint8_t dynamic_range_info(bitfile *ld, drc_info *drc);
```

**Acceptance criteria**:
- Parse fill data
- Parse SBR extension data (for later SBR support)
- Parse dynamic range control info

---

### Step 3.12: Raw Data Block Parser (Main Entry Point)
**Source**: `syntax.c` (raw_data_block, ~150 lines)
**Lines**: ~150
**Files to create**:
- `internal/syntax/raw_data_block.go`

**Function to port**:
```c
void raw_data_block(NeAACDecStruct *hDecoder, NeAACDecFrameInfo *hInfo,
                    bitfile *ld, program_config *pce, drc_info *drc);
```

**Parsing loop**:
```c
while ((id_syn_ele = (uint8_t)faad_getbits(ld, LEN_SE_ID)) != ID_END) {
    switch (id_syn_ele) {
    case ID_SCE: single_lfe_channel_element(...); break;
    case ID_CPE: channel_pair_element(...); break;
    case ID_CCE: coupling_channel_element(...); break;
    case ID_LFE: single_lfe_channel_element(...); break;
    case ID_DSE: data_stream_element(...); break;
    case ID_PCE: program_config_element(...); break;
    case ID_FIL: fill_element(...); break;
    }
}
```

**Acceptance criteria**:
- Parse all element types
- Proper element counting
- Channel mapping

---

## Phase 4: Spectral Processing Layer

### Step 4.1: Window Grouping Information
**Source**: `specrec.c` (window_grouping_info, ~100 lines)
**Lines**: ~100
**Files to create**:
- `internal/spectrum/window.go`
- `internal/spectrum/window_test.go`

**Function to port**:
```c
uint8_t window_grouping_info(NeAACDecStruct *hDecoder, ic_stream *ics);
```

**Window sequences**:
- ONLY_LONG_SEQUENCE (0)
- LONG_START_SEQUENCE (1)
- EIGHT_SHORT_SEQUENCE (2)
- LONG_STOP_SEQUENCE (3)

**Acceptance criteria**:
- Compute window group lengths
- Compute SFB offsets per group
- Handle short block grouping

---

### Step 4.2: Pulse Data Decoder
**Source**: `pulse.c` (58 lines), `pulse.h` (43 lines)
**Lines**: ~60
**Files to create**:
- `internal/spectrum/pulse.go`
- `internal/spectrum/pulse_test.go`

**Function to port**:
```c
uint8_t pulse_decode(ic_stream *ics, int16_t *spec_coef, uint16_t framelen);
```

**Acceptance criteria**:
- Add pulse amplitudes to spectral coefficients
- Validate pulse positions
- Error on pulse in short blocks

---

### Step 4.3: Inverse Quantization
**Source**: `specrec.c` (inverse_quantization, ~150 lines)
**Lines**: ~150
**Files to create**:
- `internal/spectrum/requant.go`
- `internal/spectrum/requant_test.go`

**Functions to port**:
```c
static void quant_to_spec(ic_stream *ics, real_t *spec_coef, uint16_t frame_len);
// Uses iq_table for x^(4/3) computation
```

**Formula**: `spec[i] = sign(quant[i]) * |quant[i]|^(4/3)`

**Acceptance criteria**:
- Correct inverse quantization
- Proper sign handling
- Use lookup table for efficiency

---

### Step 4.4: Scale Factor Application
**Source**: `specrec.c` (apply_scalefactors, ~100 lines)
**Lines**: ~100
**Files to create**:
- `internal/spectrum/scalefac.go`
- `internal/spectrum/scalefac_test.go`

**Function to port**:
```c
static void apply_scalefactors(NeAACDecStruct *hDecoder, ic_stream *ics,
                               real_t *x_invquant, uint16_t frame_len);
```

**Formula**: `spec[i] *= 2^((sf - SF_OFFSET) / 4)`

**Acceptance criteria**:
- Apply scale factors per SFB
- Handle window groups correctly

---

### Step 4.5: M/S Stereo Decoding
**Source**: `ms.c` (77 lines), `ms.h` (44 lines)
**Lines**: ~80
**Files to create**:
- `internal/spectrum/ms.go`
- `internal/spectrum/ms_test.go`

**Function to port**:
```c
void ms_decode(ic_stream *ics, ic_stream *icsr, real_t *l_spec, real_t *r_spec,
               uint16_t frame_len);
```

**Transform**:
```
L = (M + S) / sqrt(2)
R = (M - S) / sqrt(2)
```

**Acceptance criteria**:
- Apply M/S transform per SFB
- Respect ms_used flags
- Handle window groups

---

### Step 4.6: Intensity Stereo Decoding
**Source**: `is.c` (106 lines), `is.h` (67 lines)
**Lines**: ~100
**Files to create**:
- `internal/spectrum/is.go`
- `internal/spectrum/is_test.go`

**Function to port**:
```c
void is_decode(ic_stream *ics, ic_stream *icsr, real_t *l_spec, real_t *r_spec,
               uint16_t frame_len);
```

**Acceptance criteria**:
- Detect intensity stereo bands (INTENSITY_HCB, INTENSITY_HCB2)
- Apply intensity stereo scaling
- Handle polarity inversion

---

### Step 4.7: Perceptual Noise Substitution (PNS)
**Source**: `pns.c` (270 lines), `pns.h` (57 lines)
**Lines**: ~200
**Files to create**:
- `internal/spectrum/pns.go`
- `internal/spectrum/rng.go` - Random number generator
- `internal/spectrum/pns_test.go`

**Functions to port**:
```c
void pns_decode(ic_stream *ics_left, ic_stream *ics_right,
                real_t *spec_left, real_t *spec_right, uint16_t frame_len,
                uint8_t channel_pair, uint8_t object_type,
                uint32_t *__r1, uint32_t *__r2);
```

**Also port RNG**:
```c
uint32_t ne_rng(uint32_t *__r1, uint32_t *__r2);
```

**Acceptance criteria**:
- Detect noise-filled bands (NOISE_HCB)
- Generate pseudo-random noise
- Scale noise by PNS scale factor
- Handle PNS correlation between channels

---

### Step 4.8: Temporal Noise Shaping (TNS)
**Source**: `tns.c` (339 lines), `tns.h` (51 lines)
**Lines**: ~250
**Files to create**:
- `internal/spectrum/tns.go`
- `internal/spectrum/tns_test.go`

**Function to port**:
```c
void tns_decode_frame(ic_stream *ics, tns_info *tns, uint8_t sr_index,
                      uint8_t object_type, real_t *spec, uint16_t frame_len);
```

**Internal functions**:
```c
static void tns_decode_coef(uint8_t order, uint8_t coef_res_bits,
                            uint8_t coef_compress, uint8_t *coef, real_t *a);
static void tns_ar_filter(real_t *spectrum, uint16_t size, int8_t inc,
                          real_t *lpc, uint8_t order);
```

**Acceptance criteria**:
- Decode TNS filter coefficients
- Apply all-pole IIR filter
- Handle forward and backward filtering
- Handle multiple TNS filters per window

---

### Step 4.9: Long Term Prediction (LTP) - Optional
**Source**: `lt_predict.c` (215 lines), `lt_predict.h` (66 lines)
**Lines**: ~200
**Files to create**:
- `internal/spectrum/ltp.go`
- `internal/spectrum/ltp_test.go`

**Functions to port**:
```c
void lt_prediction(ic_stream *ics, ltp_info *ltp, real_t *spec,
                   int16_t *lt_pred_stat, fb_info *fb,
                   uint8_t win_shape, uint8_t win_shape_prev,
                   uint8_t sr_index, uint8_t object_type, uint16_t frame_len);
void lt_update_state(int16_t *lt_pred_stat, real_t *time,
                     real_t *overlap, uint16_t frame_len, uint8_t object_type);
```

**Acceptance criteria**:
- Decode LTP lag and coefficient
- Apply LTP prediction
- Update LTP state buffer
- (Required for LTP profile, optional for LC)

---

### Step 4.10: Main Profile Prediction (IC Predict) - Optional
**Source**: `ic_predict.c` (281 lines), `ic_predict.h` (252 lines)
**Lines**: ~300
**Files to create**:
- `internal/spectrum/predict.go`
- `internal/spectrum/predict_test.go`

**Functions to port**:
```c
void ic_prediction(ic_stream *ics, real_t *spec, pred_state *state,
                   uint16_t frame_len, uint8_t sf_index);
void pns_reset_pred_state(ic_stream *ics, pred_state *state);
void reset_all_predictors(pred_state *state, uint16_t frame_len);
```

**Acceptance criteria**:
- (Required for MAIN profile, not for LC)
- Backward adaptive prediction
- Predictor reset handling

---

### Step 4.11: Spectral Reconstruction - Single Channel
**Source**: `specrec.c` (reconstruct_single_channel, ~200 lines)
**Lines**: ~200
**Files to create**:
- `internal/spectrum/reconstruct.go` - Main reconstruction logic
- `internal/spectrum/reconstruct_test.go`

**Function to port**:
```c
uint8_t reconstruct_single_channel(NeAACDecStruct *hDecoder, ic_stream *ics,
                                   element *sce, int16_t *spec_data);
```

**Processing order**:
1. Pulse decode
2. Inverse quantization
3. Apply scale factors
4. PNS (if noise bands present)
5. TNS (if TNS data present)
6. LTP (if LTP profile)
7. IC Prediction (if MAIN profile)

**Acceptance criteria**:
- Correct processing order
- All tools applied correctly
- Output matches reference

---

### Step 4.12: Spectral Reconstruction - Channel Pair
**Source**: `specrec.c` (reconstruct_channel_pair, ~300 lines)
**Lines**: ~300
**Files to create**:
- (extends `internal/specrec/reconstruct.go`)

**Function to port**:
```c
uint8_t reconstruct_channel_pair(NeAACDecStruct *hDecoder, ic_stream *ics1,
                                 ic_stream *ics2, element *cpe,
                                 int16_t *spec_data1, int16_t *spec_data2);
```

**Additional processing for stereo**:
1. M/S stereo decode
2. Intensity stereo decode
3. PNS with correlation

**Acceptance criteria**:
- Single channel processing for each channel
- M/S applied correctly
- Intensity stereo applied correctly
- PNS correlation handled

---

## Phase 5: Filter Bank Layer

### Step 5.1: FFT Implementation (Complex FFT)
**Source**: `cfft.c` (1050 lines), `cfft.h` (56 lines), `cfft_tab.h` (1823 lines)
**Lines**: ~800 (split across 4 files)
**Files to create**:
- `internal/fft/cfft.go` - FFT algorithm
- `internal/fft/twiddle.go` - Twiddle factor computation
- `internal/fft/tables.go` - Precomputed twiddle tables (generated)
- `internal/fft/cfft_test.go`

**Functions to port**:
```c
cfft_info *cffti(uint16_t n);
void cfftu(cfft_info *cfft);
void cfftf(cfft_info *cfft, complex_t *c);  // Forward FFT
void cfftb(cfft_info *cfft, complex_t *c);  // Backward FFT
```

**Data structure**:
```c
typedef struct {
    uint16_t n;
    uint16_t ifac[15];
    complex_t *work;
    complex_t *tab;
} cfft_info;
```

**Acceptance criteria**:
- FFT sizes: 64, 512 (for MDCT 128, 1024)
- Forward and backward transforms
- Results match reference implementation
- Twiddle factor tables precomputed

---

### Step 5.2: MDCT Implementation
**Source**: `mdct.c` (301 lines), `mdct.h` (48 lines), `mdct_tab.h` (3655 lines)
**Lines**: ~400
**Files to create**:
- `internal/mdct/mdct.go`
- `internal/mdct/mdct_test.go`
- `internal/mdct/tables.go`

**Functions to port**:
```c
mdct_info *faad_mdct_init(uint16_t N);
void faad_mdct_end(mdct_info *mdct);
void faad_imdct(mdct_info *mdct, real_t *X_in, real_t *X_out);  // Inverse MDCT
void faad_mdct(mdct_info *mdct, real_t *X_in, real_t *X_out);   // Forward MDCT (for LTP)
```

**Data structure**:
```c
typedef struct {
    uint16_t N;
    cfft_info *cfft;
    complex_t *sincos;
} mdct_info;
```

**Acceptance criteria**:
- IMDCT sizes: 256 (short), 2048 (long)
- Results match reference
- Pre/post twiddle tables

---

### Step 5.3: Window Functions
**Source**: `kbd_win.h` (2297 lines), `sine_win.h` (4304 lines)
**Lines**: ~200 (generated, split by window type)
**Files to create**:
- `internal/filterbank/window.go` - Window lookup interface
- `internal/filterbank/window_sine.go` - Sine window tables (generated)
- `internal/filterbank/window_kbd.go` - KBD window tables (generated)
- `scripts/generate_windows.go` - Window generator script

**Window types**:
1. **Sine window**: `sin((pi/N) * (n + 0.5))`
2. **KBD window**: Kaiser-Bessel Derived window

**Window sizes**:
- Long: 2048 samples
- Short: 256 samples
- LD: 512, 480 samples

**Acceptance criteria**:
- Generate windows programmatically via `go generate`
- Match FAAD2 window values
- Both sine and KBD windows
- Generated files have `// Code generated` header

---

### Step 5.4: Filter Bank - Overlap-Add
**Source**: `filtbank.c` (408 lines), `filtbank.h` (61 lines)
**Lines**: ~350
**Files to create**:
- `internal/filterbank/filterbank.go`
- `internal/filterbank/filterbank_test.go`

**Functions to port**:
```c
fb_info *filter_bank_init(uint16_t frame_len);
void filter_bank_end(fb_info *fb);
void ifilter_bank(fb_info *fb, uint8_t window_sequence, uint8_t window_shape,
                  uint8_t window_shape_prev, real_t *freq_in,
                  real_t *time_out, real_t *overlap,
                  uint8_t object_type, uint16_t frame_len);
```

**Window sequence handling**:
```
ONLY_LONG_SEQUENCE:   [----long----][----long----]
LONG_START_SEQUENCE:  [----long----][short|zero-]
EIGHT_SHORT_SEQUENCE: [s][s][s][s][s][s][s][s]
LONG_STOP_SEQUENCE:   [-zero|short][----long----]
```

**Acceptance criteria**:
- IMDCT + windowing + overlap-add
- All window sequences handled
- Window shape transitions (sine/KBD)
- Output time samples correct

---

### Step 5.5: Filter Bank for LTP (Forward Transform) - Optional
**Source**: `filtbank.c` (filter_bank_ltp, ~100 lines)
**Lines**: ~100
**Files to create**:
- (extends filterbank.go)

**Function to port**:
```c
void filter_bank_ltp(fb_info *fb, uint8_t window_sequence, uint8_t window_shape,
                     uint8_t window_shape_prev, real_t *in_data, real_t *out_mdct,
                     uint8_t object_type, uint16_t frame_len);
```

**Acceptance criteria**:
- Forward MDCT for LTP
- (Required only for LTP profile)

---

## Phase 6: Output Stage

### Step 6.1: Dynamic Range Control (DRC)
**Source**: `drc.c` (172 lines), `drc.h` (49 lines)
**Lines**: ~150
**Files to create**:
- `internal/output/drc.go`
- `internal/output/drc_test.go`

**Functions to port**:
```c
drc_info *drc_init(real_t cut, real_t boost);
void drc_end(drc_info *drc);
void drc_decode(drc_info *drc, real_t *spec);
```

**Acceptance criteria**:
- Apply DRC compression/expansion
- Handle excluded channels
- Configurable cut/boost factors

---

### Step 6.2: PCM Output Conversion
**Source**: `output.c` (563 lines), `output.h` (48 lines)
**Lines**: ~400
**Files to create**:
- `internal/output/pcm.go`
- `internal/output/pcm_test.go`

**Function to port**:
```c
void* output_to_PCM(NeAACDecStruct *hDecoder, real_t **input,
                    void *samplebuffer, uint8_t channels,
                    uint16_t frame_len, uint8_t format);
```

**Output formats**:
- 16-bit signed integer (most common)
- 24-bit signed integer
- 32-bit signed integer
- 32-bit float
- 64-bit double

**Acceptance criteria**:
- All output formats working
- Proper dithering for 16-bit
- Clipping handling
- Channel interleaving
- **FAAD2 Validation**: PCM output MUST match `frame_NNNN_pcm.bin` exactly (bit-perfect)

**Test procedure**:
```bash
# Generate reference PCM
./scripts/check_faad2 testdata/test.aac

# In Go test, compare PCM output
func TestPCMOutput_FAAD2Reference(t *testing.T) {
    // Decode frame with Go
    pcm := decoder.DecodeFrame(frameData)

    // Load FAAD2 reference
    ref, _ := os.ReadFile("/tmp/faad2_ref_test/frame_0001_pcm.bin")

    // Compare (int16 little-endian)
    for i := 0; i < len(ref)/2; i++ {
        expected := int16(binary.LittleEndian.Uint16(ref[i*2:]))
        if pcm[i] != expected {
            t.Errorf("Sample %d: got %d, want %d", i, pcm[i], expected)
        }
    }
}
```

**Tolerance**: For floating-point intermediate stages, allow ±1 LSB difference in final int16 output due to rounding.

---

### Step 6.3: Downmix Matrix (5.1 to Stereo)
**Source**: `output.c` (downmix sections)
**Lines**: ~100
**Files to create**:
- `internal/output/downmix.go`

**Downmix coefficients**:
```
L = FL + 0.707*FC + 0.707*BL
R = FR + 0.707*FC + 0.707*BR
```

**Acceptance criteria**:
- 5.1 to stereo downmix
- Configurable enable/disable

---

## Phase 7: Decoder Integration

### Step 7.1: Decoder State Structure
**Source**: `structs.h` (NeAACDecStruct, ~100 lines)
**Lines**: ~150
**Files to create**:
- `decoder.go`

**State to maintain**:
```go
type Decoder struct {
    // Configuration
    config       Config
    objectType   uint8
    sfIndex      uint8
    frameLength  uint16

    // Header state
    adtsPresent  bool
    adifPresent  bool

    // Processing state
    fb           *FilterBank
    drc          *DRC

    // Per-channel state
    timeOut      [][]float32
    fbIntermed   [][]float32
    windowShape  []uint8

    // LTP state (if enabled)
    ltpState     [][]int16

    // SBR state (if enabled)
    sbr          []*SBR
}
```

**Acceptance criteria**:
- All decoder state encapsulated
- Proper initialization
- Proper cleanup

---

### Step 7.2: Decoder Initialization
**Source**: `decoder.c` (NeAACDecOpen, NeAACDecInit, ~200 lines)
**Lines**: ~200
**Files to create**:
- (extends decoder.go)

**Functions to port**:
```c
NeAACDecHandle NeAACDecOpen(void);
long NeAACDecInit(NeAACDecHandle hDecoder, unsigned char *buffer,
                  unsigned long buffer_size, unsigned long *samplerate,
                  unsigned char *channels);
char NeAACDecInit2(NeAACDecHandle hDecoder, unsigned char *pBuffer,
                   unsigned long SizeOfDecoderSpecificInfo,
                   unsigned long *samplerate, unsigned char *channels);
```

**Acceptance criteria**:
- Create decoder with defaults
- Initialize from ADTS/ADIF stream
- Initialize from AudioSpecificConfig
- Return sample rate and channels

---

### Step 7.3: Frame Decoding Loop
**Source**: `decoder.c` (aac_frame_decode, ~400 lines)
**Lines**: ~400
**Files to create**:
- (extends decoder.go)

**Function to port**:
```c
static void* aac_frame_decode(NeAACDecStruct *hDecoder,
                              NeAACDecFrameInfo *hInfo,
                              unsigned char *buffer,
                              unsigned long buffer_size,
                              void **sample_buffer2,
                              unsigned long sample_buffer_size);
```

**Processing flow**:
1. Skip ID3 tags if present
2. Parse ADTS header if present
3. Parse raw_data_block
4. For each element:
   - Reconstruct spectral data
   - Apply filter bank
5. SBR processing (if enabled)
6. Convert to PCM output

**Acceptance criteria**:
- Full frame decode working
- Proper error handling
- Frame info populated

---

### Step 7.4: Public API
**Source**: `neaacdec.h`, `decoder.c`
**Lines**: ~150
**Files to create**:
- `aac.go` (public API)

**Go API design**:
```go
// NewDecoder creates a new AAC decoder
func NewDecoder() *Decoder

// Init initializes decoder from ADTS/ADIF stream, returns sample rate and channels
func (d *Decoder) Init(data []byte) (sampleRate uint32, channels uint8, err error)

// Init2 initializes decoder from AudioSpecificConfig
func (d *Decoder) Init2(asc []byte) (sampleRate uint32, channels uint8, err error)

// Decode decodes one AAC frame, returns PCM samples
func (d *Decoder) Decode(frame []byte) ([]int16, error)

// DecodeFloat decodes one AAC frame, returns float samples
func (d *Decoder) DecodeFloat(frame []byte) ([]float32, error)

// Close releases decoder resources
func (d *Decoder) Close()
```

**Acceptance criteria**:
- Clean Go-idiomatic API
- Thread-safe
- Proper resource cleanup
- Error returns (not error codes)

---

## Phase 8: HE-AAC Support (Optional - SBR)

### Step 8.1: SBR Data Structures
**Source**: `sbr_dec.h` (258 lines)
**Lines**: ~200
**Files to create**:
- `internal/sbr/structs.go`

**Key structure**: `sbr_info` with ~100 fields

**Acceptance criteria**:
- All SBR state fields defined

---

### Step 8.2: SBR Initialization
**Source**: `sbr_dec.c` (sbrDecodeInit, ~100 lines)
**Lines**: ~100
**Files to create**:
- `internal/sbr/sbr.go`

**Acceptance criteria**:
- Initialize SBR state
- Allocate QMF buffers

---

### Step 8.3: SBR Bitstream Parsing
**Source**: `sbr_syntax.c` (921 lines), `sbr_huff.c` (365 lines)
**Lines**: ~800
**Files to create**:
- `internal/sbr/syntax.go`
- `internal/sbr/huffman.go`

**Functions to port**:
```c
uint8_t sbr_extension_data(bitfile *ld, sbr_info *sbr, uint16_t cnt, uint8_t id_aac);
static void sbr_header(bitfile *ld, sbr_info *sbr);
static void sbr_data(bitfile *ld, sbr_info *sbr);
static void sbr_grid(bitfile *ld, sbr_info *sbr, uint8_t ch);
static void sbr_dtdf(bitfile *ld, sbr_info *sbr, uint8_t ch);
static void sbr_invf(bitfile *ld, sbr_info *sbr, uint8_t ch);
static void sbr_envelope(bitfile *ld, sbr_info *sbr, uint8_t ch);
static void sbr_noise(bitfile *ld, sbr_info *sbr, uint8_t ch);
static void sinusoidal_coding(bitfile *ld, sbr_info *sbr, uint8_t ch);
```

**Acceptance criteria**:
- Parse SBR header
- Parse SBR frame data
- Envelope and noise data

---

### Step 8.4: QMF Analysis Filter Bank
**Source**: `sbr_qmf.c` (635 lines), `sbr_qmf_c.h` (368 lines)
**Lines**: ~500
**Files to create**:
- `internal/sbr/qmf.go`
- `internal/sbr/qmf_tables.go`

**Functions to port**:
```c
qmfa_info *qmfa_init(uint8_t channels);
void qmfa_end(qmfa_info *qmfa);
void sbr_qmf_analysis_32(sbr_info *sbr, qmfa_info *qmfa, const real_t *input,
                         qmf_t X[MAX_NTSRHFG][64], uint8_t offset, uint8_t kx);
```

**Acceptance criteria**:
- 32-band QMF analysis
- Polyphase implementation

---

### Step 8.5: QMF Synthesis Filter Bank
**Source**: `sbr_qmf.c`
**Lines**: ~300
**Files to create**:
- (extends qmf.go)

**Functions to port**:
```c
qmfs_info *qmfs_init(uint8_t channels);
void qmfs_end(qmfs_info *qmfs);
void sbr_qmf_synthesis_32(sbr_info *sbr, qmfs_info *qmfs,
                          qmf_t X[MAX_NTSRHFG][64], real_t *output);
void sbr_qmf_synthesis_64(sbr_info *sbr, qmfs_info *qmfs,
                          qmf_t X[MAX_NTSRHFG][64], real_t *output);
```

**Acceptance criteria**:
- 64-band QMF synthesis (upsampling)
- Proper reconstruction

---

### Step 8.6: SBR Frequency Band Tables
**Source**: `sbr_fbt.c` (767 lines), `sbr_fbt.h` (55 lines)
**Lines**: ~500
**Files to create**:
- `internal/sbr/fbt.go`

**Functions to port**:
```c
uint8_t sbr_start_freq(uint8_t bs_start_freq, uint8_t rate, uint8_t is_aac_plus);
uint8_t sbr_stop_freq(uint8_t bs_stop_freq, uint8_t rate, uint8_t is_aac_plus);
uint8_t master_frequency_table(sbr_info *sbr, uint8_t k0, uint8_t k2, uint8_t bs_freq_scale,
                               uint8_t bs_alter_scale);
void derived_frequency_table(sbr_info *sbr, uint8_t bs_xover_band,
                             uint8_t k2, uint8_t is_aac_plus);
```

**Acceptance criteria**:
- Master frequency table
- High/low frequency tables
- Noise band table
- Limiter band table

---

### Step 8.7: SBR Time-Frequency Grid
**Source**: `sbr_tf_grid.c` (262 lines), `sbr_tf_grid.h` (47 lines)
**Lines**: ~200
**Files to create**:
- `internal/sbr/tfgrid.go`

**Function to port**:
```c
uint8_t sbr_time_freq_grid(bitfile *ld, sbr_info *sbr, uint8_t ch);
```

**Acceptance criteria**:
- Envelope borders
- Noise floor borders

---

### Step 8.8: SBR High Frequency Generation
**Source**: `sbr_hfgen.c` (684 lines), `sbr_hfgen.h` (49 lines)
**Lines**: ~500
**Files to create**:
- `internal/sbr/hfgen.go`

**Functions to port**:
```c
void hf_generation(sbr_info *sbr, qmf_t Xlow[MAX_NTSRHFG][64],
                   qmf_t Xhigh[MAX_NTSRHFG][64], real_t *bw_array,
                   uint8_t ch);
```

**Acceptance criteria**:
- Patching from low to high bands
- Bandwidth expansion
- Chirp factor handling

---

### Step 8.9: SBR High Frequency Adjustment
**Source**: `sbr_hfadj.c` (1742 lines), `sbr_hfadj.h` (57 lines)
**Lines**: ~1200
**Files to create**:
- `internal/sbr/hfadj.go`

**Functions to port**:
```c
void hf_adjustment(sbr_info *sbr, qmf_t Xsbr[MAX_NTSRHFG][64], uint8_t ch);
static void estimate_current_envelope(sbr_info *sbr, sbr_hfadj_info *adj,
                                      qmf_t Xsbr[MAX_NTSRHFG][64], uint8_t ch);
static void calculate_gain(sbr_info *sbr, sbr_hfadj_info *adj, uint8_t ch);
static void hf_assembly(sbr_info *sbr, sbr_hfadj_info *adj,
                        qmf_t Xsbr[MAX_NTSRHFG][64], uint8_t ch);
```

**Acceptance criteria**:
- Envelope estimation
- Gain calculation
- Gain limiting
- Noise addition
- Sinusoid addition

---

### Step 8.10: SBR Envelope/Noise Dequantization
**Source**: `sbr_e_nf.c` (510 lines), `sbr_e_nf.h` (50 lines)
**Lines**: ~400
**Files to create**:
- `internal/sbr/envelope.go`

**Functions to port**:
```c
void envelope_noise_dequantisation(sbr_info *sbr, uint8_t ch);
```

**Acceptance criteria**:
- Delta coding modes
- Dequantization formulas

---

### Step 8.11: SBR DCT
**Source**: `sbr_dct.c` (2279 lines), `sbr_dct.h` (52 lines)
**Lines**: ~1500
**Files to create**:
- `internal/sbr/dct.go`
- `internal/sbr/dct_tables.go`

**Functions to port**:
```c
void DCT3_32_unscaled(real_t *y, real_t *x);
void DCT4_64(real_t *y, real_t *x);
void DST4_64(real_t *y, real_t *x);
void DCT2_64_unscaled(real_t *y, real_t *x);
```

**Acceptance criteria**:
- Various DCT types for QMF
- Optimized implementations

---

### Step 8.12: SBR Main Decode
**Source**: `sbr_dec.c` (698 lines)
**Lines**: ~500
**Files to create**:
- (extends sbr.go)

**Functions to port**:
```c
uint8_t sbrDecodeCoupleFrame(sbr_info *sbr, real_t *left_chan, real_t *right_chan,
                             const uint8_t just_seeked, const uint8_t downSampledSBR);
uint8_t sbrDecodeSingleFrame(sbr_info *sbr, real_t *channel,
                             const uint8_t just_seeked, const uint8_t downSampledSBR);
```

**Acceptance criteria**:
- Full SBR decode pipeline
- Upsampled output (32kHz -> 64kHz)
- Single and stereo modes

---

## Phase 9: HE-AACv2 Support (Optional - PS)

### Step 9.1: Parametric Stereo Data Structures
**Source**: `ps_dec.h` (155 lines), `ps_tables.h` (550 lines)
**Lines**: ~400
**Files to create**:
- `internal/ps/structs.go`
- `internal/ps/tables.go`

**Acceptance criteria**:
- PS state structure
- Hybrid filter tables
- Decorrelation tables

---

### Step 9.2: PS Bitstream Parsing
**Source**: `ps_syntax.c` (552 lines)
**Lines**: ~400
**Files to create**:
- `internal/ps/syntax.go`

**Functions to port**:
```c
uint16_t ps_data(ps_info *ps, bitfile *ld, uint8_t *header);
```

**Acceptance criteria**:
- IID/ICC/IPD parameters
- Envelope parsing

---

### Step 9.3: PS Decoding
**Source**: `ps_dec.c` (2040 lines)
**Lines**: ~1500
**Files to create**:
- `internal/ps/decode.go`

**Functions to port**:
```c
uint8_t ps_decode(ps_info *ps, qmf_t X_left[38][64], qmf_t X_right[38][64]);
```

**Acceptance criteria**:
- Stereo reconstruction from mono + PS data
- Hybrid analysis/synthesis
- Decorrelation

---

## Phase 10: Error Resilience (Optional)

### Step 10.1: HCR (Huffman Codeword Reordering)
**Source**: `hcr.c` (433 lines)
**Lines**: ~400
**Files to create**:
- `internal/er/hcr.go`

**Acceptance criteria**:
- Virtual codebook reordering
- Priority decoding

---

### Step 10.2: RVLC (Reversible Variable Length Coding)
**Source**: `rvlc.c` (551 lines), `rvlc.h` (56 lines)
**Lines**: ~500
**Files to create**:
- `internal/er/rvlc.go`

**Acceptance criteria**:
- Forward/backward decoding
- Error detection

---

## Phase 11: Testing & Validation

### Step 11.1: Test Infrastructure
**Files already created**:
- `testdata/generate.go` - Comprehensive test file generator
- `scripts/faad2_debug.c` - FAAD2 reference data dumper
- `scripts/check_faad2` - Comparison script
- `scripts/Makefile` - Build system for debug tools

**Test file generation**:
```bash
# Generate comprehensive test suite
go run testdata/generate.go

# Generate reference data for all test files
for f in testdata/generated/**/*.aac; do
    ./scripts/check_faad2 "$f"
done

# Quick single file test
ffmpeg -f lavfi -i "sine=frequency=440:duration=1" -c:a aac -b:a 128k test.aac
./scripts/check_faad2 test.aac
```

---

### Step 11.2: FAAD2 Reference Comparison Tests (PRIMARY)
**Files to create**:
- `test/faad2_reference_test.go`

**Methodology**:
1. Generate reference data with `./scripts/check_faad2`
2. Decode with go-aac
3. Compare against FAAD2 reference data (bit-perfect for PCM)

```go
func TestDecoder_FAAD2Reference(t *testing.T) {
    testCases := []string{
        "testdata/generated/aac_lc/44100_16_stereo_128k/sine1k.aac",
        "testdata/generated/aac_lc/48000_16_mono_64k/sweep.aac",
        // ... more cases
    }

    for _, tc := range testCases {
        t.Run(tc, func(t *testing.T) {
            // Generate reference if not exists
            refDir := generateFAAD2Reference(tc)

            // Decode with Go
            goOutput := decodeWithGo(tc)

            // Compare each frame
            for frame := 0; ; frame++ {
                refPCM := loadRefPCM(refDir, frame)
                if refPCM == nil {
                    break
                }
                goPCM := goOutput[frame]

                comparePCM(t, refPCM, goPCM, frame)
            }
        })
    }
}
```

---

### Step 11.3: Conformance Tests
**Source**: ISO/IEC 14496-4 conformance bitstreams

**Test categories**:
- AAC-LC mono/stereo
- Various sample rates (8kHz - 96kHz)
- Various bit rates (24kbps - 320kbps)
- Window sequence transitions (long/short/start/stop)
- TNS/PNS enabled files
- HE-AAC (SBR) if implemented
- HE-AACv2 (PS) if implemented

**IMPORTANT**: All conformance tests MUST first be validated with `faad2_debug` to ensure FAAD2 can decode them correctly before testing the Go implementation.

---

## Summary Statistics

| Phase | Description | Estimated Lines | Priority |
|-------|-------------|-----------------|----------|
| 1 | Foundation | ~650 | Critical |
| 2 | Huffman | ~900 | Critical |
| 3 | Syntax Parsing | ~1,500 | Critical |
| 4 | Spectral Processing | ~2,000 | Critical |
| 5 | Filter Bank | ~1,850 | Critical |
| 6 | Output | ~650 | Critical |
| 7 | Integration | ~900 | Critical |
| 8 | SBR (HE-AAC) | ~5,800 | Optional |
| 9 | PS (HE-AACv2) | ~2,300 | Optional |
| 10 | Error Resilience | ~900 | Optional |
| 11 | Testing | ~500 | Critical |

**Total for AAC-LC decoder**: ~8,950 lines
**Total with HE-AAC support**: ~17,050 lines
**Total with all optional features**: ~18,550 lines

---

## Recommended Implementation Order

1. **Milestone 1**: Bit reader + Huffman decoder (testable in isolation)
2. **Milestone 2**: ADTS parser + basic syntax parsing
3. **Milestone 3**: Complete spectral reconstruction for mono
4. **Milestone 4**: Filter bank + PCM output (first audio!)
5. **Milestone 5**: Stereo support (M/S, IS)
6. **Milestone 6**: TNS/PNS for better quality
7. **Milestone 7**: Full AAC-LC compliance
8. **Milestone 8**: SBR support (HE-AAC)
9. **Milestone 9**: PS support (HE-AACv2)

Each milestone should be independently testable and produce verifiable output.

---

## Final Package Structure

After complete migration, the project should have this structure:

```
go-aac/
├── aac.go                          # Public API: Decoder, Config, Error types
├── decoder.go                      # Main decoder implementation
├── CLAUDE.md                       # Project instructions
├── go.mod
├── Makefile
├── internal/
│   ├── bits/
│   │   ├── reader.go               # Bitstream reader
│   │   └── reader_test.go
│   ├── huffman/
│   │   ├── codebook.go             # Types and lookup logic
│   │   ├── codebook_1.go           # HCB_1 table
│   │   ├── codebook_2.go           # HCB_2 table
│   │   ├── codebook_3.go           # HCB_3 table
│   │   ├── codebook_4.go           # HCB_4 table
│   │   ├── codebook_5.go           # HCB_5 table
│   │   ├── codebook_6.go           # HCB_6 table
│   │   ├── codebook_7.go           # HCB_7 table
│   │   ├── codebook_8.go           # HCB_8 table
│   │   ├── codebook_9.go           # HCB_9 table
│   │   ├── codebook_10.go          # HCB_10 table
│   │   ├── codebook_11.go          # HCB_11 table
│   │   ├── codebook_sf.go          # Scale factor codebook
│   │   ├── decoder.go              # Huffman decoding functions
│   │   └── decoder_test.go
│   ├── syntax/
│   │   ├── types.go                # Shared types (ic_stream, element, etc.)
│   │   ├── adts.go                 # ADTS header parsing
│   │   ├── adts_test.go
│   │   ├── adif.go                 # ADIF header parsing
│   │   ├── adif_test.go
│   │   ├── pce.go                  # Program Config Element
│   │   ├── pce_test.go
│   │   ├── asc.go                  # AudioSpecificConfig (MP4)
│   │   ├── asc_test.go
│   │   ├── ics.go                  # Individual Channel Stream
│   │   ├── section.go              # Section data parsing
│   │   ├── scalefactor.go          # Scale factor parsing
│   │   ├── spectral.go             # Spectral data parsing
│   │   ├── ics_test.go
│   │   ├── sce.go                  # Single Channel Element
│   │   ├── cpe.go                  # Channel Pair Element
│   │   ├── cce.go                  # Coupling Channel Element
│   │   ├── fill.go                 # Fill elements and extensions
│   │   ├── raw_data_block.go       # Main parsing entry point
│   │   └── raw_data_block_test.go
│   ├── spectrum/
│   │   ├── window.go               # Window grouping info
│   │   ├── pulse.go                # Pulse data decoding
│   │   ├── requant.go              # Inverse quantization
│   │   ├── scalefac.go             # Scale factor application
│   │   ├── ms.go                   # M/S stereo
│   │   ├── is.go                   # Intensity stereo
│   │   ├── pns.go                  # Perceptual Noise Substitution
│   │   ├── rng.go                  # Random number generator for PNS
│   │   ├── tns.go                  # Temporal Noise Shaping
│   │   ├── ltp.go                  # Long Term Prediction (optional)
│   │   ├── predict.go              # Main profile prediction (optional)
│   │   ├── reconstruct.go          # Main reconstruction logic
│   │   └── *_test.go               # Tests for each
│   ├── filterbank/
│   │   ├── filterbank.go           # Main filter bank (IMDCT + overlap-add)
│   │   ├── window.go               # Window lookup interface
│   │   ├── window_sine.go          # Sine windows (generated)
│   │   ├── window_kbd.go           # KBD windows (generated)
│   │   └── filterbank_test.go
│   ├── mdct/
│   │   ├── mdct.go                 # MDCT/IMDCT
│   │   ├── tables.go               # Pre/post twiddle tables
│   │   └── mdct_test.go
│   ├── fft/
│   │   ├── cfft.go                 # Complex FFT
│   │   ├── twiddle.go              # Twiddle factor computation
│   │   ├── tables.go               # Precomputed twiddle tables
│   │   └── cfft_test.go
│   ├── tables/
│   │   ├── sample_rates.go         # Sample rate lookup
│   │   ├── sfb.go                  # SFB lookup functions
│   │   ├── sfb_long.go             # SFB offsets for long windows
│   │   ├── sfb_short.go            # SFB offsets for short windows
│   │   └── iq_table.go             # Inverse quantization table (generated)
│   ├── output/
│   │   ├── pcm.go                  # PCM output conversion
│   │   ├── drc.go                  # Dynamic Range Control
│   │   ├── downmix.go              # 5.1 to stereo downmix
│   │   └── pcm_test.go
│   ├── sbr/                        # HE-AAC (optional)
│   │   ├── sbr.go                  # Main SBR decoder
│   │   ├── syntax.go               # SBR bitstream parsing
│   │   ├── huffman.go              # SBR Huffman tables
│   │   ├── qmf_analysis.go         # QMF analysis filter bank
│   │   ├── qmf_synthesis.go        # QMF synthesis filter bank
│   │   ├── qmf_tables.go           # QMF filter coefficients
│   │   ├── fbt.go                  # Frequency band tables
│   │   ├── tfgrid.go               # Time-frequency grid
│   │   ├── hfgen.go                # High frequency generation
│   │   ├── hfadj.go                # High frequency adjustment
│   │   ├── envelope.go             # Envelope/noise dequantization
│   │   ├── dct.go                  # DCT implementations
│   │   └── *_test.go
│   ├── ps/                         # HE-AACv2 (optional)
│   │   ├── ps.go                   # Main PS decoder
│   │   ├── syntax.go               # PS bitstream parsing
│   │   ├── tables.go               # PS tables
│   │   └── *_test.go
│   └── er/                         # Error resilience (optional)
│       ├── hcr.go                  # Huffman Codeword Reordering
│       └── rvlc.go                 # Reversible VLC
├── scripts/
│   ├── generate_iq_table.go        # IQ table generator
│   ├── generate_windows.go         # Window coefficient generator
│   └── generate_test_vectors.sh    # Test file generator
└── testdata/
    ├── mono_44100.aac
    ├── stereo_44100.aac
    └── ...
```

**File count summary** (core AAC-LC):
- `internal/bits/`: 2 files
- `internal/huffman/`: 15 files
- `internal/syntax/`: 18 files
- `internal/spectrum/`: 14 files
- `internal/filterbank/`: 5 files
- `internal/mdct/`: 3 files
- `internal/fft/`: 4 files
- `internal/tables/`: 5 files
- `internal/output/`: 4 files
- Root: 2 files

**Total**: ~72 Go files for core AAC-LC (excluding SBR/PS/ER)
