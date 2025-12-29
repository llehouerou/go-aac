// internal/spectrum/tns_tables.go
package spectrum

// TNSMaxOrder is the maximum TNS filter order.
// Ported from: TNS_MAX_ORDER in ~/dev/faad2/libfaad/tns.h:39
const TNSMaxOrder = 20

// TNS coefficient lookup tables.
// These tables convert transmitted coefficient indices to actual filter coefficient values.
// The table selection depends on coef_compress and coef_res_bits (coefficient resolution).
//
// Ported from: tns_coef_0_3, tns_coef_0_4, tns_coef_1_3, tns_coef_1_4
// in ~/dev/faad2/libfaad/tns.c:52-79

// tnsCoef03 is used when coef_compress=0 and coef_res_bits=3
var tnsCoef03 = [16]float64{
	0.0, 0.4338837391, 0.7818314825, 0.9749279122,
	-0.9848077530, -0.8660254038, -0.6427876097, -0.3420201433,
	-0.4338837391, -0.7818314825, -0.9749279122, -0.9749279122,
	-0.9848077530, -0.8660254038, -0.6427876097, -0.3420201433,
}

// tnsCoef04 is used when coef_compress=0 and coef_res_bits=4
var tnsCoef04 = [16]float64{
	0.0, 0.2079116908, 0.4067366431, 0.5877852523,
	0.7431448255, 0.8660254038, 0.9510565163, 0.9945218954,
	-0.9957341763, -0.9618256432, -0.8951632914, -0.7980172273,
	-0.6736956436, -0.5264321629, -0.3612416662, -0.1837495178,
}

// tnsCoef13 is used when coef_compress=1 and coef_res_bits=3
var tnsCoef13 = [16]float64{
	0.0, 0.4338837391, -0.6427876097, -0.3420201433,
	0.9749279122, 0.7818314825, -0.6427876097, -0.3420201433,
	-0.4338837391, -0.7818314825, -0.6427876097, -0.3420201433,
	-0.7818314825, -0.4338837391, -0.6427876097, -0.3420201433,
}

// tnsCoef14 is used when coef_compress=1 and coef_res_bits=4
var tnsCoef14 = [16]float64{
	0.0, 0.2079116908, 0.4067366431, 0.5877852523,
	-0.6736956436, -0.5264321629, -0.3612416662, -0.1837495178,
	0.9945218954, 0.9510565163, 0.8660254038, 0.7431448255,
	-0.6736956436, -0.5264321629, -0.3612416662, -0.1837495178,
}

// allTNSCoefs provides indexed access to coefficient tables.
// Index = 2*coef_compress + (coef_res_bits != 3 ? 1 : 0)
// Ported from: all_tns_coefs in ~/dev/faad2/libfaad/tns.c:81
var allTNSCoefs = [4]*[16]float64{
	&tnsCoef03, // index 0: compress=0, res=3
	&tnsCoef04, // index 1: compress=0, res=4
	&tnsCoef13, // index 2: compress=1, res=3
	&tnsCoef14, // index 3: compress=1, res=4
}

// getTNSCoefTable returns the appropriate coefficient table.
// coefCompress: 0 or 1 (from bitstream)
// coefRes: coef_res field from bitstream (0 means 3-bit, 1 means 4-bit)
//
// Ported from: table_index calculation in ~/dev/faad2/libfaad/tns.c:199
func getTNSCoefTable(coefCompress uint8, coefRes uint8) *[16]float64 {
	// In FAAD2: table_index = 2 * (coef_compress != 0) + (coef_res_bits != 3)
	// coef_res_bits = coef_res + 3, so (coef_res_bits != 3) == (coef_res != 0)
	index := 0
	if coefCompress != 0 {
		index = 2
	}
	if coefRes != 0 {
		index++
	}
	return allTNSCoefs[index]
}
