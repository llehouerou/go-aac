// internal/syntax/pce.go
package syntax

import (
	"errors"

	"github.com/llehouerou/go-aac/internal/bits"
)

// ErrTooManyChannels is returned when a PCE has more channels than supported.
var ErrTooManyChannels = errors.New("too many channels in program config")

// ProgramConfig contains Program Configuration Element data.
// The PCE describes the channel configuration for complex streams,
// mapping syntax elements to output channels.
//
// Ported from: program_config in ~/dev/faad2/libfaad/structs.h:103-144
type ProgramConfig struct {
	// Basic info
	ElementInstanceTag uint8 // Element instance tag
	ObjectType         uint8 // Audio object type
	SFIndex            uint8 // Sample frequency index

	// Element counts
	NumFrontChannelElements uint8 // Front channel element count
	NumSideChannelElements  uint8 // Side channel element count
	NumBackChannelElements  uint8 // Back channel element count
	NumLFEChannelElements   uint8 // LFE channel element count
	NumAssocDataElements    uint8 // Associated data element count
	NumValidCCElements      uint8 // Valid coupling channel count

	// Mixdown info
	MonoMixdownPresent         bool  // Mono mixdown element present
	MonoMixdownElementNumber   uint8 // Mono mixdown element number
	StereoMixdownPresent       bool  // Stereo mixdown element present
	StereoMixdownElementNumber uint8 // Stereo mixdown element number
	MatrixMixdownIdxPresent    bool  // Matrix mixdown present
	PseudoSurroundEnable       bool  // Pseudo surround enabled
	MatrixMixdownIdx           uint8 // Matrix mixdown index

	// Element configuration (up to 16 of each type)
	FrontElementIsCPE         [16]bool  // True if front element is CPE
	FrontElementTagSelect     [16]uint8 // Front element instance tags
	SideElementIsCPE          [16]bool  // True if side element is CPE
	SideElementTagSelect      [16]uint8 // Side element instance tags
	BackElementIsCPE          [16]bool  // True if back element is CPE
	BackElementTagSelect      [16]uint8 // Back element instance tags
	LFEElementTagSelect       [16]uint8 // LFE element instance tags
	AssocDataElementTagSelect [16]uint8 // Assoc data element tags
	CCElementIsIndSW          [16]bool  // CC element is independently switched
	ValidCCElementTagSelect   [16]uint8 // Valid CC element tags

	// Total channel count (computed)
	Channels uint8

	// Comment field
	CommentFieldBytes uint8      // Comment length
	CommentFieldData  [257]uint8 // Comment data

	// Derived values (computed after parsing)
	NumFrontChannels uint8     // Total front channels
	NumSideChannels  uint8     // Total side channels
	NumBackChannels  uint8     // Total back channels
	NumLFEChannels   uint8     // Total LFE channels
	SCEChannel       [16]uint8 // SCE to channel mapping
	CPEChannel       [16]uint8 // CPE to channel mapping
}

