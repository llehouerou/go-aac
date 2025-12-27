package tables

import "testing"

func TestNumSWB1024Window(t *testing.T) {
	expected := [12]uint8{41, 41, 47, 49, 49, 51, 47, 47, 43, 43, 43, 40}
	for i := 0; i < 12; i++ {
		if NumSWB1024Window[i] != expected[i] {
			t.Errorf("NumSWB1024Window[%d] = %d, want %d", i, NumSWB1024Window[i], expected[i])
		}
	}
}

func TestNumSWB960Window(t *testing.T) {
	expected := [12]uint8{40, 40, 45, 49, 49, 49, 46, 46, 42, 42, 42, 40}
	for i := 0; i < 12; i++ {
		if NumSWB960Window[i] != expected[i] {
			t.Errorf("NumSWB960Window[%d] = %d, want %d", i, NumSWB960Window[i], expected[i])
		}
	}
}

func TestSWBOffset1024_96(t *testing.T) {
	expected := []uint16{
		0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56,
		64, 72, 80, 88, 96, 108, 120, 132, 144, 156, 172, 188, 212, 240,
		276, 320, 384, 448, 512, 576, 640, 704, 768, 832, 896, 960, 1024,
	}
	if len(SWBOffset1024_96) != len(expected) {
		t.Fatalf("len(SWBOffset1024_96) = %d, want %d", len(SWBOffset1024_96), len(expected))
	}
	for i, v := range expected {
		if SWBOffset1024_96[i] != v {
			t.Errorf("SWBOffset1024_96[%d] = %d, want %d", i, SWBOffset1024_96[i], v)
		}
	}
}

func TestSWBOffset1024_64(t *testing.T) {
	expected := []uint16{
		0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56,
		64, 72, 80, 88, 100, 112, 124, 140, 156, 172, 192, 216, 240, 268,
		304, 344, 384, 424, 464, 504, 544, 584, 624, 664, 704, 744, 784, 824,
		864, 904, 944, 984, 1024,
	}
	if len(SWBOffset1024_64) != len(expected) {
		t.Fatalf("len(SWBOffset1024_64) = %d, want %d", len(SWBOffset1024_64), len(expected))
	}
	for i, v := range expected {
		if SWBOffset1024_64[i] != v {
			t.Errorf("SWBOffset1024_64[%d] = %d, want %d", i, SWBOffset1024_64[i], v)
		}
	}
}

func TestSWBOffset1024_48(t *testing.T) {
	expected := []uint16{
		0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 48, 56, 64, 72,
		80, 88, 96, 108, 120, 132, 144, 160, 176, 196, 216, 240, 264, 292,
		320, 352, 384, 416, 448, 480, 512, 544, 576, 608, 640, 672, 704, 736,
		768, 800, 832, 864, 896, 928, 1024,
	}
	if len(SWBOffset1024_48) != len(expected) {
		t.Fatalf("len(SWBOffset1024_48) = %d, want %d", len(SWBOffset1024_48), len(expected))
	}
	for i, v := range expected {
		if SWBOffset1024_48[i] != v {
			t.Errorf("SWBOffset1024_48[%d] = %d, want %d", i, SWBOffset1024_48[i], v)
		}
	}
}

func TestSWBOffset1024_32(t *testing.T) {
	expected := []uint16{
		0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 48, 56, 64, 72,
		80, 88, 96, 108, 120, 132, 144, 160, 176, 196, 216, 240, 264, 292,
		320, 352, 384, 416, 448, 480, 512, 544, 576, 608, 640, 672, 704, 736,
		768, 800, 832, 864, 896, 928, 960, 992, 1024,
	}
	if len(SWBOffset1024_32) != len(expected) {
		t.Fatalf("len(SWBOffset1024_32) = %d, want %d", len(SWBOffset1024_32), len(expected))
	}
	for i, v := range expected {
		if SWBOffset1024_32[i] != v {
			t.Errorf("SWBOffset1024_32[%d] = %d, want %d", i, SWBOffset1024_32[i], v)
		}
	}
}

