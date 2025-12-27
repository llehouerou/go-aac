// internal/syntax/element_test.go
package syntax

import "testing"

func TestElement_Fields(t *testing.T) {
	var e Element

	// Core fields
	e.Channel = 0
	e.PairedChannel = -1 // -1 indicates no pair
	e.ElementInstanceTag = 0
	e.CommonWindow = false
}

func TestElement_ICStreams(t *testing.T) {
	var e Element

	// Should have two ICStream fields for CPE
	e.ICS1.GlobalGain = 100
	e.ICS2.GlobalGain = 100

	if e.ICS1.GlobalGain != 100 || e.ICS2.GlobalGain != 100 {
		t.Error("ICStream fields not accessible")
	}
}

func TestElement_SCEUsage(t *testing.T) {
	// For SCE (Single Channel Element), only ICS1 is used
	var e Element
	e.Channel = 0
	e.PairedChannel = -1
	e.CommonWindow = false
	e.ICS1.WindowSequence = OnlyLongSequence
}

func TestElement_CPEUsage(t *testing.T) {
	// For CPE (Channel Pair Element), both ICS1 and ICS2 are used
	var e Element
	e.Channel = 0
	e.PairedChannel = 1
	e.CommonWindow = true
	e.ICS1.WindowSequence = OnlyLongSequence
	e.ICS2.WindowSequence = OnlyLongSequence
}
