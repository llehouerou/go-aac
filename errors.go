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

	// Initialization errors.
	// Ported from: decoder.c:276-278, 405-412
	ErrNilDecoder            Error = 35 // nil decoder handle
	ErrNilBuffer             Error = 36 // nil buffer passed to Init
	ErrBufferTooSmall        Error = 37 // buffer too small (< 2 bytes)
	ErrUnsupportedObjectType Error = 38 // unsupported audio object type
	ErrInvalidSampleRate     Error = 39 // invalid sample rate (0)
	ErrADIFNotSupported      Error = 40 // ADIF format not yet supported
)

// errMessages contains error messages matching FAAD2 exactly.
// Source: ~/dev/faad2/libfaad/error.c:34-69
var errMessages = map[Error]string{
	0:  "No error",
	1:  "Gain control not yet implemented",
	2:  "Pulse coding not allowed in short blocks",
	3:  "Invalid huffman codebook",
	4:  "Scalefactor out of range",
	5:  "Unable to find ADTS syncword",
	6:  "Channel coupling not yet implemented",
	7:  "Channel configuration not allowed in error resilient frame",
	8:  "Bit error in error resilient scalefactor decoding",
	9:  "Error decoding huffman scalefactor (bitstream error)",
	10: "Error decoding huffman codeword (bitstream error)",
	11: "Non existent huffman codebook number found",
	12: "Invalid number of channels",
	13: "Maximum number of bitstream elements exceeded",
	14: "Input data buffer too small",
	15: "Array index out of range",
	16: "Maximum number of scalefactor bands exceeded",
	17: "Quantised value out of range",
	18: "LTP lag out of range",
	19: "Invalid SBR parameter decoded",
	20: "SBR called without being initialised",
	21: "Unexpected channel configuration change",
	22: "Error in program_config_element",
	23: "First SBR frame is not the same as first AAC frame",
	24: "Unexpected fill element with SBR data",
	25: "Not all elements were provided with SBR data",
	26: "LTP decoding not available",
	27: "Output data buffer too small",
	28: "CRC error in DRM data",
	29: "PNS not allowed in DRM data stream",
	30: "No standard extension payload allowed in DRM",
	31: "PCE shall be the first element in a frame",
	32: "Bitstream value not allowed by specification",
	33: "MAIN prediction not initialised",
	// Initialization errors (go-aac specific, not from FAAD2 error.c)
	35: "nil decoder handle",
	36: "nil buffer passed to Init",
	37: "buffer too small",
	38: "unsupported audio object type",
	39: "invalid sample rate",
	40: "ADIF format not yet supported",
}

// Error implements the error interface.
func (e Error) Error() string {
	if msg, ok := errMessages[e]; ok {
		return msg
	}
	return "unknown error"
}
