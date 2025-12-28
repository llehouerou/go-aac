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

// parseDynamicRangeInfo parses the dynamic_range_info() element.
// Returns the number of "logical bytes" consumed (for extension_payload counting).
//
// Ported from: dynamic_range_info() in ~/dev/faad2/libfaad/syntax.c:2302-2364
func parseDynamicRangeInfo(r *bits.Reader, drc *DRCInfo) uint8 {
	var n uint8 = 1

	drc.NumBands = 1

	// has_instance_tag (1 bit)
	if r.Get1Bit() == 1 {
		drc.PCEInstanceTag = uint8(r.GetBits(4))
		_ = r.GetBits(4) // drc_tag_reserved_bits
		n++
	}

	// excluded_chns_present (1 bit)
	drc.ExcludedChnsPresent = r.Get1Bit() == 1
	if drc.ExcludedChnsPresent {
		n += parseExcludedChannels(r, drc)
	}

	// has_bands_data (1 bit)
	if r.Get1Bit() == 1 {
		bandIncr := uint8(r.GetBits(4))
		_ = r.GetBits(4) // drc_bands_reserved_bits
		n++
		drc.NumBands += bandIncr

		for i := uint8(0); i < drc.NumBands; i++ {
			drc.BandTop[i] = uint8(r.GetBits(8))
			n++
		}
	}

	// has_prog_ref_level (1 bit)
	if r.Get1Bit() == 1 {
		drc.ProgRefLevel = uint8(r.GetBits(7))
		_ = r.Get1Bit() // prog_ref_level_reserved_bits
		n++
	}

	// Read dynamic range data for each band
	for i := uint8(0); i < drc.NumBands; i++ {
		drc.DynRngSgn[i] = r.Get1Bit()
		drc.DynRngCtl[i] = uint8(r.GetBits(7))
		n++
	}

	return n
}