// ParsePCE parses a Program Configuration Element from the bitstream.
// It reads the channel configuration and computes derived channel mappings.
//
// Ported from: program_config_element() in ~/dev/faad2/libfaad/syntax.c:174-323
func ParsePCE(r *bits.Reader) (*ProgramConfig, error) {
	pce := &ProgramConfig{}

	// Basic info
	pce.ElementInstanceTag = uint8(r.GetBits(4))
	pce.ObjectType = uint8(r.GetBits(2))
	pce.SFIndex = uint8(r.GetBits(4))

	// Element counts
	pce.NumFrontChannelElements = uint8(r.GetBits(4))
	pce.NumSideChannelElements = uint8(r.GetBits(4))
	pce.NumBackChannelElements = uint8(r.GetBits(4))
	pce.NumLFEChannelElements = uint8(r.GetBits(2))
	pce.NumAssocDataElements = uint8(r.GetBits(3))
	pce.NumValidCCElements = uint8(r.GetBits(4))

	// Mixdown flags and optional element numbers
	pce.MonoMixdownPresent = r.Get1Bit() == 1
	if pce.MonoMixdownPresent {
		pce.MonoMixdownElementNumber = uint8(r.GetBits(4))
	}

	pce.StereoMixdownPresent = r.Get1Bit() == 1
	if pce.StereoMixdownPresent {
		pce.StereoMixdownElementNumber = uint8(r.GetBits(4))
	}

	pce.MatrixMixdownIdxPresent = r.Get1Bit() == 1
	if pce.MatrixMixdownIdxPresent {
		pce.MatrixMixdownIdx = uint8(r.GetBits(2))
		pce.PseudoSurroundEnable = r.Get1Bit() == 1
	}

	// Front channel elements
	for i := uint8(0); i < pce.NumFrontChannelElements; i++ {
		pce.FrontElementIsCPE[i] = r.Get1Bit() == 1
		pce.FrontElementTagSelect[i] = uint8(r.GetBits(4))

		if pce.FrontElementIsCPE[i] {
			pce.CPEChannel[pce.FrontElementTagSelect[i]] = pce.Channels
			pce.NumFrontChannels += 2
			pce.Channels += 2
		} else {
			pce.SCEChannel[pce.FrontElementTagSelect[i]] = pce.Channels
			pce.NumFrontChannels++
			pce.Channels++
		}
	}

	// Side channel elements
	for i := uint8(0); i < pce.NumSideChannelElements; i++ {
		pce.SideElementIsCPE[i] = r.Get1Bit() == 1
		pce.SideElementTagSelect[i] = uint8(r.GetBits(4))

		if pce.SideElementIsCPE[i] {
			pce.CPEChannel[pce.SideElementTagSelect[i]] = pce.Channels
			pce.NumSideChannels += 2
			pce.Channels += 2
		} else {
			pce.SCEChannel[pce.SideElementTagSelect[i]] = pce.Channels
			pce.NumSideChannels++
			pce.Channels++
		}
	}

	// Back channel elements
	for i := uint8(0); i < pce.NumBackChannelElements; i++ {
		pce.BackElementIsCPE[i] = r.Get1Bit() == 1
		pce.BackElementTagSelect[i] = uint8(r.GetBits(4))

		if pce.BackElementIsCPE[i] {
			pce.CPEChannel[pce.BackElementTagSelect[i]] = pce.Channels
			pce.NumBackChannels += 2
			pce.Channels += 2
		} else {
			pce.SCEChannel[pce.BackElementTagSelect[i]] = pce.Channels
			pce.NumBackChannels++
			pce.Channels++
		}
	}

	// LFE channel elements (no CPE flag, always SCE)
	for i := uint8(0); i < pce.NumLFEChannelElements; i++ {
		pce.LFEElementTagSelect[i] = uint8(r.GetBits(4))

		pce.SCEChannel[pce.LFEElementTagSelect[i]] = pce.Channels
		pce.NumLFEChannels++
		pce.Channels++
	}

	// Associated data elements (just tag, no channel assignment)
	for i := uint8(0); i < pce.NumAssocDataElements; i++ {
		pce.AssocDataElementTagSelect[i] = uint8(r.GetBits(4))
	}

	// Coupling channel elements
	for i := uint8(0); i < pce.NumValidCCElements; i++ {
		pce.CCElementIsIndSW[i] = r.Get1Bit() == 1
		pce.ValidCCElementTagSelect[i] = uint8(r.GetBits(4))
	}

	// Byte align before comment field
	r.ByteAlign()

	// Comment field
	pce.CommentFieldBytes = uint8(r.GetBits(8))
	for i := uint8(0); i < pce.CommentFieldBytes; i++ {
		pce.CommentFieldData[i] = uint8(r.GetBits(8))
	}

	// Validate channel count
	if pce.Channels > MaxChannels {
		return nil, ErrTooManyChannels
	}

	return pce, nil
}
