# Test Audio Samples

This directory contains source WAV files for real-world audio testing.

## Required Files

The test generator expects these files (public domain LibriVox recordings):

| File | Source | Description |
|------|--------|-------------|
| `jane_eyre_5s.wav` | LibriVox | "Jane Eyre" Ch.1, first 5 seconds |
| `monte_cristo_5s.wav` | LibriVox | "Count of Monte Cristo" Ch.1, first 5 seconds |

## How to Create Sample Files

### Option 1: Download and Extract (Recommended)

```bash
# Jane Eyre - LibriVox recording
curl -L "https://www.archive.org/download/jane_eyre_ver03_0809_librivox/janeeyre_01_bronte_64kb.mp3" -o jane_eyre.mp3
ffmpeg -i jane_eyre.mp3 -t 5 -ar 44100 -ac 2 jane_eyre_5s.wav

# Count of Monte Cristo - LibriVox recording
curl -L "https://www.archive.org/download/count_of_monte_cristo_0711_librivox/montecristo_001_dumas_64kb.mp3" -o monte_cristo.mp3
ffmpeg -i monte_cristo.mp3 -t 5 -ar 44100 -ac 2 monte_cristo_5s.wav

# Clean up MP3 files
rm jane_eyre.mp3 monte_cristo.mp3
```

### Option 2: Use Any Public Domain Audio

You can substitute any WAV files as long as they are:
- 44.1 kHz sample rate
- 16-bit depth
- Stereo (2 channels)
- At least 5 seconds long

Just name them `jane_eyre_5s.wav` and `monte_cristo_5s.wav`.

### Option 3: Skip Real Audio Tests

If these files are not present, the test generator will skip real audio tests
and only generate synthetic test cases. This is fine for basic testing.

## License

All LibriVox recordings are in the **public domain** and can be freely used
for any purpose including testing.

Source: https://librivox.org/
