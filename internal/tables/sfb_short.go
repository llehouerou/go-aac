// Package tables contains lookup tables for AAC decoding.
// This file contains Scalefactor Band (SFB) offset tables for short windows (128 samples).
// Ported from: ~/dev/faad2/libfaad/specrec.c:66-285
package tables

// NumSWB128Window contains the number of scale factor window bands
// for 128-sample short windows at each sample rate index.
// Source: ~/dev/faad2/libfaad/specrec.c:86-89
var NumSWB128Window = [12]uint8{
	12, 12, 12, 14, 14, 14, 15, 15, 15, 15, 15, 15,
}

// SWBOffset128_96 contains SFB offsets for 96kHz/88.2kHz at 128 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:98-101
var SWBOffset128_96 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 92, 128,
}

// SWBOffset128_64 contains SFB offsets for 64kHz at 128 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:111-114
var SWBOffset128_64 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 92, 128,
}

// SWBOffset128_48 contains SFB offsets for 48kHz/44.1kHz/32kHz at 128 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:140-143
var SWBOffset128_48 = []uint16{
	0, 4, 8, 12, 16, 20, 28, 36, 44, 56, 68, 80, 96, 112, 128,
}

// SWBOffset128_24 contains SFB offsets for 24kHz/22.05kHz at 128 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:192-195
var SWBOffset128_24 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 36, 44, 52, 64, 76, 92, 108, 128,
}

// SWBOffset128_16 contains SFB offsets for 16kHz/12kHz/11.025kHz at 128 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:204-207
var SWBOffset128_16 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 40, 48, 60, 72, 88, 108, 128,
}

// SWBOffset128_8 contains SFB offsets for 8kHz at 128 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:216-219
var SWBOffset128_8 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 36, 44, 52, 60, 72, 88, 108, 128,
}

// SWBOffset128Window maps sample rate index to the appropriate short window SFB offset table.
// Source: ~/dev/faad2/libfaad/specrec.c:271-285
var SWBOffset128Window = [12][]uint16{
	SWBOffset128_96, // 96000
	SWBOffset128_96, // 88200
	SWBOffset128_64, // 64000
	SWBOffset128_48, // 48000
	SWBOffset128_48, // 44100
	SWBOffset128_48, // 32000
	SWBOffset128_24, // 24000
	SWBOffset128_24, // 22050
	SWBOffset128_16, // 16000
	SWBOffset128_16, // 12000
	SWBOffset128_16, // 11025
	SWBOffset128_8,  // 8000
}
