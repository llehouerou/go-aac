package tables

import "testing"

func TestNumSWB128Window(t *testing.T) {
	expected := [12]uint8{12, 12, 12, 14, 14, 14, 15, 15, 15, 15, 15, 15}
	for i := 0; i < 12; i++ {
		if NumSWB128Window[i] != expected[i] {
			t.Errorf("NumSWB128Window[%d] = %d, want %d", i, NumSWB128Window[i], expected[i])
		}
	}
}

func TestSWBOffset128_96(t *testing.T) {
	expected := []uint16{0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 92, 128}
	if len(SWBOffset128_96) != len(expected) {
		t.Fatalf("len(SWBOffset128_96) = %d, want %d", len(SWBOffset128_96), len(expected))
	}
	for i, v := range expected {
		if SWBOffset128_96[i] != v {
			t.Errorf("SWBOffset128_96[%d] = %d, want %d", i, SWBOffset128_96[i], v)
		}
	}
}

func TestSWBOffset128_64(t *testing.T) {
	expected := []uint16{0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 92, 128}
	if len(SWBOffset128_64) != len(expected) {
		t.Fatalf("len(SWBOffset128_64) = %d, want %d", len(SWBOffset128_64), len(expected))
	}
	for i, v := range expected {
		if SWBOffset128_64[i] != v {
			t.Errorf("SWBOffset128_64[%d] = %d, want %d", i, SWBOffset128_64[i], v)
		}
	}
}

func TestSWBOffset128_48(t *testing.T) {
	expected := []uint16{0, 4, 8, 12, 16, 20, 28, 36, 44, 56, 68, 80, 96, 112, 128}
	if len(SWBOffset128_48) != len(expected) {
		t.Fatalf("len(SWBOffset128_48) = %d, want %d", len(SWBOffset128_48), len(expected))
	}
	for i, v := range expected {
		if SWBOffset128_48[i] != v {
			t.Errorf("SWBOffset128_48[%d] = %d, want %d", i, SWBOffset128_48[i], v)
		}
	}
}

func TestSWBOffset128_24(t *testing.T) {
	expected := []uint16{0, 4, 8, 12, 16, 20, 24, 28, 36, 44, 52, 64, 76, 92, 108, 128}
	if len(SWBOffset128_24) != len(expected) {
		t.Fatalf("len(SWBOffset128_24) = %d, want %d", len(SWBOffset128_24), len(expected))
	}
	for i, v := range expected {
		if SWBOffset128_24[i] != v {
			t.Errorf("SWBOffset128_24[%d] = %d, want %d", i, SWBOffset128_24[i], v)
		}
	}
}

func TestSWBOffset128_16(t *testing.T) {
	expected := []uint16{0, 4, 8, 12, 16, 20, 24, 28, 32, 40, 48, 60, 72, 88, 108, 128}
	if len(SWBOffset128_16) != len(expected) {
		t.Fatalf("len(SWBOffset128_16) = %d, want %d", len(SWBOffset128_16), len(expected))
	}
	for i, v := range expected {
		if SWBOffset128_16[i] != v {
			t.Errorf("SWBOffset128_16[%d] = %d, want %d", i, SWBOffset128_16[i], v)
		}
	}
}

func TestSWBOffset128_8(t *testing.T) {
	expected := []uint16{0, 4, 8, 12, 16, 20, 24, 28, 36, 44, 52, 60, 72, 88, 108, 128}
	if len(SWBOffset128_8) != len(expected) {
		t.Fatalf("len(SWBOffset128_8) = %d, want %d", len(SWBOffset128_8), len(expected))
	}
	for i, v := range expected {
		if SWBOffset128_8[i] != v {
			t.Errorf("SWBOffset128_8[%d] = %d, want %d", i, SWBOffset128_8[i], v)
		}
	}
}

func TestNumSWB128MatchesTableLength(t *testing.T) {
	for i := 0; i < 12; i++ {
		numSWB := NumSWB128Window[i]
		tableLen := len(SWBOffset128Window[i])
		// NumSWB should equal table length - 1 (table includes both start and end offsets)
		if int(numSWB) != tableLen-1 {
			t.Errorf("Sample rate index %d: NumSWB128Window=%d but SWBOffset128Window has %d entries (expected %d)",
				i, numSWB, tableLen, numSWB+1)
		}
	}
}

func TestSWBOffset128Window(t *testing.T) {
	tests := []struct {
		index    int
		expected *uint16
	}{
		{0, &SWBOffset128_96[0]},
		{1, &SWBOffset128_96[0]},
		{2, &SWBOffset128_64[0]},
		{3, &SWBOffset128_48[0]},
		{4, &SWBOffset128_48[0]},
		{5, &SWBOffset128_48[0]},
		{6, &SWBOffset128_24[0]},
		{7, &SWBOffset128_24[0]},
		{8, &SWBOffset128_16[0]},
		{9, &SWBOffset128_16[0]},
		{10, &SWBOffset128_16[0]},
		{11, &SWBOffset128_8[0]},
	}

	for _, tt := range tests {
		actual := &SWBOffset128Window[tt.index][0]
		if actual != tt.expected {
			t.Errorf("SWBOffset128Window[%d] points to wrong table", tt.index)
		}
	}
}
