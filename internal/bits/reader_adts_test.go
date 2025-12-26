package bits

import (
	"os"
	"testing"
)

// TestReader_ADTSHeaderParsing validates bit reading against known ADTS header structure.
// ADTS header is 56 bits (7 bytes) without CRC, 72 bits (9 bytes) with CRC.
//
// Reference: ISO/IEC 13818-7 section 6.2.1
func TestReader_ADTSHeaderParsing(t *testing.T) {
	// Read test file if available
	data, err := os.ReadFile("../../testdata/sine1k.aac")
	if err != nil {
		t.Skipf("Test file not available: %v", err)
	}

	r := NewReader(data)

	// Parse ADTS header fields per spec
	// https://wiki.multimedia.cx/index.php/ADTS
	syncword := r.GetBits(12)
	if syncword != 0xFFF {
		t.Fatalf("Invalid syncword: 0x%X, expected 0xFFF", syncword)
	}

	id := r.Get1Bit()               // 0 = MPEG-4, 1 = MPEG-2
	layer := r.GetBits(2)           // Always 0
	protectionAbsent := r.Get1Bit() // 1 = no CRC
	profile := r.GetBits(2)         // 0=Main, 1=LC, 2=SSR, 3=reserved
	sfIndex := r.GetBits(4)         // Sample rate index
	privateBit := r.Get1Bit()       // Ignored
	channelConfig := r.GetBits(3)   // Channel configuration
	original := r.Get1Bit()         // Ignored
	home := r.Get1Bit()             // Ignored
	copyrightIDBit := r.Get1Bit()   // Ignored
	copyrightIDStart := r.Get1Bit() // Ignored
	frameLength := r.GetBits(13)    // Frame length including header
	bufferFullness := r.GetBits(11) // Buffer fullness
	numRawBlocks := r.GetBits(2)    // Number of raw data blocks - 1

	// Validate parsed values
	t.Logf("ADTS Header parsed:")
	t.Logf("  ID (MPEG version): %d", id)
	t.Logf("  Layer: %d (should be 0)", layer)
	t.Logf("  Protection absent: %d", protectionAbsent)
	t.Logf("  Profile: %d (1=LC)", profile)
	t.Logf("  Sample rate index: %d", sfIndex)
	t.Logf("  Private bit: %d", privateBit)
	t.Logf("  Channel config: %d", channelConfig)
	t.Logf("  Original: %d", original)
	t.Logf("  Home: %d", home)
	t.Logf("  Copyright ID bit: %d", copyrightIDBit)
	t.Logf("  Copyright ID start: %d", copyrightIDStart)
	t.Logf("  Frame length: %d bytes", frameLength)
	t.Logf("  Buffer fullness: %d", bufferFullness)
	t.Logf("  Num raw blocks: %d", numRawBlocks)

	// Basic validity checks
	if layer != 0 {
		t.Errorf("Layer should be 0, got %d", layer)
	}

	if profile > 3 {
		t.Errorf("Profile out of range: %d", profile)
	}

	if sfIndex > 12 {
		t.Errorf("Sample rate index out of range: %d", sfIndex)
	}

	if channelConfig > 7 {
		t.Errorf("Channel config out of range: %d", channelConfig)
	}

	// Frame length should be reasonable (header + some data)
	if frameLength < 7 || frameLength > 8192 {
		t.Errorf("Frame length suspicious: %d", frameLength)
	}

	// Check we consumed exactly 56 bits
	consumed := r.GetProcessedBits()
	if consumed != 56 {
		t.Errorf("Consumed %d bits, expected 56 for ADTS header", consumed)
	}
}

// TestReader_ADTSMultipleFrames parses multiple ADTS frames to validate
// buffer handling across frame boundaries.
func TestReader_ADTSMultipleFrames(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sine1k.aac")
	if err != nil {
		t.Skipf("Test file not available: %v", err)
	}

	offset := 0
	frameCount := 0
	maxFrames := 5

	for offset < len(data) && frameCount < maxFrames {
		if len(data)-offset < 7 {
			break // Not enough data for header
		}

		r := NewReader(data[offset:])

		// Check syncword
		syncword := r.GetBits(12)
		if syncword != 0xFFF {
			t.Fatalf("Frame %d: Invalid syncword at offset %d: 0x%X",
				frameCount, offset, syncword)
		}

		// Skip to frame length field
		// Bit counts per field:
		//   id=1, layer=2, protection_absent=1 = 4 bits
		//   profile=2, sf_index=4, private_bit=1, channel_config=3 = 10 bits
		//   original=1, home=1, copyright_id_bit=1, copyright_id_start=1 = 4 bits
		//   frame_length=13 bits
		_ = r.GetBits(4)  // id, layer, protection_absent
		_ = r.GetBits(10) // profile, sf_index, private, channel_config
		_ = r.GetBits(4)  // original, home, copyright bits
		frameLength := r.GetBits(13)

		t.Logf("Frame %d at offset %d: length=%d bytes", frameCount, offset, frameLength)

		if frameLength < 7 {
			t.Fatalf("Frame %d: Invalid frame length %d", frameCount, frameLength)
		}

		offset += int(frameLength)
		frameCount++
	}

	t.Logf("Successfully parsed %d ADTS frames", frameCount)
}

