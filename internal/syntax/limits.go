package syntax

// Limit constants for AAC decoding.
// Source: ~/dev/faad2/libfaad/structs.h:43-48

const (
	MaxChannels        = 64 // Maximum number of channels
	MaxSyntaxElements  = 48 // Maximum number of syntax elements
	MaxWindowGroups    = 8  // Maximum number of window groups
	MaxSFB             = 51 // Maximum number of scalefactor bands
	MaxLTPSFB          = 40 // Maximum LTP scalefactor bands (long)
	MaxLTPSFBS         = 8  // Maximum LTP scalefactor bands (short)
	MaxCoupledElements = 8  // Maximum coupled elements in CCE (3 bits = 0-7, +1 loop)
)
