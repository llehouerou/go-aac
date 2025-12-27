// internal/syntax/tns.go
package syntax

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