// TestReader_ADTSSampleRateIndex validates that sample rate index decoding
// matches expected values for common sample rates.
func TestReader_ADTSSampleRateIndex(t *testing.T) {
	// Sample rate index to Hz mapping per ISO/IEC 14496-3
	sampleRates := map[uint32]int{
		0:  96000,
		1:  88200,
		2:  64000,
		3:  48000,
		4:  44100,
		5:  32000,
		6:  24000,
		7:  22050,
		8:  16000,
		9:  12000,
		10: 11025,
		11: 8000,
		12: 7350,
	}

	data, err := os.ReadFile("../../testdata/sine1k.aac")
	if err != nil {
		t.Skipf("Test file not available: %v", err)
	}

	r := NewReader(data)

	// Skip to sample rate index (bits 16-19)
	_ = r.GetBits(12) // syncword
	_ = r.GetBits(4)  // id, layer, protection_absent
	_ = r.GetBits(2)  // profile
	sfIndex := r.GetBits(4)

	expectedRate, ok := sampleRates[sfIndex]
	if !ok {
		t.Errorf("Unknown sample rate index: %d", sfIndex)
	} else {
		t.Logf("Sample rate index %d = %d Hz", sfIndex, expectedRate)
		// Our test file was created with 44100 Hz (index 4)
		if sfIndex != 4 {
			t.Logf("Note: Expected sample rate index 4 (44100 Hz) for test file")
		}
	}
}

// TestReader_ADTSChannelConfig validates channel configuration parsing.
func TestReader_ADTSChannelConfig(t *testing.T) {
	// Channel config to channel count mapping per ISO/IEC 14496-3
	channelCounts := map[uint32]int{
		0: 0, // Defined in AOT Specific Config
		1: 1, // Mono
		2: 2, // Stereo
		3: 3, // Front center + Left/Right
		4: 4, // Front center + Left/Right + Rear center
		5: 5, // Front center + Left/Right + Rear Left/Right
		6: 6, // Front center + Left/Right + Rear Left/Right + LFE
		7: 8, // 7.1 surround
	}

	data, err := os.ReadFile("../../testdata/sine1k.aac")
	if err != nil {
		t.Skipf("Test file not available: %v", err)
	}

	r := NewReader(data)

	// Skip to channel config (bits 23-25)
	_ = r.GetBits(12) // syncword
	_ = r.GetBits(4)  // id, layer, protection_absent
	_ = r.GetBits(2)  // profile
	_ = r.GetBits(4)  // sample rate index
	_ = r.Get1Bit()   // private bit
	channelConfig := r.GetBits(3)

	channelCount, ok := channelCounts[channelConfig]
	if !ok {
		t.Errorf("Unknown channel config: %d", channelConfig)
	} else {
		t.Logf("Channel config %d = %d channels", channelConfig, channelCount)
		// Our test file was created with mono (config 1)
		if channelConfig != 1 {
			t.Logf("Note: Expected channel config 1 (mono) for test file")
		}
	}
}

// TestReader_ADTSProtectionAbsent validates CRC handling.
func TestReader_ADTSProtectionAbsent(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sine1k.aac")
	if err != nil {
		t.Skipf("Test file not available: %v", err)
	}

	r := NewReader(data)

	// Get protection_absent flag (bit 15)
	_ = r.GetBits(12) // syncword
	_ = r.Get1Bit()   // id
	_ = r.GetBits(2)  // layer
	protectionAbsent := r.Get1Bit()

	if protectionAbsent == 1 {
		t.Logf("No CRC present (protection_absent=1), header is 7 bytes")
	} else {
		t.Logf("CRC present (protection_absent=0), header is 9 bytes")
	}

	// FFmpeg typically creates files without CRC
	if protectionAbsent != 1 {
		t.Logf("Note: Expected protection_absent=1 (no CRC) for FFmpeg-generated file")
	}
}
