// internal/syntax/tns_data_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestParseTNSData_LongWindow_SingleFilter(t *testing.T) {
	// Long window with 1 filter, order 4
	// Bit layout:
	// n_filt[0]: 1 (2 bits) = 0b01
	// coef_res: 1 (1 bit) = 0b1 -> 4-bit coefficients
	// length: 20 (6 bits) = 0b010100
	// order: 4 (5 bits) = 0b00100
	// direction: 0 (1 bit) = 0b0
	// coef_compress: 0 (1 bit) = 0b0
	// coef[0-3]: 1, 2, 3, 4 (each 4 bits) = 0b0001_0010_0011_0100
	//
	// Binary: 01 1 010100 00100 0 0 0001 0010 0011 0100
	// Grouped: 0110_1010 0001_0000 0001_0010 0011_0100
	// Hex:     0x6A       0x10      0x12      0x34
	data := []byte{0x6A, 0x10, 0x12, 0x34}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		NumWindows:     1,
	}
	tns := &TNSInfo{}

	ParseTNSData(r, ics, tns)

	if tns.NFilt[0] != 1 {
		t.Errorf("NFilt[0]: got %d, want 1", tns.NFilt[0])
	}
	if tns.CoefRes[0] != 1 {
		t.Errorf("CoefRes[0]: got %d, want 1", tns.CoefRes[0])
	}
	if tns.Length[0][0] != 20 {
		t.Errorf("Length[0][0]: got %d, want 20", tns.Length[0][0])
	}
	if tns.Order[0][0] != 4 {
		t.Errorf("Order[0][0]: got %d, want 4", tns.Order[0][0])
	}
	if tns.Direction[0][0] != 0 {
		t.Errorf("Direction[0][0]: got %d, want 0", tns.Direction[0][0])
	}
	if tns.CoefCompress[0][0] != 0 {
		t.Errorf("CoefCompress[0][0]: got %d, want 0", tns.CoefCompress[0][0])
	}
	// Check coefficients
	expectedCoefs := []uint8{1, 2, 3, 4}
	for i, want := range expectedCoefs {
		if tns.Coef[0][0][i] != want {
			t.Errorf("Coef[0][0][%d]: got %d, want %d", i, tns.Coef[0][0][i], want)
		}
	}
}

func TestParseTNSData_LongWindow_NoFilter(t *testing.T) {
	// Long window with 0 filters
	// n_filt[0]: 0 (2 bits) = 0b00
	data := []byte{0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		NumWindows:     1,
	}
	tns := &TNSInfo{}

	ParseTNSData(r, ics, tns)

	if tns.NFilt[0] != 0 {
		t.Errorf("NFilt[0]: got %d, want 0", tns.NFilt[0])
	}
}

func TestParseTNSData_LongWindow_TwoFilters(t *testing.T) {
	// Long window with 2 filters
	// n_filt[0]: 2 (2 bits) = 0b10
	// coef_res: 0 (1 bit) = 0b0 -> 3-bit coefficients
	// Filter 0:
	//   length: 10 (6 bits) = 0b001010
	//   order: 2 (5 bits) = 0b00010
	//   direction: 1 (1 bit) = 0b1
	//   coef_compress: 0 (1 bit) = 0b0
	//   coef[0-1]: 3, 5 (each 3 bits) = 0b011_101
	// Filter 1:
	//   length: 15 (6 bits) = 0b001111
	//   order: 0 (5 bits) = 0b00000 (no coefficients)
	//
	// Binary: 10 0 001010 00010 1 0 011 101 001111 00000
	// Grouped: 1000_0101 0000_1010 0111_0100 1111_0000 0xxx_xxxx
	// Hex:     0x85      0x0A      0x74      0xF0      ...
	data := []byte{0x85, 0x0A, 0x74, 0xF0, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence: LongStartSequence,
		NumWindows:     1,
	}
	tns := &TNSInfo{}

	ParseTNSData(r, ics, tns)

	if tns.NFilt[0] != 2 {
		t.Errorf("NFilt[0]: got %d, want 2", tns.NFilt[0])
	}
	if tns.CoefRes[0] != 0 {
		t.Errorf("CoefRes[0]: got %d, want 0", tns.CoefRes[0])
	}

	// Filter 0
	if tns.Length[0][0] != 10 {
		t.Errorf("Length[0][0]: got %d, want 10", tns.Length[0][0])
	}
	if tns.Order[0][0] != 2 {
		t.Errorf("Order[0][0]: got %d, want 2", tns.Order[0][0])
	}
	if tns.Direction[0][0] != 1 {
		t.Errorf("Direction[0][0]: got %d, want 1", tns.Direction[0][0])
	}
	if tns.CoefCompress[0][0] != 0 {
		t.Errorf("CoefCompress[0][0]: got %d, want 0", tns.CoefCompress[0][0])
	}
	if tns.Coef[0][0][0] != 3 {
		t.Errorf("Coef[0][0][0]: got %d, want 3", tns.Coef[0][0][0])
	}
	if tns.Coef[0][0][1] != 5 {
		t.Errorf("Coef[0][0][1]: got %d, want 5", tns.Coef[0][0][1])
	}

	// Filter 1 (order 0, no coefficients)
	if tns.Length[0][1] != 15 {
		t.Errorf("Length[0][1]: got %d, want 15", tns.Length[0][1])
	}
	if tns.Order[0][1] != 0 {
		t.Errorf("Order[0][1]: got %d, want 0", tns.Order[0][1])
	}
}

