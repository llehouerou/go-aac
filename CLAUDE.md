# go-aac: Pure Go AAC Decoder

## Project Goal

Port FAAD2 (Freeware Advanced Audio Decoder) from C to pure Go, creating a complete AAC decoder library with no CGO dependencies.

## Why This Project

- No pure Go AAC decoder exists that outputs PCM audio
- Comcast/gaad is only a parser, not a decoder
- AAC is the most common codec in M4A files (alongside ALAC)
- Needed for a pure Go audio player project

## Reference Implementation (MANDATORY)

**FAAD2**: `~/dev/faad2` (cloned from https://github.com/knik0/faad2)

- License: GPL
- Language: C
- Well-structured, readable codebase
- Supports: AAC-LC, Main, LTP, HE-AAC (SBR), HE-AACv2 (SBR+PS)

**IMPORTANT: When in doubt about implementation details, ALWAYS consult the FAAD2 source code in `~/dev/faad2/libfaad/`. This is the authoritative reference for this project. Do not guess or invent behavior - read the C code and port it accurately.**

## AAC Specifications

- **ISO/IEC 13818-7**: MPEG-2 AAC (original)
- **ISO/IEC 14496-3**: MPEG-4 Audio (current standard)

## FAAD2 Decoder Structure

Core files in `~/dev/faad2/libfaad/`:

```
decoder.c      # Main entry point, NeAACDecDecode()
syntax.c       # Bitstream parsing, raw_data_block()
specrec.c      # Spectral reconstruction
filtbank.c     # Filter bank (IMDCT)
mdct.c         # Modified Discrete Cosine Transform
huffman.c      # Huffman decoding tables
bits.c         # Bit reading utilities
output.c       # PCM output formatting
tns.c          # Temporal Noise Shaping
pns.c          # Perceptual Noise Substitution
sbr_*.c        # Spectral Band Replication (12+ files, for HE-AAC)
ps_*.c         # Parametric Stereo (for HE-AACv2)
```

## Implementation Strategy

### Phase 1: AAC-LC (Minimum Viable Decoder)
1. Bit reader (bits.c)
2. Huffman decoding (huffman.c)
3. Syntax parsing (syntax.c) - AAC-LC only
4. Spectral reconstruction (specrec.c)
5. IMDCT / Filter bank (filtbank.c, mdct.c)
6. PCM output (output.c)

### Phase 2: Full AAC Support
- Temporal Noise Shaping (tns.c)
- Perceptual Noise Substitution (pns.c)
- Long Term Prediction (ltp.c)

### Phase 3: HE-AAC (Optional)
- Spectral Band Replication (sbr_*.c)
- Parametric Stereo (ps_*.c)

## Lessons Learned from ALAC Port

The alicebob/alac library was ported from C and had several translation bugs:

1. **sizeof translation**: C's `memcpy(dst, src, n * sizeof(int32))` becomes Go's `copy(dst, src[:n])` - don't multiply by element size
2. **Sign extension**: C bitfield tricks like `struct { signed int x:24; }` need explicit bit manipulation in Go: `(v << 8) >> 8`
3. **Buffer sizing**: Verify slice bounds carefully when translating pointer arithmetic

## Development Workflow: Test-Driven Development (MANDATORY)

**All code changes MUST follow TDD. No exceptions.**

### The TDD Cycle

1. **RED**: Write a failing test first
   - Define the expected behavior before writing implementation
   - Run the test and verify it fails for the right reason

2. **GREEN**: Write minimal code to pass the test
   - Only write enough code to make the test pass
   - Do not add extra functionality

3. **REFACTOR**: Clean up while tests stay green
   - Improve code structure without changing behavior
   - Run tests after each refactor step

### Rules

- Never write implementation code without a failing test
- Each new function/method needs corresponding test coverage
- When fixing bugs, first write a test that reproduces the bug
- Run `make check` before committing any changes

### Testing Against FAAD2 Reference (MANDATORY)

**Every implementation step MUST be validated against FAAD2 reference output.**

We use a custom `faad2_debug` tool that decodes AAC files and dumps intermediate values at each pipeline stage. This allows precise comparison at every step, not just final PCM output.

#### FAAD2 Debug Tool

Located in `scripts/`:
```bash
# Build the debug tool
cd scripts && make

# Generate reference data for an AAC file
./check_faad2 input.aac

# Reference data is written to /tmp/faad2_ref_<name>/
```

Output files per frame:
- `frame_NNNN_adts.bin` - Parsed ADTS header (16 bytes)
- `frame_NNNN_pcm.bin` - Final PCM output (int16 interleaved)
- `info.json` - Stream metadata (sample rate, channels, frame count)

#### Testing Workflow

1. **Generate test AAC files** with FFmpeg:
   ```bash
   ffmpeg -f lavfi -i "sine=frequency=1000:duration=1" -c:a aac -b:a 128k test.aac
   ```

2. **Generate FAAD2 reference data**:
   ```bash
   ./scripts/check_faad2 test.aac
   ```

3. **Write Go tests that compare against reference**:
   ```go
   func TestADTSParser(t *testing.T) {
       // Load reference data
       ref, _ := os.ReadFile("/tmp/faad2_ref_test/frame_0001_adts.bin")

       // Parse with Go implementation
       data, _ := os.ReadFile("testdata/test.aac")
       header, _ := syntax.ParseADTS(bits.NewReader(data))

       // Compare against FAAD2 reference
       assert.Equal(t, ref[6], header.SFIndex)  // Sample rate index
       assert.Equal(t, ref[8], header.ChannelConfig)
   }
   ```

4. **Run comparison script** (when Go output is available):
   ```bash
   ./scripts/check_faad2 test.aac /tmp/go_output
   ```

#### Test Data Generation

Use `testdata/generate.go` to create comprehensive test files:
```bash
go run testdata/generate.go
```

This generates AAC files with various:
- Sample rates (8kHz - 96kHz)
- Channel configurations (mono, stereo)
- Profiles (AAC-LC, HE-AAC, HE-AACv2)
- Audio types (silence, sine, sweep, noise, speech-like)

## Related Projects

- **alicebob/alac**: Pure Go ALAC decoder (we fixed bugs in this)
- **hajimehoshi/go-mp3**: Pure Go MP3 decoder (good reference for structure)
- **mewkiz/flac**: Pure Go FLAC decoder

## Commands (MANDATORY: Use Makefile)

**Always use the Makefile for common operations. Never run raw go commands directly.**

```bash
# Format, lint, and test (run before committing)
make check

# Individual targets
make fmt          # Format code with goimports-reviser
make lint         # Run golangci-lint
make test         # Run all tests
make test PKG=./bits  # Run tests for specific package
make coverage     # Run tests with coverage report
make build        # Verify compilation

# Install tools (goimports-reviser, golangci-lint)
make tools

# Install pre-commit hook
make install-hooks
```

### FAAD2 Exploration

```bash
# Explore FAAD2 structure
find ~/dev/faad2/libfaad -name "*.c" | xargs wc -l | sort -n

# Count lines in core decoder (excluding SBR/PS)
wc -l ~/dev/faad2/libfaad/{decoder,syntax,specrec,filtbank,mdct,huffman,bits,output}.c
```

## Code Organization Rules (MANDATORY)

**Do NOT replicate FAAD2's flat C structure. Use proper Go package organization.**

### Package Structure

```
go-aac/
├── aac.go                    # Public API only (Decoder, Config, Error types)
├── decoder.go                # Main decoder implementation
├── internal/
│   ├── bits/                 # Bitstream reading
│   │   ├── reader.go
│   │   └── reader_test.go
│   ├── huffman/              # Huffman decoding
│   │   ├── decoder.go
│   │   ├── codebook.go       # Codebook types and lookup logic
│   │   ├── codebook_1.go     # HCB_1 table (one file per codebook)
│   │   ├── codebook_2.go     # HCB_2 table
│   │   ├── ...               # etc.
│   │   ├── codebook_sf.go    # Scale factor codebook
│   │   └── decoder_test.go
│   ├── syntax/               # Bitstream syntax parsing
│   │   ├── adts.go           # ADTS header parsing
│   │   ├── adif.go           # ADIF header parsing
│   │   ├── pce.go            # Program Config Element
│   │   ├── ics.go            # Individual Channel Stream
│   │   ├── sce.go            # Single Channel Element
│   │   ├── cpe.go            # Channel Pair Element
│   │   ├── fill.go           # Fill elements
│   │   └── raw_data_block.go # Main parsing entry point
│   ├── spectrum/             # Spectral processing
│   │   ├── requant.go        # Inverse quantization
│   │   ├── scalefac.go       # Scale factor application
│   │   ├── ms.go             # M/S stereo
│   │   ├── is.go             # Intensity stereo
│   │   ├── pns.go            # Perceptual Noise Substitution
│   │   ├── tns.go            # Temporal Noise Shaping
│   │   └── reconstruct.go    # Main reconstruction
│   ├── filterbank/           # Filter bank (IMDCT + windowing)
│   │   ├── filterbank.go
│   │   ├── window_sine.go    # Sine windows
│   │   ├── window_kbd.go     # KBD windows
│   │   └── filterbank_test.go
│   ├── mdct/                 # MDCT implementation
│   │   ├── mdct.go
│   │   ├── tables.go
│   │   └── mdct_test.go
│   ├── fft/                  # FFT implementation
│   │   ├── cfft.go
│   │   ├── tables.go
│   │   └── cfft_test.go
│   ├── tables/               # Lookup tables
│   │   ├── sample_rates.go
│   │   ├── sfb_long.go       # SFB tables for long windows
│   │   ├── sfb_short.go      # SFB tables for short windows
│   │   └── iq_table.go       # Inverse quantization table
│   ├── output/               # PCM output conversion
│   │   ├── pcm.go
│   │   ├── drc.go
│   │   └── downmix.go
│   ├── sbr/                  # HE-AAC (SBR) - optional
│   │   ├── ...
│   └── ps/                   # HE-AACv2 (PS) - optional
│       ├── ...
└── testdata/                 # Test files
```

### File Size Rules

**Maximum ~300 lines per file.** Split larger logical units:

| Instead of... | Split into... |
|---------------|---------------|
| `codebook.go` (3000 lines) | `codebook_1.go`, `codebook_2.go`, ... `codebook_sf.go` |
| `tables.go` (5000 lines) | `tables_iq.go`, `tables_sfb.go`, `tables_window.go` |
| `syntax.go` (2700 lines) | `adts.go`, `adif.go`, `ics.go`, `sce.go`, `cpe.go`, etc. |
| `specrec.go` (1400 lines) | `requant.go`, `scalefac.go`, `reconstruct.go` |

### Naming Conventions

1. **Package names**: Short, lowercase, single word (e.g., `bits`, `huffman`, `syntax`)
2. **File names**: Descriptive, snake_case for multi-word (e.g., `raw_data_block.go`)
3. **Test files**: Same name with `_test.go` suffix, same package
4. **Table files**: Prefix with what they contain (e.g., `codebook_1.go`, `window_sine.go`)

### Cross-Reference Comments

Since we're not matching FAAD2's flat structure, add source reference comments:

```go
// Package bits implements AAC bitstream reading.
// Ported from: ~/dev/faad2/libfaad/bits.c
package bits

// Reader reads bits from a byte buffer.
// Ported from: bitfile struct in bits.h:48-60
type Reader struct {
    // ...
}

// GetBits reads n bits from the stream.
// Ported from: faad_getbits() in bits.h:130-146
func (r *Reader) GetBits(n uint) uint32 {
    // ...
}
```

### Package Dependencies

Enforce clean dependency graph (no cycles):

```
aac (public API)
 └── internal/decoder
      ├── internal/syntax
      │    └── internal/bits
      │    └── internal/huffman
      ├── internal/spectrum
      │    └── internal/tables
      ├── internal/filterbank
      │    └── internal/mdct
      │         └── internal/fft
      └── internal/output
```
