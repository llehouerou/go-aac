package spectrum

import (
	"errors"

	"github.com/llehouerou/go-aac/internal/tables"
)

// ErrLengthMismatch indicates input and output slice lengths don't match.
var ErrLengthMismatch = errors.New("spectrum: input and output length mismatch")

// InverseQuantize applies inverse quantization to an array of spectral coefficients.
// For each element: spec[i] = sign(quant[i]) * |quant[i]|^(4/3)
//
// The quantData and specData slices must have the same length.
// Uses the precomputed IQTable for efficiency.
//
// Ported from: iquant() usage in ~/dev/faad2/libfaad/specrec.c:636-639
func InverseQuantize(quantData []int16, specData []float64) error {
	if len(quantData) != len(specData) {
		return ErrLengthMismatch
	}

	for i, q := range quantData {
		val, err := tables.IQuant(q)
		if err != nil {
			return err
		}
		specData[i] = val
	}

	return nil
}
