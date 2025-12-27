// Package huffman implements AAC Huffman decoding.
package huffman

import (
	"testing"
	"unsafe"
)

func TestHCBStructSize(t *testing.T) {
	// HCB struct should be 2 bytes (offset + extra_bits)
	if size := unsafe.Sizeof(HCB{}); size != 2 {
		t.Errorf("HCB size = %d, want 2", size)
	}
}

func TestHCB2QuadStructSize(t *testing.T) {
	// HCB2Quad struct should be 5 bytes (bits + x + y + v + w)
	if size := unsafe.Sizeof(HCB2Quad{}); size != 5 {
		t.Errorf("HCB2Quad size = %d, want 5", size)
	}
}

func TestHCB2PairStructSize(t *testing.T) {
	// HCB2Pair struct should be 3 bytes (bits + x + y)
	if size := unsafe.Sizeof(HCB2Pair{}); size != 3 {
		t.Errorf("HCB2Pair size = %d, want 3", size)
	}
}

func TestHCBBinQuadStructSize(t *testing.T) {
	// HCBBinQuad struct should be 5 bytes (is_leaf + data[4])
	if size := unsafe.Sizeof(HCBBinQuad{}); size != 5 {
		t.Errorf("HCBBinQuad size = %d, want 5", size)
	}
}

func TestHCBBinPairStructSize(t *testing.T) {
	// HCBBinPair struct should be 3 bytes (is_leaf + data[2])
	if size := unsafe.Sizeof(HCBBinPair{}); size != 3 {
		t.Errorf("HCBBinPair size = %d, want 3", size)
	}
}
