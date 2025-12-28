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

// FillConfig holds configuration for Fill Element parsing.
// Currently empty, but structured for future SBR support.
type FillConfig struct {
	// SBRElement specifies which SBR element to associate with.
	// Set to InvalidSBRElement (255) if no SBR association.
	SBRElement uint8
}

// ParseFillElement parses a Fill Element (ID_FIL).
// Fill elements contain extension payloads including DRC and SBR data.
//
// For now, SBR data is skipped. SBR support is implemented in Phase 8.
//
// Ported from: fill_element() in ~/dev/faad2/libfaad/syntax.c:1110-1197
func ParseFillElement(r *bits.Reader, drc *DRCInfo) error {
	return ParseFillElementWithConfig(r, drc, &FillConfig{SBRElement: InvalidSBRElement})
}

// ParseFillElementWithConfig parses a Fill Element with explicit configuration.
// This variant allows specifying SBR element association for future SBR support.
//
// Ported from: fill_element() in ~/dev/faad2/libfaad/syntax.c:1110-1197
func ParseFillElementWithConfig(r *bits.Reader, drc *DRCInfo, cfg *FillConfig) error {
	_ = cfg // Unused for now, will be used for SBR in Phase 8

	// count (4 bits)
	count := uint16(r.GetBits(4))

	// If count == 15, read extended count
	if count == 15 {
		count += uint16(r.GetBits(8)) - 1
	}

	if count == 0 {
		return nil
	}

	// Check for SBR extension (Phase 8)
	// For now, just peek and skip SBR data
	bsExtensionType := ExtensionType(r.ShowBits(4))

	if bsExtensionType == ExtSBRData || bsExtensionType == ExtSBRDataCRC {
		// SBR data - skip for now (Phase 8 implementation)
		// Just consume all the count bytes
		for i := uint16(0); i < count; i++ {
			r.GetBits(8)
		}
		return nil
	}

	// Parse extension payloads until count is exhausted
	for count > 0 {
		payloadBytes := parseExtensionPayload(r, drc, count)
		if payloadBytes <= count {
			count -= payloadBytes
		} else {
			count = 0
		}
	}

	return nil
}

// parseExtensionPayload parses an extension_payload() element.
// Returns the number of payload bytes consumed.
//
// Ported from: extension_payload() in ~/dev/faad2/libfaad/syntax.c:2240-2299
func parseExtensionPayload(r *bits.Reader, drc *DRCInfo, count uint16) uint16 {
	var align uint8 = 4

	extensionType := ExtensionType(r.GetBits(4))

	switch extensionType {
	case ExtDynamicRange:
		drc.Present = true
		n := parseDynamicRangeInfo(r, drc)
		return uint16(n)

	case ExtFillData:
		// fill_nibble (must be 0000)
		_ = r.GetBits(4)
		// fill_byte (must be 0xA5 "10100101")
		for i := uint16(0); i < count-1; i++ {
			_ = r.GetBits(8)
		}
		return count

	case ExtDataElement:
		dataElementVersion := r.GetBits(4)
		switch dataElementVersion {
		case AncData:
			loopCounter := uint16(0)
			dataElementLength := uint16(0)
			for {
				dataElementLengthPart := uint8(r.GetBits(8))
				dataElementLength += uint16(dataElementLengthPart)
				loopCounter++
				if dataElementLengthPart != 255 {
					break
				}
			}
			// Read first data_element_byte if present
			// Note: FAAD2 returns after reading only the first byte, which seems like a bug
			// in the original C code (the for loop reads one byte then returns).
			// We follow the same behavior for compatibility.
			if dataElementLength > 0 {
				_ = r.GetBits(8)
				return dataElementLength + loopCounter + 1
			}
			// If dataElementLength is 0
			return loopCounter + 1
		default:
			align = 0
		}
		fallthrough

	case ExtFil:
		fallthrough

	default:
		// Read fill_nibble or align bits
		r.GetBits(uint(align))
		// Read remaining bytes
		for i := uint16(0); i < count-1; i++ {
			_ = r.GetBits(8)
		}
		return count
	}
}
