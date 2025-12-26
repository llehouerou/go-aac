// Package syntax implements AAC bitstream syntax parsing.
// Ported from: ~/dev/faad2/libfaad/syntax.c, syntax.h
package syntax

// ElementID represents a syntax element identifier.
// Source: ~/dev/faad2/libfaad/syntax.h:85-94
type ElementID uint8

// Syntax Element IDs.
const (
	IDSCE            ElementID = 0x0 // Single Channel Element
	IDCPE            ElementID = 0x1 // Channel Pair Element
	IDCCE            ElementID = 0x2 // Coupling Channel Element
	IDLFE            ElementID = 0x3 // LFE Channel Element
	IDDSE            ElementID = 0x4 // Data Stream Element
	IDPCE            ElementID = 0x5 // Program Config Element
	IDFIL            ElementID = 0x6 // Fill Element
	IDEND            ElementID = 0x7 // Terminating Element
	InvalidElementID ElementID = 255
)

// WindowSequence represents the window sequence type.
// Source: ~/dev/faad2/libfaad/syntax.h:96-99
type WindowSequence uint8

// Window Sequences.
const (
	OnlyLongSequence   WindowSequence = 0x0
	LongStartSequence  WindowSequence = 0x1
	EightShortSequence WindowSequence = 0x2
	LongStopSequence   WindowSequence = 0x3
)

// ExtensionType represents an extension element type.
// Source: ~/dev/faad2/libfaad/syntax.h:79-83
type ExtensionType uint8

// Extension Types.
const (
	ExtFil          ExtensionType = 0  // Filler extension
	ExtFillData     ExtensionType = 1  // Fill with MPEG surround data
	ExtDataElement  ExtensionType = 2  // Data element
	ExtDynamicRange ExtensionType = 11 // Dynamic Range Control
)

// AncData is the ancillary data type.
// Source: ~/dev/faad2/libfaad/syntax.h:83
const AncData = 0

// Bit length constants for parsing.
// Source: ~/dev/faad2/libfaad/syntax.h:74-77
const (
	LenSEID = 3 // Syntax element identifier length in bits
	LenTag  = 4 // Element instance tag length in bits
	LenByte = 8 // Byte length in bits
)

// DRMChannelConfig represents a DRM channel configuration.
// Source: ~/dev/faad2/libfaad/syntax.h:62-67
type DRMChannelConfig uint8

// DRM Channel Configurations.
const (
	DRMCHMono        DRMChannelConfig = 1
	DRMCHStereo      DRMChannelConfig = 2
	DRMCHSBRMono     DRMChannelConfig = 3
	DRMCHSBRStereo   DRMChannelConfig = 4
	DRMCHSBRPSStereo DRMChannelConfig = 5
)

// ERObjectStart is the first object type that has Error Resilience.
// Source: ~/dev/faad2/libfaad/syntax.h:71
const ERObjectStart = 17

// InvalidSBRElement marks an invalid SBR element.
// Source: ~/dev/faad2/libfaad/syntax.h:110
const InvalidSBRElement = 255
