// Copyright 2024 The go-aac Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syntax

import (
	"os"
	"testing"
)

func TestParseSCE_FAAD2Reference(t *testing.T) {
	// Skip if no reference data available
	refDir := os.Getenv("FAAD2_REF_DIR")
	if refDir == "" {
		t.Skip("FAAD2_REF_DIR not set - skipping reference comparison")
	}

	// TODO: Implement detailed FAAD2 comparison
	// 1. Load mono AAC test file
	// 2. Parse ADTS header to get configuration
	// 3. Parse SCE and compare spectral data against FAAD2 reference
	//
	// Test files to use:
	// - testdata/generated/aac_lc/44100_16_mono_128k/*.aac
	//
	// Reference generation:
	// ./scripts/check_faad2 testdata/test_mono.aac
	// Reference data in: /tmp/faad2_ref_test_mono/

	_ = refDir // silence unused variable warning
	t.Skip("TODO: Implement FAAD2 reference comparison for SCE")
}

func TestParseLFE_FAAD2Reference(t *testing.T) {
	// Skip if no reference data available
	refDir := os.Getenv("FAAD2_REF_DIR")
	if refDir == "" {
		t.Skip("FAAD2_REF_DIR not set - skipping reference comparison")
	}

	// TODO: Implement detailed FAAD2 comparison for LFE
	// Need a 5.1 surround test file to extract LFE data

	_ = refDir // silence unused variable warning
	t.Skip("TODO: Implement FAAD2 reference comparison for LFE")
}
