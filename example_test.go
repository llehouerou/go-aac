package aac_test

import (
	"fmt"

	"github.com/llehouerou/go-aac"
)

func Example() {
	// Create a new decoder
	dec := aac.NewDecoder()
	defer dec.Close()

	// Check library capabilities
	caps := aac.GetCapabilities()
	fmt.Printf("Supports AAC-LC: %v\n", caps&aac.CapabilityLC != 0)

	// Initialize from ADTS data (in real usage, this would be from a file)
	// Using a minimal valid ADTS header for the example
	adtsHeader := []byte{0xFF, 0xF1, 0x50, 0x80, 0x00, 0x1F, 0xFC}

	sampleRate, channels, err := dec.SimpleInit(adtsHeader)
	if err != nil {
		fmt.Printf("Init error: %v\n", err)
		return
	}

	fmt.Printf("Sample rate: %d Hz\n", sampleRate)
	fmt.Printf("Channels: %d\n", channels)

	// Output:
	// Supports AAC-LC: true
	// Sample rate: 44100 Hz
	// Channels: 2
}

func ExampleDecoder_DecodeInt16() {
	dec := aac.NewDecoder()
	defer dec.Close()

	// Initialize (simplified example)
	adtsData := []byte{0xFF, 0xF1, 0x50, 0x80, 0x00, 0x1F, 0xFC}
	_, _, _ = dec.SimpleInit(adtsData)

	// Decode would return samples for complete AAC frames.
	// Note: This example uses a minimal ADTS header without actual frame data,
	// which will cause a decode error. In production, use complete AAC frames.
	samples, err := dec.DecodeInt16(adtsData)
	if err != nil {
		// Expected: decoding incomplete frame data returns an error
		fmt.Println("Decode returned error (expected for incomplete data)")
	}
	if samples == nil {
		fmt.Println("No samples returned")
	}

	// Output:
	// Decode returned error (expected for incomplete data)
	// No samples returned
}
