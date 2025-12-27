// Package tables contains lookup tables for AAC decoding.
// This file contains Scalefactor Band (SFB) offset tables for long windows.
// Ported from: ~/dev/faad2/libfaad/specrec.c:66-235
package tables

// NumSWB1024Window contains the number of scale factor window bands
// for 1024-sample long windows at each sample rate index.
// Source: ~/dev/faad2/libfaad/specrec.c:81-84
var NumSWB1024Window = [12]uint8{
	41, 41, 47, 49, 49, 51, 47, 47, 43, 43, 43, 40,
}

// NumSWB960Window contains the number of scale factor window bands
// for 960-sample long windows at each sample rate index.
// Source: ~/dev/faad2/libfaad/specrec.c:76-79
var NumSWB960Window = [12]uint8{
	40, 40, 45, 49, 49, 49, 46, 46, 42, 42, 42, 40,
}

// SWBOffset1024_96 contains SFB offsets for 96kHz/88.2kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:91-96
var SWBOffset1024_96 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56,
	64, 72, 80, 88, 96, 108, 120, 132, 144, 156, 172, 188, 212, 240,
	276, 320, 384, 448, 512, 576, 640, 704, 768, 832, 896, 960, 1024,
}

// SWBOffset1024_64 contains SFB offsets for 64kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:103-109
var SWBOffset1024_64 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56,
	64, 72, 80, 88, 100, 112, 124, 140, 156, 172, 192, 216, 240, 268,
	304, 344, 384, 424, 464, 504, 544, 584, 624, 664, 704, 744, 784, 824,
	864, 904, 944, 984, 1024,
}

// SWBOffset1024_48 contains SFB offsets for 48kHz/44.1kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:116-122
var SWBOffset1024_48 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 48, 56, 64, 72,
	80, 88, 96, 108, 120, 132, 144, 160, 176, 196, 216, 240, 264, 292,
	320, 352, 384, 416, 448, 480, 512, 544, 576, 608, 640, 672, 704, 736,
	768, 800, 832, 864, 896, 928, 1024,
}

// SWBOffset1024_32 contains SFB offsets for 32kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:145-151
var SWBOffset1024_32 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 48, 56, 64, 72,
	80, 88, 96, 108, 120, 132, 144, 160, 176, 196, 216, 240, 264, 292,
	320, 352, 384, 416, 448, 480, 512, 544, 576, 608, 640, 672, 704, 736,
	768, 800, 832, 864, 896, 928, 960, 992, 1024,
}

// SWBOffset1024_24 contains SFB offsets for 24kHz/22.05kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:169-175
var SWBOffset1024_24 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 52, 60, 68,
	76, 84, 92, 100, 108, 116, 124, 136, 148, 160, 172, 188, 204, 220,
	240, 260, 284, 308, 336, 364, 396, 432, 468, 508, 552, 600, 652, 704,
	768, 832, 896, 960, 1024,
}

// SWBOffset1024_16 contains SFB offsets for 16kHz/12kHz/11.025kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:197-202
var SWBOffset1024_16 = []uint16{
	0, 8, 16, 24, 32, 40, 48, 56, 64, 72, 80, 88, 100, 112, 124,
	136, 148, 160, 172, 184, 196, 212, 228, 244, 260, 280, 300, 320, 344,
	368, 396, 424, 456, 492, 532, 572, 616, 664, 716, 772, 832, 896, 960, 1024,
}

// SWBOffset1024_8 contains SFB offsets for 8kHz at 1024 samples.
// Source: ~/dev/faad2/libfaad/specrec.c:209-214
var SWBOffset1024_8 = []uint16{
	0, 12, 24, 36, 48, 60, 72, 84, 96, 108, 120, 132, 144, 156, 172,
	188, 204, 220, 236, 252, 268, 288, 308, 328, 348, 372, 396, 420, 448,
	476, 508, 544, 580, 620, 664, 712, 764, 820, 880, 944, 1024,
}

// SWBOffset1024Window maps sample rate index to the appropriate SFB offset table.
// Source: ~/dev/faad2/libfaad/specrec.c:221-235
var SWBOffset1024Window = [12][]uint16{
	SWBOffset1024_96, // 96000
	SWBOffset1024_96, // 88200
	SWBOffset1024_64, // 64000
	SWBOffset1024_48, // 48000
	SWBOffset1024_48, // 44100
	SWBOffset1024_32, // 32000
	SWBOffset1024_24, // 24000
	SWBOffset1024_24, // 22050
	SWBOffset1024_16, // 16000
	SWBOffset1024_16, // 12000
	SWBOffset1024_16, // 11025
	SWBOffset1024_8,  // 8000
}
