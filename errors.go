package aac

// Error represents an AAC decoder error code.
// Ported from: ~/dev/faad2/libfaad/error.c, error.h
type Error int

// Error codes from FAAD2.
// Source: ~/dev/faad2/libfaad/error.c:34-69
const (
	ErrNone                      Error = 0
	ErrGainControlNotImplemented Error = 1
	ErrPulseInShortBlock         Error = 2
	ErrInvalidHuffmanCodebook    Error = 3
	ErrScalefactorOutOfRange     Error = 4
	ErrADTSSyncwordNotFound      Error = 5
	ErrChannelCouplingNotImpl    Error = 6
	ErrChannelConfigNotAllowed   Error = 7
	ErrBitErrorScalefactor       Error = 8
	ErrHuffmanScalefactor        Error = 9
	ErrHuffmanCodeword           Error = 10
	ErrNonExistentCodebook       Error = 11
	ErrInvalidNumChannels        Error = 12
	ErrMaxBitstreamElements      Error = 13
	ErrInputBufferTooSmall       Error = 14
	ErrArrayIndexOutOfRange      Error = 15
	ErrMaxScalefactorBands       Error = 16
	ErrQuantisedValueOutOfRange  Error = 17
	ErrLTPLagOutOfRange          Error = 18
	ErrInvalidSBRParameter       Error = 19
	ErrSBRNotInitialised         Error = 20
	ErrUnexpectedChannelChange   Error = 21
	ErrProgramConfigElement      Error = 22
	ErrSBRFirstFrame             Error = 23
	ErrUnexpectedFillElement     Error = 24
	ErrSBRDataMissing            Error = 25
	ErrLTPNotAvailable           Error = 26
	ErrOutputBufferTooSmall      Error = 27
	ErrDRMCRC                    Error = 28
	ErrPNSNotAllowedInDRM        Error = 29
	ErrNoExtPayloadInDRM         Error = 30
	ErrPCENotFirst               Error = 31
	ErrBitstreamValueNotAllowed  Error = 32
	ErrMAINPredictionNotInit     Error = 33
)

// errMessages contains error messages matching FAAD2 exactly.
// Source: ~/dev/faad2/libfaad/error.c:34-69
var errMessages = [34]string{
	"No error",
	"Gain control not yet implemented",
	"Pulse coding not allowed in short blocks",
	"Invalid huffman codebook",
	"Scalefactor out of range",
	"Unable to find ADTS syncword",
	"Channel coupling not yet implemented",
	"Channel configuration not allowed in error resilient frame",
	"Bit error in error resilient scalefactor decoding",
	"Error decoding huffman scalefactor (bitstream error)",
	"Error decoding huffman codeword (bitstream error)",
	"Non existent huffman codebook number found",
	"Invalid number of channels",
	"Maximum number of bitstream elements exceeded",
	"Input data buffer too small",
	"Array index out of range",
	"Maximum number of scalefactor bands exceeded",
	"Quantised value out of range",
	"LTP lag out of range",
	"Invalid SBR parameter decoded",
	"SBR called without being initialised",
	"Unexpected channel configuration change",
	"Error in program_config_element",
	"First SBR frame is not the same as first AAC frame",
	"Unexpected fill element with SBR data",
	"Not all elements were provided with SBR data",
	"LTP decoding not available",
	"Output data buffer too small",
	"CRC error in DRM data",
	"PNS not allowed in DRM data stream",
	"No standard extension payload allowed in DRM",
	"PCE shall be the first element in a frame",
	"Bitstream value not allowed by specification",
	"MAIN prediction not initialised",
}

// Error implements the error interface.
func (e Error) Error() string {
	if e >= 0 && int(e) < len(errMessages) {
		return errMessages[e]
	}
	return "unknown error"
}
