//go:build ignore

// This script generates test data for AAC decoder testing.
// Run with: go run testdata/generate.go
//
// Requirements: FFmpeg must be installed and available in PATH.
//
// Generated test data structure:
//   testdata/generated/
//   ├── aac_lc/           # AAC-LC profile tests
//   │   ├── 44100_16_mono/
//   │   ├── 44100_16_stereo/
//   │   └── ...
//   ├── he_aac/           # HE-AAC (SBR) tests
//   │   └── ...
//   ├── he_aac_v2/        # HE-AACv2 (SBR+PS) tests
//   │   └── ...
//   └── real_audio/       # Real audio samples
//       └── ...

package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
)

// Real audio samples from LibriVox (public domain)
// Source WAV files are stored in testdata/samples/
// Download instructions in testdata/samples/README.md
var realAudioSamples = []struct {
	name   string
	source string // relative to testdata/
}{
	// "Jane Eyre" by Charlotte Bronte, Chapter 1 (5 seconds)
	// LibriVox recording (public domain)
	{"librivox_jane_eyre", "samples/jane_eyre_5s.wav"},
	// "The Count of Monte Cristo" by Alexandre Dumas, Chapter 1 (5 seconds)
	// LibriVox recording (public domain)
	{"librivox_monte_cristo", "samples/monte_cristo_5s.wav"},
}

// TestConfig describes a test configuration
type TestConfig struct {
	SampleRate  int    `json:"sample_rate"`
	BitDepth    int    `json:"bit_depth"`    // Output bit depth (16 or 24)
	NumChannels int    `json:"num_channels"` // 1=mono, 2=stereo
	Profile     string `json:"profile"`      // "aac_lc", "he_aac", "he_aac_v2"
	Bitrate     int    `json:"bitrate"`      // Target bitrate in kbps
}

// AAC-LC configurations (baseline)
var aacLCConfigs = []TestConfig{
	// Standard configurations
	{44100, 16, 1, "aac_lc", 64},  // Mono 64kbps
	{44100, 16, 2, "aac_lc", 128}, // Stereo 128kbps
	{44100, 16, 2, "aac_lc", 256}, // Stereo high quality
	{48000, 16, 1, "aac_lc", 64},  // 48kHz mono
	{48000, 16, 2, "aac_lc", 128}, // 48kHz stereo
	{48000, 16, 2, "aac_lc", 320}, // 48kHz high bitrate
	{96000, 16, 2, "aac_lc", 256}, // High sample rate
	{22050, 16, 1, "aac_lc", 32},  // Low sample rate mono
	{22050, 16, 2, "aac_lc", 64},  // Low sample rate stereo
	{16000, 16, 1, "aac_lc", 24},  // Speech-like
}

// HE-AAC configurations (SBR)
var heAACConfigs = []TestConfig{
	{44100, 16, 2, "he_aac", 48}, // Low bitrate stereo
	{44100, 16, 2, "he_aac", 64}, // Medium bitrate
	{44100, 16, 1, "he_aac", 32}, // Mono
	{48000, 16, 2, "he_aac", 64}, // 48kHz
	{22050, 16, 2, "he_aac", 32}, // Low sample rate
}

// HE-AACv2 configurations (SBR + Parametric Stereo)
var heAACv2Configs = []TestConfig{
	{44100, 16, 2, "he_aac_v2", 24}, // Very low bitrate
	{44100, 16, 2, "he_aac_v2", 32}, // Low bitrate
	{44100, 16, 2, "he_aac_v2", 48}, // Medium bitrate
	{48000, 16, 2, "he_aac_v2", 32}, // 48kHz low bitrate
}

var audioTypes = []string{"silence", "sine1k", "sweep", "noise", "impulse", "speech_like"}

