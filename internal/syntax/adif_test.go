package syntax

import "testing"

func TestADIFHeader_Fields(t *testing.T) {
	var h ADIFHeader

	h.CopyrightIDPresent = false
	h.OriginalCopy = false
	h.Bitrate = 0
	h.ADIFBufferFullness = 0
	h.NumProgramConfigElements = 0
	h.Home = false
	h.BitstreamType = 0
}

func TestADIFHeader_CopyrightID(t *testing.T) {
	var h ADIFHeader

	// Copyright ID is 10 bytes (per FAAD2 structs.h:173)
	if len(h.CopyrightID) != 10 {
		t.Errorf("CopyrightID should have 10 bytes, got %d", len(h.CopyrightID))
	}
}

func TestADIFHeader_PCEs(t *testing.T) {
	var h ADIFHeader

	// Up to 16 PCEs
	if len(h.PCE) != 16 {
		t.Errorf("PCE should have 16 elements, got %d", len(h.PCE))
	}

	// Each PCE should be a ProgramConfig
	h.PCE[0].Channels = 2
}
