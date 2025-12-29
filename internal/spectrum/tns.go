// internal/spectrum/tns.go
package spectrum

// tnsDecodeCoef converts transmitted TNS coefficients to LPC filter coefficients.
// Uses Levinson-Durbin recursion to convert reflection coefficients to direct form.
//
// Parameters:
//   - order: filter order (0-20)
//   - coefRes: coefficient resolution (0=3-bit, 1=4-bit)
//   - coefCompress: compression flag (0 or 1)
//   - coef: transmitted coefficient indices
//   - lpc: output LPC coefficients (must be len >= order+1)
//
// Ported from: tns_decode_coef() in ~/dev/faad2/libfaad/tns.c:193-242
func tnsDecodeCoef(order uint8, coefRes uint8, coefCompress uint8, coef []uint8, lpc []float64) {
	// Get the appropriate coefficient table
	tnsCoef := getTNSCoefTable(coefCompress, coefRes)

	// Convert transmitted indices to coefficient values
	tmp2 := make([]float64, TNSMaxOrder+1)
	for i := uint8(0); i < order; i++ {
		tmp2[i] = tnsCoef[coef[i]]
	}

	// Levinson-Durbin recursion to convert reflection coefficients to LPC
	// a[0] is always 1.0
	lpc[0] = 1.0

	b := make([]float64, TNSMaxOrder+1)
	for m := uint8(1); m <= order; m++ {
		// Set a[m] = reflection coefficient
		lpc[m] = tmp2[m-1]

		// Update previous coefficients
		for i := uint8(1); i < m; i++ {
			b[i] = lpc[i] + lpc[m]*lpc[m-i]
		}
		for i := uint8(1); i < m; i++ {
			lpc[i] = b[i]
		}
	}
}