func main() {
	if err := checkFFmpeg(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please install FFmpeg: https://ffmpeg.org/download.html\n")
		os.Exit(1)
	}

	baseDir := filepath.Join("testdata", "generated")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Generate AAC-LC test cases
	fmt.Println("=== Generating AAC-LC test data ===")
	generateProfileTests(baseDir, "aac_lc", aacLCConfigs)

	// Generate HE-AAC test cases
	fmt.Println("\n=== Generating HE-AAC test data ===")
	generateProfileTests(baseDir, "he_aac", heAACConfigs)

	// Generate HE-AACv2 test cases
	fmt.Println("\n=== Generating HE-AACv2 test data ===")
	generateProfileTests(baseDir, "he_aac_v2", heAACv2Configs)

	// Generate real audio samples
	fmt.Println("\n=== Generating real audio test data ===")
	realDir := filepath.Join(baseDir, "real_audio")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating real audio directory: %v\n", err)
	} else {
		for _, sample := range realAudioSamples {
			sourcePath := filepath.Join("testdata", sample.source)
			if err := generateRealAudioTestCase(realDir, sample.name, sourcePath); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating real audio %s: %v\n", sample.name, err)
			} else {
				fmt.Printf("Generated real_audio/%s\n", sample.name)
			}
		}
	}

	fmt.Println("\nDone!")
}

func generateProfileTests(baseDir, profile string, configs []TestConfig) {
	profileDir := filepath.Join(baseDir, profile)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", profileDir, err)
		return
	}

	for _, cfg := range configs {
		dirName := fmt.Sprintf("%d_%d_%s_%dk",
			cfg.SampleRate, cfg.BitDepth, channelName(cfg.NumChannels), cfg.Bitrate)
		dir := filepath.Join(profileDir, dirName)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", dir, err)
			continue
		}

		for _, audioType := range audioTypes {
			if err := generateTestCase(dir, audioType, cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating %s/%s/%s: %v\n", profile, dirName, audioType, err)
			} else {
				fmt.Printf("Generated %s/%s/%s\n", profile, dirName, audioType)
			}
		}
	}
}

func checkFFmpeg() error {
	cmd := exec.Command("ffmpeg", "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg not found: %w", err)
	}
	return nil
}

func channelName(n int) string {
	if n == 1 {
		return "mono"
	}
	return "stereo"
}