func TestParseTNSData_ShortWindow(t *testing.T) {
	// 8 short windows, each with 1 filter
	// For short windows: n_filt=1bit, length=4bits, order=3bits
	//
	// Window 0:
	//   n_filt: 1 (1 bit)
	//   coef_res: 1 (1 bit) -> 4-bit coefficients
	//   length: 5 (4 bits)
	//   order: 3 (3 bits)
	//   direction: 0 (1 bit)
	//   coef_compress: 1 (1 bit) -> 3-bit coefficients (4-1)
	//   coef[0-2]: 1, 2, 3 (each 3 bits)
	// Windows 1-7: n_filt: 0 (1 bit each)
	//
	// Binary: 1 1 0101 011 0 1 001 010 011 0 0 0 0 0 0 0
	// Grouped: 1101_0101 1010_0101 0011_0000 0000_xxxx
	// Hex:     0xD5      0xA5      0x30      0x00
	data := []byte{0xD5, 0xA5, 0x30, 0x00}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence: EightShortSequence,
		NumWindows:     8,
	}
	tns := &TNSInfo{}

	ParseTNSData(r, ics, tns)

	// Window 0
	if tns.NFilt[0] != 1 {
		t.Errorf("NFilt[0]: got %d, want 1", tns.NFilt[0])
	}
	if tns.CoefRes[0] != 1 {
		t.Errorf("CoefRes[0]: got %d, want 1", tns.CoefRes[0])
	}
	if tns.Length[0][0] != 5 {
		t.Errorf("Length[0][0]: got %d, want 5", tns.Length[0][0])
	}
	if tns.Order[0][0] != 3 {
		t.Errorf("Order[0][0]: got %d, want 3", tns.Order[0][0])
	}
	if tns.Direction[0][0] != 0 {
		t.Errorf("Direction[0][0]: got %d, want 0", tns.Direction[0][0])
	}
	if tns.CoefCompress[0][0] != 1 {
		t.Errorf("CoefCompress[0][0]: got %d, want 1", tns.CoefCompress[0][0])
	}
	// With coef_compress=1, we use 3 bits (4-1)
	expectedCoefs := []uint8{1, 2, 3}
	for i, want := range expectedCoefs {
		if tns.Coef[0][0][i] != want {
			t.Errorf("Coef[0][0][%d]: got %d, want %d", i, tns.Coef[0][0][i], want)
		}
	}

	// Windows 1-7 should have 0 filters
	for w := 1; w < 8; w++ {
		if tns.NFilt[w] != 0 {
			t.Errorf("NFilt[%d]: got %d, want 0", w, tns.NFilt[w])
		}
	}
}

func TestParseTNSData_CoefCompress(t *testing.T) {
	// Test that coef_compress reduces coefficient bit width
	// Long window, 1 filter, coef_res=0 (3 bits), coef_compress=1 (2 bits)
	//
	// n_filt: 1 (2 bits)
	// coef_res: 0 (1 bit) -> start with 3-bit coefs
	// length: 8 (6 bits) = 0b001000
	// order: 2 (5 bits) = 0b00010
	// direction: 0 (1 bit)
	// coef_compress: 1 (1 bit) -> 2-bit coefs (3-1)
	// coef[0-1]: 1, 2 (each 2 bits) = 0b01_10
	//
	// Binary: 01 0 001000 00010 0 1 01 10 xxxx
	// Grouped: 0100_0100 0000_1001 0110_xxxx
	// Hex:     0x44      0x09      0x60
	data := []byte{0x44, 0x09, 0x60}
	r := bits.NewReader(data)

	ics := &ICStream{
		WindowSequence: OnlyLongSequence,
		NumWindows:     1,
	}
	tns := &TNSInfo{}

	ParseTNSData(r, ics, tns)

	if tns.NFilt[0] != 1 {
		t.Errorf("NFilt[0]: got %d, want 1", tns.NFilt[0])
	}
	if tns.CoefRes[0] != 0 {
		t.Errorf("CoefRes[0]: got %d, want 0", tns.CoefRes[0])
	}
	if tns.Order[0][0] != 2 {
		t.Errorf("Order[0][0]: got %d, want 2", tns.Order[0][0])
	}
	if tns.CoefCompress[0][0] != 1 {
		t.Errorf("CoefCompress[0][0]: got %d, want 1", tns.CoefCompress[0][0])
	}
	// 2-bit coefficients: 1 and 2
	if tns.Coef[0][0][0] != 1 {
		t.Errorf("Coef[0][0][0]: got %d, want 1", tns.Coef[0][0][0])
	}
	if tns.Coef[0][0][1] != 2 {
		t.Errorf("Coef[0][0][1]: got %d, want 2", tns.Coef[0][0][1])
	}
}
