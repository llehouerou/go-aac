// internal/syntax/pce.go
package syntax

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