func TestSWBOffset1024_24(t *testing.T) {
	expected := []uint16{
		0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 52, 60, 68,
		76, 84, 92, 100, 108, 116, 124, 136, 148, 160, 172, 188, 204, 220,
		240, 260, 284, 308, 336, 364, 396, 432, 468, 508, 552, 600, 652, 704,
		768, 832, 896, 960, 1024,
	}
	if len(SWBOffset1024_24) != len(expected) {
		t.Fatalf("len(SWBOffset1024_24) = %d, want %d", len(SWBOffset1024_24), len(expected))
	}
	for i, v := range expected {
		if SWBOffset1024_24[i] != v {
			t.Errorf("SWBOffset1024_24[%d] = %d, want %d", i, SWBOffset1024_24[i], v)
		}
	}
}

func TestSWBOffset1024_16(t *testing.T) {
	expected := []uint16{
		0, 8, 16, 24, 32, 40, 48, 56, 64, 72, 80, 88, 100, 112, 124,
		136, 148, 160, 172, 184, 196, 212, 228, 244, 260, 280, 300, 320, 344,
		368, 396, 424, 456, 492, 532, 572, 616, 664, 716, 772, 832, 896, 960, 1024,
	}
	if len(SWBOffset1024_16) != len(expected) {
		t.Fatalf("len(SWBOffset1024_16) = %d, want %d", len(SWBOffset1024_16), len(expected))
	}
	for i, v := range expected {
		if SWBOffset1024_16[i] != v {
			t.Errorf("SWBOffset1024_16[%d] = %d, want %d", i, SWBOffset1024_16[i], v)
		}
	}
}

func TestSWBOffset1024_8(t *testing.T) {
	expected := []uint16{
		0, 12, 24, 36, 48, 60, 72, 84, 96, 108, 120, 132, 144, 156, 172,
		188, 204, 220, 236, 252, 268, 288, 308, 328, 348, 372, 396, 420, 448,
		476, 508, 544, 580, 620, 664, 712, 764, 820, 880, 944, 1024,
	}
	if len(SWBOffset1024_8) != len(expected) {
		t.Fatalf("len(SWBOffset1024_8) = %d, want %d", len(SWBOffset1024_8), len(expected))
	}
	for i, v := range expected {
		if SWBOffset1024_8[i] != v {
			t.Errorf("SWBOffset1024_8[%d] = %d, want %d", i, SWBOffset1024_8[i], v)
		}
	}
}

func TestSWBOffset1024Window(t *testing.T) {
	// Verify lookup array maps correctly
	testCases := []struct {
		index    int
		expected []uint16
		name     string
	}{
		{0, SWBOffset1024_96, "96000"},
		{1, SWBOffset1024_96, "88200"},
		{2, SWBOffset1024_64, "64000"},
		{3, SWBOffset1024_48, "48000"},
		{4, SWBOffset1024_48, "44100"},
		{5, SWBOffset1024_32, "32000"},
		{6, SWBOffset1024_24, "24000"},
		{7, SWBOffset1024_24, "22050"},
		{8, SWBOffset1024_16, "16000"},
		{9, SWBOffset1024_16, "12000"},
		{10, SWBOffset1024_16, "11025"},
		{11, SWBOffset1024_8, "8000"},
	}

	for _, tc := range testCases {
		if len(SWBOffset1024Window[tc.index]) != len(tc.expected) {
			t.Errorf("SWBOffset1024Window[%d] (%s): len = %d, want %d",
				tc.index, tc.name, len(SWBOffset1024Window[tc.index]), len(tc.expected))
		}
		// Verify it's the same slice (pointer comparison)
		if &SWBOffset1024Window[tc.index][0] != &tc.expected[0] {
			t.Errorf("SWBOffset1024Window[%d] (%s) does not point to expected table",
				tc.index, tc.name)
		}
	}
}

func TestNumSWBMatchesTableLength(t *testing.T) {
	// Verify NumSWB1024Window[i] == len(SWBOffset1024Window[i]) - 1
	// The -1 is because the offset table includes both start and end offsets
	for i := 0; i < 12; i++ {
		expectedBands := len(SWBOffset1024Window[i]) - 1
		if int(NumSWB1024Window[i]) != expectedBands {
			t.Errorf("NumSWB1024Window[%d] = %d, but len(SWBOffset1024Window[%d])-1 = %d",
				i, NumSWB1024Window[i], i, expectedBands)
		}
	}
}
