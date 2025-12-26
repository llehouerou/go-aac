// Package huffman implements AAC Huffman decoding.
// Ported from: ~/dev/faad2/libfaad/huffman.c, huffman.h
package huffman

// Codebook represents a Huffman codebook identifier.
// Source: ~/dev/faad2/libfaad/syntax.h:101-108
type Codebook uint8

// Huffman Codebook identifiers.
const (
	ZeroHCB       Codebook = 0  // No spectral data
	FirstPairHCB  Codebook = 5  // First pair codebook
	EscHCB        Codebook = 11 // Escape codebook
	NoiseHCB      Codebook = 13 // Perceptual Noise Substitution
	IntensityHCB2 Codebook = 14 // Intensity stereo (out of phase)
	IntensityHCB  Codebook = 15 // Intensity stereo (in phase)
)

// Codeword length constants.
// Source: ~/dev/faad2/libfaad/syntax.h:104-105
const (
	QuadLen = 4 // Quadruple length (4 coefficients)
	PairLen = 2 // Pair length (2 coefficients)
)
