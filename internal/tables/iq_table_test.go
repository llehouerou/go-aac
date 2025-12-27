// internal/tables/iq_table_test.go
package tables

import "testing"

func TestIQTable_Size(t *testing.T) {
	if len(IQTable) != IQTableSize {
		t.Errorf("IQTable size: got %d, want %d", len(IQTable), IQTableSize)
	}
}

func TestIQTable_FirstValues(t *testing.T) {
	// Known values from FAAD2's iq_table.h
	expected := []float64{
		0,
		1,
		2.5198420997897464,
		4.3267487109222245,
		6.3496042078727974,
		8.5498797333834844,
		10.902723556992836,
		13.390518279406722,
		15.999999999999998,
		18.720754407467133,
	}

	for i, want := range expected {
		got := IQTable[i]
		if got != want {
			t.Errorf("IQTable[%d]: got %v, want %v", i, got, want)
		}
	}
}