func generateTestCase(dir, audioType string, cfg TestConfig) error {
	wavPath := filepath.Join(dir, audioType+".wav")
	aacPath := filepath.Join(dir, audioType+".aac") // ADTS format
	m4aPath := filepath.Join(dir, audioType+".m4a") // M4A container
	rawPath := filepath.Join(dir, audioType+".raw") // Reference PCM
	jsonPath := filepath.Join(dir, audioType+".json")

	// Skip if all files exist
	if fileExists(aacPath) && fileExists(m4aPath) && fileExists(rawPath) && fileExists(jsonPath) {
		return nil
	}

	// Generate WAV (1 second of audio)
	duration := 1.0
	samples := int(float64(cfg.SampleRate) * duration)
	if err := generateWAV(wavPath, audioType, cfg, samples); err != nil {
		return fmt.Errorf("generating WAV: %w", err)
	}

	// Encode to AAC (both ADTS and M4A formats)
	if err := encodeAAC(wavPath, aacPath, m4aPath, cfg); err != nil {
		return fmt.Errorf("encoding AAC: %w", err)
	}

	// Decode to raw PCM (reference output using M4A)
	if err := decodeToRaw(m4aPath, rawPath, cfg); err != nil {
		return fmt.Errorf("decoding to raw: %w", err)
	}

	// Write config JSON
	if err := writeConfig(jsonPath, cfg); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	// Clean up intermediate WAV
	os.Remove(wavPath)

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func generateWAV(path, audioType string, cfg TestConfig, samples int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	bytesPerSample := cfg.BitDepth / 8
	dataSize := samples * cfg.NumChannels * bytesPerSample

	// Write WAV header
	if err := writeWAVHeader(f, cfg, dataSize); err != nil {
		return err
	}

	// Generate and write samples
	for i := 0; i < samples; i++ {
		for ch := 0; ch < cfg.NumChannels; ch++ {
			var sample float64
			t := float64(i) / float64(cfg.SampleRate)

			switch audioType {
			case "silence":
				sample = 0

			case "sine1k":
				// Pure 1kHz sine wave
				sample = 0.8 * math.Sin(2*math.Pi*1000*t)

			case "sweep":
				// Logarithmic sweep from 20Hz to Nyquist/2
				maxFreq := float64(cfg.SampleRate) / 4
				progress := float64(i) / float64(samples)
				freq := 20 * math.Pow(maxFreq/20, progress)
				sample = 0.7 * math.Sin(2*math.Pi*freq*t)

			case "noise":
				// Pseudo-random noise using LCG (deterministic)
				seed := uint32(i*cfg.NumChannels + ch + 12345)
				seed = seed*1103515245 + 12345
				sample = float64(int32(seed)) / float64(math.MaxInt32) * 0.5

			case "impulse":
				// Periodic impulses (tests transient handling)
				period := cfg.SampleRate / 10 // 10 impulses per second
				if i%period == 0 {
					sample = 0.9
				} else {
					sample = 0
				}

			case "speech_like":
				// Combination of frequencies mimicking speech spectrum
				// Fundamental + harmonics with decreasing amplitude
				f0 := 150.0 // Fundamental frequency
				sample = 0.3 * math.Sin(2*math.Pi*f0*t)
				sample += 0.2 * math.Sin(2*math.Pi*2*f0*t)
				sample += 0.15 * math.Sin(2*math.Pi*3*f0*t)
				sample += 0.1 * math.Sin(2*math.Pi*4*f0*t)
				// Add some noise
				seed := uint32(i*cfg.NumChannels + ch + 54321)
				seed = seed*1103515245 + 12345
				noise := float64(int32(seed)) / float64(math.MaxInt32) * 0.05
				sample += noise
				// Amplitude modulation (syllable-like)
				envelope := 0.5 + 0.5*math.Sin(2*math.Pi*4*t)
				sample *= envelope
			}

			// Stereo: slight phase difference for spatial effect
			if cfg.NumChannels == 2 && ch == 1 && audioType != "silence" {
				// Add small delay effect for right channel
				sample *= 0.95
			}

			// Scale to bit depth and write
			writeSample(f, sample, cfg.BitDepth)
		}
	}

	return nil
}

func writeSample(f *os.File, sample float64, bitDepth int) {
	// Clamp to [-1, 1]
	if sample > 1.0 {
		sample = 1.0
	} else if sample < -1.0 {
		sample = -1.0
	}

	switch bitDepth {
	case 16:
		val := int16(sample * 32767)
		binary.Write(f, binary.LittleEndian, val)
	case 24:
		val := int32(sample * 8388607)
		// Write 24-bit as 3 bytes (little-endian)
		f.Write([]byte{byte(val), byte(val >> 8), byte(val >> 16)})
	case 32:
		val := int32(sample * 2147483647)
		binary.Write(f, binary.LittleEndian, val)
	}
}

func writeWAVHeader(f *os.File, cfg TestConfig, dataSize int) error {
	bytesPerSample := cfg.BitDepth / 8
	blockAlign := cfg.NumChannels * bytesPerSample
	byteRate := cfg.SampleRate * blockAlign

	// RIFF header
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	f.Write([]byte("WAVE"))

	// fmt chunk
	f.Write([]byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16)) // chunk size
	binary.Write(f, binary.LittleEndian, uint16(1))  // audio format (PCM)
	binary.Write(f, binary.LittleEndian, uint16(cfg.NumChannels))
	binary.Write(f, binary.LittleEndian, uint32(cfg.SampleRate))
	binary.Write(f, binary.LittleEndian, uint32(byteRate))
	binary.Write(f, binary.LittleEndian, uint16(blockAlign))
	binary.Write(f, binary.LittleEndian, uint16(cfg.BitDepth))

	// data chunk
	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataSize))

	return nil
}

