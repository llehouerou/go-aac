// Package aac provides a pure Go AAC (Advanced Audio Coding) decoder.
//
// This package is a port of FAAD2 (Freeware Advanced Audio Decoder) from C to Go,
// providing AAC decoding without CGO dependencies.
//
// # Basic Usage
//
// To decode an AAC stream:
//
//	dec := aac.NewDecoder()
//	defer dec.Close()
//
//	// Initialize from ADTS stream
//	sampleRate, channels, err := dec.SimpleInit(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Decode frames
//	for {
//	    samples, err := dec.DecodeInt16(frameData)
//	    if err != nil {
//	        break
//	    }
//	    // Use samples...
//	}
//
// # API Variants
//
// The package provides two API styles:
//
// Simplified API (matching MIGRATION_STEPS.md):
//   - SimpleInit, SimpleInit2: Return (sampleRate, channels, error)
//   - DecodeInt16, DecodeFloat32: Return (samples, error)
//
// Detailed API (matching FAAD2):
//   - Init, Init2: Return (InitResult, error) with BytesConsumed
//   - Decode, DecodeFloat: Return (samples, *FrameInfo, error)
//
// # Supported Formats
//
// Object Types: AAC-LC, Main, LTP, LD, Error Resilient LC/LTP
// Container Formats: ADTS, Raw AAC (via Init2/AudioSpecificConfig)
// Output Formats: 16-bit, 24-bit, 32-bit integer; 32/64-bit float
//
// HE-AAC (SBR) and HE-AACv2 (PS) support is planned for future releases.
//
// # Thread Safety
//
// Decoder instances are NOT safe for concurrent use. Each goroutine should
// have its own Decoder. Read-only accessors (SampleRate, Channels, etc.)
// are safe to call concurrently after Init.
//
// # Reference
//
// Ported from FAAD2: https://github.com/knik0/faad2
package aac
