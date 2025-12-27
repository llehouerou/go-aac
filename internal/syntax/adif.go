// internal/syntax/adif.go
package syntax

// ADIFMagic is the 4-byte magic number for ADIF files.
var ADIFMagic = [4]byte{'A', 'D', 'I', 'F'}

// ADIFHeader contains Audio Data Interchange Format header data.
// ADIF is a simple header-at-beginning format, less common than ADTS.
// It contains one or more Program Configuration Elements.
//
// Ported from: adif_header in ~/dev/faad2/libfaad/structs.h:170-183
type ADIFHeader struct {
	CopyrightIDPresent       bool              // Copyright ID field present
	CopyrightID              [10]int8          // Copyright ID (10 bytes, per FAAD2 structs.h:173)
	OriginalCopy             bool              // Original/copy flag
	Home                     bool              // Home flag
	BitstreamType            uint8             // 0=constant rate, 1=variable rate
	Bitrate                  uint32            // Bit rate (bits/sec)
	NumProgramConfigElements uint8             // Number of PCEs (0-15)
	ADIFBufferFullness       uint32            // Buffer fullness (variable rate only)
	PCE                      [16]ProgramConfig // Program Configuration Elements (up to 16)
}