func encodeAAC(wavPath, aacPath, m4aPath string, cfg TestConfig) error {
	// Determine encoder and profile options
	var encoder string
	var profileArgs []string

	switch cfg.Profile {
	case "aac_lc":
		encoder = "aac" // FFmpeg native AAC encoder (LC profile)
		profileArgs = []string{"-profile:a", "aac_low"}

	case "he_aac":
		// Use libfdk_aac if available, otherwise fall back to native
		if hasEncoder("libfdk_aac") {
			encoder = "libfdk_aac"
			profileArgs = []string{"-profile:a", "aac_he"}
		} else {
			// FFmpeg native doesn't support HE-AAC well, use LC as fallback
			fmt.Fprintf(os.Stderr, "Warning: libfdk_aac not available, using LC for HE-AAC test\n")
			encoder = "aac"
			profileArgs = []string{"-profile:a", "aac_low"}
		}

	case "he_aac_v2":
		// HE-AACv2 requires libfdk_aac
		if hasEncoder("libfdk_aac") {
			encoder = "libfdk_aac"
			profileArgs = []string{"-profile:a", "aac_he_v2"}
		} else {
			fmt.Fprintf(os.Stderr, "Warning: libfdk_aac not available, using LC for HE-AACv2 test\n")
			encoder = "aac"
			profileArgs = []string{"-profile:a", "aac_low"}
		}
	}

	bitrateArg := fmt.Sprintf("%dk", cfg.Bitrate)

	// Encode to ADTS (.aac)
	aacArgs := []string{"-y", "-i", wavPath, "-c:a", encoder}
	aacArgs = append(aacArgs, profileArgs...)
	aacArgs = append(aacArgs, "-b:a", bitrateArg, "-f", "adts", aacPath)

	cmd := exec.Command("ffmpeg", aacArgs...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("encoding ADTS: %w", err)
	}

	// Encode to M4A container
	m4aArgs := []string{"-y", "-i", wavPath, "-c:a", encoder}
	m4aArgs = append(m4aArgs, profileArgs...)
	m4aArgs = append(m4aArgs, "-b:a", bitrateArg, m4aPath)

	cmd = exec.Command("ffmpeg", m4aArgs...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("encoding M4A: %w", err)
	}

	return nil
}

func hasEncoder(encoder string) bool {
	cmd := exec.Command("ffmpeg", "-encoders")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return contains(string(output), encoder)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func decodeToRaw(m4aPath, rawPath string, cfg TestConfig) error {
	// Always decode to 16-bit PCM for comparison simplicity
	format := "s16le"

	cmd := exec.Command("ffmpeg", "-y", "-i", m4aPath,
		"-f", format, "-acodec", "pcm_"+format, rawPath)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeConfig(path string, cfg TestConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// generateRealAudioTestCase processes a local WAV sample into AAC test data
func generateRealAudioTestCase(dir, name, wavSourcePath string) error {
	aacPath := filepath.Join(dir, name+".aac")
	m4aPath := filepath.Join(dir, name+".m4a")
	rawPath := filepath.Join(dir, name+".raw")
	jsonPath := filepath.Join(dir, name+".json")

	// Skip if all files exist
	if fileExists(aacPath) && fileExists(m4aPath) && fileExists(rawPath) && fileExists(jsonPath) {
		return nil
	}

	// Check source exists
	if !fileExists(wavSourcePath) {
		return fmt.Errorf("source file not found: %s (see testdata/samples/README.md)", wavSourcePath)
	}

	// Default config for real audio (44.1kHz stereo, 128kbps LC)
	cfg := TestConfig{
		SampleRate:  44100,
		BitDepth:    16,
		NumChannels: 2,
		Profile:     "aac_lc",
		Bitrate:     128,
	}

	// Encode to AAC
	if err := encodeAAC(wavSourcePath, aacPath, m4aPath, cfg); err != nil {
		return fmt.Errorf("encoding AAC: %w", err)
	}

	// Decode to raw PCM (reference)
	if err := decodeToRaw(m4aPath, rawPath, cfg); err != nil {
		return fmt.Errorf("decoding to raw: %w", err)
	}

	// Write config
	if err := writeConfig(jsonPath, cfg); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
