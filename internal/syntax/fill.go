// internal/syntax/fill.go
package syntax

import "github.com/llehouerou/go-aac/internal/bits"

// parseExcludedChannels parses the excluded_channels() element for DRC.
// Returns the number of bytes consumed (for byte counting in DRC parsing).
//
// Ported from: excluded_channels() in ~/dev/faad2/libfaad/syntax.c:2367-2394
func parseExcludedChannels(r *bits.Reader, drc *DRCInfo) uint8 {
	var n uint8
	numExclChan := 7

	// Read first 7 exclude_mask bits
	for i := 0; i < 7; i++ {
		drc.ExcludeMask[i] = r.Get1Bit()
	}
	n++

	// Read additional excluded channels groups
	for {
		additionalBit := r.Get1Bit()
		drc.AdditionalExcludedChns[n-1] = additionalBit

		if additionalBit == 0 {
			break
		}

		// Check bounds
		if numExclChan >= MaxChannels-7 {
			return n
		}

		// Read next 7 exclude_mask bits
		for i := numExclChan; i < numExclChan+7; i++ {
			if i < MaxChannels {
				drc.ExcludeMask[i] = r.Get1Bit()
			}
		}
		n++
		numExclChan += 7
	}

	return n
}
