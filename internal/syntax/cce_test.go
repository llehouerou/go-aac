// internal/syntax/cce_test.go
package syntax

import (
	"testing"

	"github.com/llehouerou/go-aac/internal/bits"
)

func TestCCEConfig_Initialization(t *testing.T) {
	cfg := &CCEConfig{
		SFIndex:     4, // 44100 Hz
		FrameLength: 1024,
		ObjectType:  2, // AAC-LC
	}

	if cfg.SFIndex != 4 {
		t.Errorf("SFIndex: got %d, want 4", cfg.SFIndex)
	}
	if cfg.FrameLength != 1024 {
		t.Errorf("FrameLength: got %d, want 1024", cfg.FrameLength)
	}
	if cfg.ObjectType != 2 {
		t.Errorf("ObjectType: got %d, want 2", cfg.ObjectType)
	}
}

func TestCCEResult_Initialization(t *testing.T) {
	result := &CCEResult{}

	if result.Tag != 0 {
		t.Errorf("Tag should be zero-initialized")
	}
	if result.IndSwCCEFlag {
		t.Errorf("IndSwCCEFlag should be false initially")
	}
	if result.NumCoupledElements != 0 {
		t.Errorf("NumCoupledElements should be zero-initialized")
	}
}

func TestCCECoupledElement_Initialization(t *testing.T) {
	elem := CCECoupledElement{
		TargetIsCPE: true,
		TargetTag:   5,
		CCL:         true,
		CCR:         false,
	}

	if !elem.TargetIsCPE {
		t.Errorf("TargetIsCPE should be true")
	}
	if elem.TargetTag != 5 {
		t.Errorf("TargetTag: got %d, want 5", elem.TargetTag)
	}
	if !elem.CCL {
		t.Errorf("CCL should be true")
	}
	if elem.CCR {
		t.Errorf("CCR should be false")
	}
}

func TestParseCCEHeader_SingleSCETarget(t *testing.T) {
	// CCE with 1 coupled SCE element:
	// element_instance_tag: 0 (4 bits) = 0000
	// ind_sw_cce_flag: 0 (1 bit) = 0
	// num_coupled_elements: 0 (3 bits) = 000 (meaning 1 element)
	// cc_target_is_cpe: 0 (1 bit) = 0 (SCE target)
	// cc_target_tag_select: 1 (4 bits) = 0001
	// cc_domain: 0 (1 bit) = 0
	// gain_element_sign: 0 (1 bit) = 0
	// gain_element_scale: 1 (2 bits) = 01
	// Total: 4+1+3+1+4+1+1+2 = 17 bits
	// Bits: 0000 0 000 | 0 0001 0 0 0 | 1...
	// Byte0: 0000_0000 = 0x00
	// Byte1: 0_0001_000 = 0x08
	// Byte2: 1_0000000 = 0x80 (gain_element_scale MSB, rest padding)
	data := []byte{0x00, 0x08, 0x80}
	r := bits.NewReader(data)

	result := &CCEResult{}
	err := parseCCEHeader(r, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Tag != 0 {
		t.Errorf("Tag: got %d, want 0", result.Tag)
	}
	if result.IndSwCCEFlag {
		t.Errorf("IndSwCCEFlag should be false")
	}
	if result.NumCoupledElements != 0 {
		t.Errorf("NumCoupledElements: got %d, want 0", result.NumCoupledElements)
	}
	if result.NumGainElementLists != 1 {
		t.Errorf("NumGainElementLists: got %d, want 1", result.NumGainElementLists)
	}
	if result.CoupledElements[0].TargetIsCPE {
		t.Errorf("CoupledElements[0].TargetIsCPE should be false")
	}
	if result.CoupledElements[0].TargetTag != 1 {
		t.Errorf("CoupledElements[0].TargetTag: got %d, want 1", result.CoupledElements[0].TargetTag)
	}
	if result.CCDomain {
		t.Errorf("CCDomain should be false")
	}
	if result.GainElementSign {
		t.Errorf("GainElementSign should be false")
	}
	if result.GainElementScale != 1 {
		t.Errorf("GainElementScale: got %d, want 1", result.GainElementScale)
	}
}

func TestParseCCEHeader_CPETarget_BothChannels(t *testing.T) {
	// CCE with 1 coupled CPE element targeting both channels:
	// element_instance_tag: 2 (4 bits) = 0010
	// ind_sw_cce_flag: 1 (1 bit) = 1
	// num_coupled_elements: 0 (3 bits) = 000 (meaning 1 element)
	// cc_target_is_cpe: 1 (1 bit) = 1 (CPE target)
	// cc_target_tag_select: 3 (4 bits) = 0011
	// cc_l: 1 (1 bit) = 1
	// cc_r: 1 (1 bit) = 1 (both channels = extra gain list)
	// cc_domain: 1 (1 bit) = 1
	// gain_element_sign: 1 (1 bit) = 1
	// gain_element_scale: 2 (2 bits) = 10
	// Total: 4+1+3+1+4+1+1+1+1+2 = 19 bits
	// Bits: 0010 1 000 | 1 0011 1 1 1 | 1 10...
	// Byte0: 0010_1000 = 0x28
	// Byte1: 1_0011_111 = 0x9F
	// Byte2: 1_10_00000 = 0xC0 (gain_element_sign=1, gain_element_scale=10, rest padding)
	data := []byte{0x28, 0x9F, 0xC0}
	r := bits.NewReader(data)

	result := &CCEResult{}
	err := parseCCEHeader(r, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Tag != 2 {
		t.Errorf("Tag: got %d, want 2", result.Tag)
	}
	if !result.IndSwCCEFlag {
		t.Errorf("IndSwCCEFlag should be true")
	}
	// Both channels = 2 gain element lists
	if result.NumGainElementLists != 2 {
		t.Errorf("NumGainElementLists: got %d, want 2", result.NumGainElementLists)
	}
	if !result.CoupledElements[0].TargetIsCPE {
		t.Errorf("CoupledElements[0].TargetIsCPE should be true")
	}
	if result.CoupledElements[0].TargetTag != 3 {
		t.Errorf("CoupledElements[0].TargetTag: got %d, want 3", result.CoupledElements[0].TargetTag)
	}
	if !result.CoupledElements[0].CCL {
		t.Errorf("CoupledElements[0].CCL should be true")
	}
	if !result.CoupledElements[0].CCR {
		t.Errorf("CoupledElements[0].CCR should be true")
	}
	if !result.CCDomain {
		t.Errorf("CCDomain should be true")
	}
	if !result.GainElementSign {
		t.Errorf("GainElementSign should be true")
	}
	if result.GainElementScale != 2 {
		t.Errorf("GainElementScale: got %d, want 2", result.GainElementScale)
	}
}
