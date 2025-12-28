// internal/syntax/adif.go
package syntax

import (
	"errors"

	"github.com/llehouerou/go-aac/internal/bits"
)

// ADIFMagic is the 4-byte magic number for ADIF files.
var ADIFMagic = [4]byte{'A', 'D', 'I', 'F'}

// ErrADIFBitstream is returned when a bitstream error occurs during ADIF parsing.
var ErrADIFBitstream = errors.New("ADIF bitstream error")

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

// HasCopyrightID returns the copyright ID string and whether it is present.
// If not present, returns empty string and false.
func (h *ADIFHeader) HasCopyrightID() (string, bool) {
	if !h.CopyrightIDPresent {
		return "", false
	}

	// Convert int8 array to string, stopping at null terminator
	result := make([]byte, 0, len(h.CopyrightID))
	for _, c := range h.CopyrightID {
		if c == 0 {
			break
		}
		result = append(result, byte(c))
	}
	return string(result), true
}

// IsConstantRate returns true if the bitstream is constant rate (bitstream_type == 0).
func (h *ADIFHeader) IsConstantRate() bool {
	return h.BitstreamType == 0
}

// GetPCEs returns a slice of the valid Program Configuration Elements.
// The number of valid PCEs is NumProgramConfigElements + 1.
func (h *ADIFHeader) GetPCEs() []ProgramConfig {
	count := int(h.NumProgramConfigElements) + 1
	return h.PCE[:count]
}

// ParseADIF parses an ADIF header from the bitstream.
// The "ADIF" magic bytes must already be consumed before calling this function.
//
// The function parses:
//   - copyright_id_present: 1 bit
//   - copyright_id: 72 bits (9 bytes) if copyright_id_present
//   - original_copy: 1 bit
//   - home: 1 bit
//   - bitstream_type: 1 bit (0=constant rate, 1=variable rate)
//   - bitrate: 23 bits
//   - num_program_config_elements: 4 bits (actual count is value+1)
//   - For each PCE (num+1 times):
//   - adif_buffer_fullness: 20 bits (only if bitstream_type=0)
//   - Call ParsePCE() to parse the PCE
//
// Ported from: get_adif_header() in ~/dev/faad2/libfaad/syntax.c:2400-2446
func ParseADIF(r *bits.Reader) (*ADIFHeader, error) {
	// Check for bitstream error before starting
	if r.Error() {
		return nil, ErrADIFBitstream
	}

	h := &ADIFHeader{}

	// copyright_id_present: 1 bit
	h.CopyrightIDPresent = r.Get1Bit() == 1

	// copyright_id: 72 bits (9 bytes) if present
	// FAAD2 reads 72/8 = 9 bytes and null-terminates at index 9
	if h.CopyrightIDPresent {
		for i := 0; i < 9; i++ {
			h.CopyrightID[i] = int8(r.GetBits(8))
		}
		h.CopyrightID[9] = 0 // null terminate
	}

	// original_copy: 1 bit
	h.OriginalCopy = r.Get1Bit() == 1

	// home: 1 bit
	h.Home = r.Get1Bit() == 1

	// bitstream_type: 1 bit (0=constant rate, 1=variable rate)
	h.BitstreamType = r.Get1Bit()

	// bitrate: 23 bits
	h.Bitrate = r.GetBits(23)

	// num_program_config_elements: 4 bits (actual count is value+1)
	h.NumProgramConfigElements = uint8(r.GetBits(4))

	// Parse each PCE (num_program_config_elements + 1 times)
	for i := uint8(0); i <= h.NumProgramConfigElements; i++ {
		// adif_buffer_fullness: 20 bits (only for constant rate, i.e., bitstream_type=0)
		if h.BitstreamType == 0 {
			h.ADIFBufferFullness = r.GetBits(20)
		} else {
			h.ADIFBufferFullness = 0
		}

		// Parse the Program Configuration Element
		pce, err := ParsePCE(r)
		if err != nil {
			return nil, err
		}
		h.PCE[i] = *pce
	}

	// Check for bitstream errors
	if r.Error() {
		return nil, ErrADIFBitstream
	}

	return h, nil
}
