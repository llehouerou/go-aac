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

### Testing Against Reference Implementation

- Use FFmpeg to encode WAV files to AAC
- Use FFmpeg to decode AAC to raw PCM (reference output)
- Compare Go decoder output against FFmpeg reference
- Test matrix: various sample rates, bit depths, channel configurations

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

## File Naming Convention

Match FAAD2 structure for easy cross-reference:
- `bits.go` <- `bits.c`
- `huffman.go` <- `huffman.c`
- `syntax.go` <- `syntax.c`
- etc.
