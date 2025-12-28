// internal/syntax/tns.go
package syntax

import "github.com/llehouerou/go-aac/internal/bits"

// TNSInfo contains Temporal Noise Shaping filter data.
// TNS applies an all-pole filter to shape the quantization noise.
// Up to 4 filters can be applied per window group.
//
// Ported from: tns_info in ~/dev/faad2/libfaad/structs.h:218-227
type TNSInfo struct {
	NFilt        [MaxWindowGroups]uint8        // Number of filters per window group (0-4)
	CoefRes      [MaxWindowGroups]uint8        // Coefficient resolution (3 or 4 bits)
	Length       [MaxWindowGroups][4]uint8     // Filter length (region) per filter
	Order        [MaxWindowGroups][4]uint8     // Filter order (0-20 for long, 0-7 for short)
	Direction    [MaxWindowGroups][4]uint8     // Filter direction (0=upward, 1=downward)
	CoefCompress [MaxWindowGroups][4]uint8     // Coefficient compression flag
	Coef         [MaxWindowGroups][4][32]uint8 // Filter coefficients (up to 32 per filter)
}

// ParseTNSData parses TNS (Temporal Noise Shaping) data from the bitstream.
// TNS shapes the temporal envelope of the decoded audio to reduce pre-echo artifacts.
//
// Bit widths depend on window type:
//   - Long windows: nFiltBits=2, lengthBits=6, orderBits=5
//   - Short windows: nFiltBits=1, lengthBits=4, orderBits=3
//
// Coefficient precision depends on coef_res and coef_compress:
//   - coef_res=0: start with 3-bit coefficients
//   - coef_res=1: start with 4-bit coefficients
//   - coef_compress=1: reduce coefficient bits by 1
//
// Ported from: tns_data() in ~/dev/faad2/libfaad/syntax.c:2019-2089
func ParseTNSData(r *bits.Reader, ics *ICStream, tns *TNSInfo) {
	var nFiltBits, lengthBits, orderBits uint

	if ics.WindowSequence == EightShortSequence {
		nFiltBits = 1
		lengthBits = 4
		orderBits = 3
	} else {
		nFiltBits = 2
		lengthBits = 6
		orderBits = 5
	}

	for w := uint8(0); w < ics.NumWindows; w++ {
		startCoefBits := uint(3)

		tns.NFilt[w] = uint8(r.GetBits(nFiltBits))

		if tns.NFilt[w] != 0 {
			tns.CoefRes[w] = r.Get1Bit()
			if tns.CoefRes[w] != 0 {
				startCoefBits = 4
			}
		}

		for filt := uint8(0); filt < tns.NFilt[w]; filt++ {
			tns.Length[w][filt] = uint8(r.GetBits(lengthBits))
			tns.Order[w][filt] = uint8(r.GetBits(orderBits))

			if tns.Order[w][filt] != 0 {
				tns.Direction[w][filt] = r.Get1Bit()
				tns.CoefCompress[w][filt] = r.Get1Bit()

				coefBits := startCoefBits - uint(tns.CoefCompress[w][filt])
				for i := uint8(0); i < tns.Order[w][filt]; i++ {
					tns.Coef[w][filt][i] = uint8(r.GetBits(coefBits))
				}
			}
		}
	}
}
