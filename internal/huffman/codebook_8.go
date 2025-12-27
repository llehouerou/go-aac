// Package huffman implements AAC Huffman decoding.
package huffman

// hcb8_1 is the first-step lookup table for codebook 8.
// Uses 5 bits to find offset into second-step table and number of extra bits to read.
//
// Copied from: ~/dev/faad2/libfaad/codebook/hcb_8.h:39-87 (hcb8_1[32])
var hcb8_1 = [32]HCB{
	// 3-bit codeword (4 duplicates)
	{0, 0}, // 00000
	{0, 0}, //
	{0, 0}, //
	{0, 0}, //

	// 4-bit codewords (2 duplicates each)
	{1, 0}, // 00100
	{1, 0}, //
	{2, 0}, // 00110
	{2, 0}, //
	{3, 0}, // 01000
	{3, 0}, //
	{4, 0}, // 01010
	{4, 0}, //
	{5, 0}, // 01100
	{5, 0}, //

	// 5-bit codewords
	{6, 0},  // 01110
	{7, 0},  // 01111
	{8, 0},  // 10000
	{9, 0},  // 10001
	{10, 0}, // 10010
	{11, 0}, // 10011
	{12, 0}, // 10100

	// 6-bit codewords
	{13, 1}, // 10101
	{15, 1}, // 10110
	{17, 1}, // 10111
	{19, 1}, // 11000
	{21, 1}, // 11001

	// 7-bit codewords
	{23, 2}, // 11010
	{27, 2}, // 11011
	{31, 2}, // 11100

	// 7/8-bit codewords
	{35, 3}, // 11101

	// 8-bit codewords
	{43, 3}, // 11110

	// 8/9/10-bit codewords
	{51, 5}, // 11111

	// Size of second level table is 51 + 32 = 83
}

// hcb8_2 is the second-step lookup table for codebook 8.
// Gives size of codeword and actual data (x, y).
// This is an UNSIGNED PAIR codebook (values 0-7).
//
// Copied from: ~/dev/faad2/libfaad/codebook/hcb_8.h:95-175 (hcb8_2[83])
var hcb8_2 = [83]HCB2Pair{
	// 3-bit codeword
	{3, 1, 1},

	// 4-bit codewords
	{4, 2, 1},
	{4, 1, 0},
	{4, 1, 2},
	{4, 0, 1},
	{4, 2, 2},

	// 5-bit codewords
	{5, 0, 0},
	{5, 2, 0},
	{5, 0, 2},
	{5, 3, 1},
	{5, 1, 3},
	{5, 3, 2},
	{5, 2, 3},

	// 6-bit codewords
	{6, 3, 3},
	{6, 4, 1},
	{6, 1, 4},
	{6, 4, 2},
	{6, 2, 4},
	{6, 3, 0},
	{6, 0, 3},
	{6, 4, 3},
	{6, 3, 4},
	{6, 5, 2},

	// 7-bit codewords
	{7, 5, 1},
	{7, 2, 5},
	{7, 1, 5},
	{7, 5, 3},
	{7, 3, 5},
	{7, 4, 4},
	{7, 5, 4},
	{7, 0, 4},
	{7, 4, 5},
	{7, 4, 0},
	{7, 2, 6},
	{7, 6, 2},

	// 7/8-bit codewords
	{7, 6, 1}, {7, 6, 1},
	{7, 1, 6}, {7, 1, 6},
	{8, 3, 6},
	{8, 6, 3},
	{8, 5, 5},
	{8, 5, 0},

	// 8-bit codewords
	{8, 6, 4},
	{8, 0, 5},
	{8, 4, 6},
	{8, 7, 1},
	{8, 7, 2},
	{8, 2, 7},
	{8, 6, 5},
	{8, 7, 3},

	// 8/9/10-bit codewords
	{8, 1, 7}, {8, 1, 7}, {8, 1, 7}, {8, 1, 7},
	{8, 5, 6}, {8, 5, 6}, {8, 5, 6}, {8, 5, 6},
	{8, 3, 7}, {8, 3, 7}, {8, 3, 7}, {8, 3, 7},
	{9, 6, 6}, {9, 6, 6},
	{9, 7, 4}, {9, 7, 4},
	{9, 6, 0}, {9, 6, 0},
	{9, 4, 7}, {9, 4, 7},
	{9, 0, 6}, {9, 0, 6},
	{9, 7, 5}, {9, 7, 5},
	{9, 7, 6}, {9, 7, 6},
	{9, 6, 7}, {9, 6, 7},
	{10, 5, 7},
	{10, 7, 0},
	{10, 0, 7},
	{10, 7, 7},
}
