package mdct

import "github.com/llehouerou/go-aac/internal/fft"

// MDCT twiddle factor tables.
// These tables contain pre-computed sin/cos values for the pre/post rotations.
//
// The formula for each entry k (0 <= k < N/4) is:
//   Re = scale * cos(2*PI*(k+1/8) / N)
//   Im = scale * sin(2*PI*(k+1/8) / N)
// where scale = sqrt(N) for floating point.
//
// Ported from: mdct_tab_2048, mdct_tab_256 in ~/dev/faad2/libfaad/mdct_tab.h
//
// TODO(Task 2): Port actual values from FAAD2 mdct_tab.h

// mdctTab2048 contains twiddle factors for N=2048 (long blocks).
// Size: 512 entries (N/4)
var mdctTab2048 [512]fft.Complex

// mdctTab256 contains twiddle factors for N=256 (short blocks).
// Size: 64 entries (N/4)
var mdctTab256 [64]fft.Complex
