// decoder_test.go
package aac

import "testing"

func TestDecoder_New(t *testing.T) {
	dec := NewDecoder()
	if dec == nil {
		t.Fatal("NewDecoder returned nil")
	}

	// Verify default configuration
	cfg := dec.Config()
	if cfg.OutputFormat != OutputFormat16Bit {
		t.Errorf("default output format: got %d, want %d", cfg.OutputFormat, OutputFormat16Bit)
	}
	if cfg.DefObjectType != ObjectTypeMain {
		t.Errorf("default object type: got %d, want %d (MAIN)", cfg.DefObjectType, ObjectTypeMain)
	}
	if cfg.DefSampleRate != 44100 {
		t.Errorf("default sample rate: got %d, want 44100", cfg.DefSampleRate)
	}

	// Verify frame length is 1024 (standard AAC)
	if dec.frameLength != 1024 {
		t.Errorf("default frame length: got %d, want 1024", dec.frameLength)
	}

	// Verify RNG seeds match FAAD2 (decoder.c:151-153)
	if dec.rngState1 != 0x2bb431ea {
		t.Errorf("rngState1: got 0x%x, want 0x2bb431ea", dec.rngState1)
	}
	if dec.rngState2 != 0x206155b7 {
		t.Errorf("rngState2: got 0x%x, want 0x206155b7", dec.rngState2)
	}
}

func TestDecoder_SetConfiguration(t *testing.T) {
	dec := NewDecoder()

	cfg := Config{
		DefObjectType: ObjectTypeHEAAC,
		DefSampleRate: 48000,
		OutputFormat:  OutputFormatFloat,
		DownMatrix:    true,
	}

	dec.SetConfiguration(cfg)

	got := dec.Config()
	if got.DefObjectType != cfg.DefObjectType {
		t.Errorf("DefObjectType: got %d, want %d", got.DefObjectType, cfg.DefObjectType)
	}
	if got.DefSampleRate != cfg.DefSampleRate {
		t.Errorf("DefSampleRate: got %d, want %d", got.DefSampleRate, cfg.DefSampleRate)
	}
	if got.OutputFormat != cfg.OutputFormat {
		t.Errorf("OutputFormat: got %d, want %d", got.OutputFormat, cfg.OutputFormat)
	}
	if got.DownMatrix != cfg.DownMatrix {
		t.Errorf("DownMatrix: got %v, want %v", got.DownMatrix, cfg.DownMatrix)
	}
}

func TestDecoder_Constants_MatchFAAD2(t *testing.T) {
	// Verify constants match FAAD2's structs.h:43-44 definitions
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"maxChannels", maxChannels, 64},
		{"maxSyntaxElements", maxSyntaxElements, 48},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s: got %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestDecoder_allocateChannelBuffers(t *testing.T) {
	dec := NewDecoder()

	// Allocate for 2 channels
	err := dec.allocateChannelBuffers(2)
	if err != nil {
		t.Fatalf("allocateChannelBuffers failed: %v", err)
	}

	// Check timeOut buffers
	for ch := 0; ch < 2; ch++ {
		if dec.timeOut[ch] == nil {
			t.Errorf("timeOut[%d] not allocated", ch)
		}
		if len(dec.timeOut[ch]) != int(dec.frameLength) {
			t.Errorf("timeOut[%d] length: got %d, want %d", ch, len(dec.timeOut[ch]), dec.frameLength)
		}
	}

	// Check fbIntermed buffers
	for ch := 0; ch < 2; ch++ {
		if dec.fbIntermed[ch] == nil {
			t.Errorf("fbIntermed[%d] not allocated", ch)
		}
		if len(dec.fbIntermed[ch]) != int(dec.frameLength) {
			t.Errorf("fbIntermed[%d] length: got %d, want %d", ch, len(dec.fbIntermed[ch]), dec.frameLength)
		}
	}
}

func TestDecoder_allocateChannelBuffers_Idempotent(t *testing.T) {
	dec := NewDecoder()

	// Allocate for 2 channels
	err := dec.allocateChannelBuffers(2)
	if err != nil {
		t.Fatalf("first allocateChannelBuffers failed: %v", err)
	}

	// Store pointers to original buffers
	origTimeOut0 := dec.timeOut[0]
	origTimeOut1 := dec.timeOut[1]
	origFbIntermed0 := dec.fbIntermed[0]
	origFbIntermed1 := dec.fbIntermed[1]

	// Call again - should be idempotent (no-op)
	err = dec.allocateChannelBuffers(2)
	if err != nil {
		t.Fatalf("second allocateChannelBuffers failed: %v", err)
	}

	// Verify buffers are the same (not reallocated)
	if &dec.timeOut[0][0] != &origTimeOut0[0] {
		t.Error("timeOut[0] was reallocated, should be idempotent")
	}
	if &dec.timeOut[1][0] != &origTimeOut1[0] {
		t.Error("timeOut[1] was reallocated, should be idempotent")
	}
	if &dec.fbIntermed[0][0] != &origFbIntermed0[0] {
		t.Error("fbIntermed[0] was reallocated, should be idempotent")
	}
	if &dec.fbIntermed[1][0] != &origFbIntermed1[0] {
		t.Error("fbIntermed[1] was reallocated, should be idempotent")
	}
}

func TestDecoder_allocateChannelBuffers_TooManyChannels(t *testing.T) {
	dec := NewDecoder()

	// Try to allocate more than maxChannels
	err := dec.allocateChannelBuffers(maxChannels + 1)
	if err != ErrInvalidNumChannels {
		t.Errorf("expected ErrInvalidNumChannels, got %v", err)
	}
}

func TestDecoder_allocateChannelBuffers_ZeroChannels(t *testing.T) {
	dec := NewDecoder()

	// Zero channels should be valid (no-op)
	err := dec.allocateChannelBuffers(0)
	if err != nil {
		t.Errorf("allocateChannelBuffers(0) failed: %v", err)
	}
}

func TestDecoder_allocateLTPBuffers(t *testing.T) {
	dec := NewDecoder()

	// Allocate LTP for 2 channels
	dec.allocateLTPBuffers(2)

	// LTP buffers are 2*frameLength for overlap storage
	expectedLen := 2 * int(dec.frameLength)

	for ch := 0; ch < 2; ch++ {
		if dec.ltPredStat[ch] == nil {
			t.Errorf("ltPredStat[%d] not allocated", ch)
		}
		if len(dec.ltPredStat[ch]) != expectedLen {
			t.Errorf("ltPredStat[%d] length: got %d, want %d", ch, len(dec.ltPredStat[ch]), expectedLen)
		}
	}
}

func TestDecoder_Close(t *testing.T) {
	dec := NewDecoder()

	// Allocate some buffers
	_ = dec.allocateChannelBuffers(2)
	dec.allocateLTPBuffers(2)

	// Simulate component references
	dec.fb = struct{}{}  // Non-nil value
	dec.drc = struct{}{} // Non-nil value
	dec.pce = struct{}{} // Non-nil value

	// Close should not panic
	dec.Close()

	// Verify per-channel buffers are nil'd (helps GC)
	for ch := 0; ch < 2; ch++ {
		if dec.timeOut[ch] != nil {
			t.Errorf("timeOut[%d] not cleared after Close", ch)
		}
		if dec.fbIntermed[ch] != nil {
			t.Errorf("fbIntermed[%d] not cleared after Close", ch)
		}
		if dec.ltPredStat[ch] != nil {
			t.Errorf("ltPredStat[%d] not cleared after Close", ch)
		}
	}

	// Verify component references are nil'd
	if dec.fb != nil {
		t.Error("fb not cleared after Close")
	}
	if dec.drc != nil {
		t.Error("drc not cleared after Close")
	}
	if dec.pce != nil {
		t.Error("pce not cleared after Close")
	}
}

func TestDecoder_StreamInfo(t *testing.T) {
	dec := NewDecoder()

	// Simulate initialized state
	dec.sfIndex = 4 // 44100 Hz
	dec.objectType = uint8(ObjectTypeLC)
	dec.channelConfiguration = 2 // Stereo
	dec.frameLength = 1024

	if dec.SampleRate() != 44100 {
		t.Errorf("SampleRate: got %d, want 44100", dec.SampleRate())
	}

	if dec.Channels() != 2 {
		t.Errorf("Channels: got %d, want 2", dec.Channels())
	}

	if dec.FrameLength() != 1024 {
		t.Errorf("FrameLength: got %d, want 1024", dec.FrameLength())
	}

	if dec.ObjectType() != ObjectTypeLC {
		t.Errorf("ObjectType: got %d, want %d", dec.ObjectType(), ObjectTypeLC)
	}
}

func TestDecoder_PostSeekReset(t *testing.T) {
	dec := NewDecoder()

	// Set some state
	dec.frame = 100
	dec.postSeekResetFlag = false

	// Reset after seek with specific frame
	dec.PostSeekReset(50)

	// Verify flag is set
	if !dec.postSeekResetFlag {
		t.Error("postSeekResetFlag not set after PostSeekReset")
	}

	// Frame should be updated
	if dec.frame != 50 {
		t.Errorf("frame not updated: got %d, want 50", dec.frame)
	}

	// Test with -1 (don't change frame)
	dec.frame = 100
	dec.PostSeekReset(-1)
	if dec.frame != 100 {
		t.Errorf("frame changed with -1: got %d, want 100", dec.frame)
	}
}

func TestInitResult_Fields(t *testing.T) {
	// Verify InitResult type exists with expected fields
	result := InitResult{
		SampleRate: 44100,
		Channels:   2,
		BytesRead:  0,
	}

	if result.SampleRate != 44100 {
		t.Errorf("SampleRate: got %d, want 44100", result.SampleRate)
	}
	if result.Channels != 2 {
		t.Errorf("Channels: got %d, want 2", result.Channels)
	}
}

func TestDecoder_Init_ADTS(t *testing.T) {
	// Create test ADTS data (minimal valid header)
	// Syncword: 0xFFF, ID: 0, Layer: 0, ProtAbsent: 1, Profile: 1 (LC)
	// SFIndex: 4 (44100Hz), PrivateBit: 0, ChanConfig: 2 (stereo)
	adtsHeader := []byte{
		0xFF, 0xF1, // syncword + id=0, layer=0, protection_absent=1
		0x50, 0x80, // profile=1(LC), sf_index=4(44100), private=0, chan_config=2, orig=0, home=0
		0x00, 0x1F, // copyright bits + frame_length (partial)
		0xFC, // frame_length + buffer_fullness (partial)
	}

	d := NewDecoder()
	result, err := d.Init(adtsHeader)

	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if result.SampleRate != 44100 {
		t.Errorf("SampleRate: got %d, want 44100", result.SampleRate)
	}
	if result.Channels != 2 {
		t.Errorf("Channels: got %d, want 2", result.Channels)
	}
	if result.BytesRead != 0 {
		t.Errorf("BytesRead: got %d, want 0 for ADTS", result.BytesRead)
	}
	if !d.adtsHeaderPresent {
		t.Error("adtsHeaderPresent should be true")
	}
}

func TestDecoder_Init_NilDecoder(t *testing.T) {
	var d *Decoder
	_, err := d.Init([]byte{0xFF, 0xF1})
	if err != ErrNilDecoder {
		t.Errorf("expected ErrNilDecoder, got %v", err)
	}
}

func TestDecoder_Init_NilBuffer(t *testing.T) {
	d := NewDecoder()
	_, err := d.Init(nil)
	if err != ErrNilBuffer {
		t.Errorf("expected ErrNilBuffer, got %v", err)
	}
}

func TestDecoder_Init_BufferTooSmall(t *testing.T) {
	d := NewDecoder()
	_, err := d.Init([]byte{0xFF})
	if err != ErrBufferTooSmall {
		t.Errorf("expected ErrBufferTooSmall, got %v", err)
	}
}

func TestDecoder_Init_InvalidObjectType(t *testing.T) {
	// Create ADTS header with profile=2 (SSR, which is not supported)
	// Syncword: 0xFFF, ID: 0, Layer: 0, ProtAbsent: 1, Profile: 2 (SSR)
	// SFIndex: 4 (44100Hz), PrivateBit: 0, ChanConfig: 2 (stereo)
	//
	// Byte layout after syncword (0xFFF):
	// - Byte 1 (0xF1): syncword[4:11]=1111, id=0, layer=00, protection_absent=1
	// - Byte 2: profile(2) + sf_index(4) + private(1) + chan_config_high(1)
	//   For SSR: profile=10, sf_index=0100, private=0, chan_high=0 -> 10010000 = 0x90
	// - Byte 3: chan_config_low(2) + orig(1) + home(1) + copyright_id_bit(1) + copyright_id_start(1) + frame_len_high(2)
	//   For chan=2: chan_low=10, orig=0, home=0 -> 10000000 = 0x80 (with frame_len bits)
	adtsHeader := []byte{
		0xFF, 0xF1, // syncword + id=0, layer=0, protection_absent=1
		0x90, 0x80, // profile=2(SSR), sf_index=4(44100), private=0, chan_config=2, orig=0, home=0
		0x00, 0x1F, // copyright bits + frame_length (partial)
		0xFC, // frame_length + buffer_fullness (partial)
	}

	d := NewDecoder()
	_, err := d.Init(adtsHeader)
	if err != ErrUnsupportedObjectType {
		t.Errorf("expected ErrUnsupportedObjectType, got %v", err)
	}
}

func TestDecoder_Init_FilterBankInitialized(t *testing.T) {
	// Create valid ADTS header for LC profile
	adtsHeader := []byte{
		0xFF, 0xF1, // syncword + id=0, layer=0, protection_absent=1
		0x50, 0x80, // profile=1(LC), sf_index=4(44100), private=0, chan_config=2, orig=0, home=0
		0x00, 0x1F, // copyright bits + frame_length (partial)
		0xFC, // frame_length + buffer_fullness (partial)
	}

	d := NewDecoder()
	_, err := d.Init(adtsHeader)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Filter bank should be initialized
	if d.fb == nil {
		t.Error("filter bank not initialized after Init")
	}
}

func TestDecoder_Init_ADIF_NotSet_For_ADTS(t *testing.T) {
	// Verify that ADTS data does NOT set adifHeaderPresent
	d := NewDecoder()

	// ADTS data (not ADIF)
	adtsData := []byte{0xFF, 0xF1, 0x50, 0x80, 0x00, 0x1F, 0xFC}
	_, err := d.Init(adtsData)

	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if d.adifHeaderPresent {
		t.Error("adifHeaderPresent should be false for ADTS data")
	}
	if !d.adtsHeaderPresent {
		t.Error("adtsHeaderPresent should be true for ADTS data")
	}
}

func TestDecoder_Init_ADIF_Detected(t *testing.T) {
	// Create minimal ADIF header with "ADIF" magic
	// ADIF format is rare and complex; for now we just detect and return an error
	adifData := []byte{
		'A', 'D', 'I', 'F', // Magic signature
		0x00, // copyright_id_present = 0
		// Additional fields would follow but we only detect magic for now
	}

	d := NewDecoder()
	_, err := d.Init(adifData)

	// ADIF should be detected (returns ErrADIFNotSupported for now)
	if err != ErrADIFNotSupported {
		t.Errorf("expected ErrADIFNotSupported for ADIF data, got %v", err)
	}
	if !d.adifHeaderPresent {
		t.Error("adifHeaderPresent should be true for ADIF data")
	}
	if d.adtsHeaderPresent {
		t.Error("adtsHeaderPresent should be false for ADIF data")
	}
}

// Tests for Init2() - AudioSpecificConfig parsing

func TestDecoder_Init2_BasicASC(t *testing.T) {
	// Minimal AudioSpecificConfig for AAC-LC, 44100Hz, stereo
	// Object type: 2 (LC), SF index: 4 (44100), Channels: 2
	//
	// Bit layout:
	// - 5 bits: objectType = 2 (00010)
	// - 4 bits: samplingFrequencyIndex = 4 (0100)
	// - 4 bits: channelConfiguration = 2 (0010)
	// - Rest: GASpecificConfig starts
	//
	// Binary: 00010 0100 0010 0... = 0001 0010 0001 0...
	// After GASpecificConfig (frameLengthFlag=0, dependsOnCoreCoder=0, extensionFlag=0):
	// Full: 00010 0100 0010 000 = 0x12 0x10
	asc := []byte{0x12, 0x10}

	d := NewDecoder()
	result, err := d.Init2(asc)

	if err != nil {
		t.Fatalf("Init2 failed: %v", err)
	}
	if result.SampleRate != 44100 {
		t.Errorf("SampleRate: got %d, want 44100", result.SampleRate)
	}
	if result.Channels != 2 {
		t.Errorf("Channels: got %d, want 2", result.Channels)
	}
	if result.BytesRead != 0 {
		t.Errorf("BytesRead: got %d, want 0 for Init2", result.BytesRead)
	}
	if d.adtsHeaderPresent || d.adifHeaderPresent {
		t.Error("Header present flags should be false for Init2")
	}
	if d.objectType != 2 {
		t.Errorf("objectType: got %d, want 2 (LC)", d.objectType)
	}
	if d.sfIndex != 4 {
		t.Errorf("sfIndex: got %d, want 4 (44100Hz)", d.sfIndex)
	}
	if d.channelConfiguration != 2 {
		t.Errorf("channelConfiguration: got %d, want 2", d.channelConfiguration)
	}
}

func TestDecoder_Init2_NilDecoder(t *testing.T) {
	var d *Decoder
	_, err := d.Init2([]byte{0x12, 0x10})
	if err != ErrNilDecoder {
		t.Errorf("expected ErrNilDecoder, got %v", err)
	}
}

func TestDecoder_Init2_NilBuffer(t *testing.T) {
	d := NewDecoder()
	_, err := d.Init2(nil)
	if err != ErrNilBuffer {
		t.Errorf("expected ErrNilBuffer, got %v", err)
	}
}

func TestDecoder_Init2_BufferTooSmall(t *testing.T) {
	d := NewDecoder()
	_, err := d.Init2([]byte{0x12})
	if err != ErrBufferTooSmall {
		t.Errorf("expected ErrBufferTooSmall, got %v", err)
	}
}

func TestDecoder_Init2_InvalidObjectType(t *testing.T) {
	// ASC with object type 0 (NULL, not supported)
	// 5 bits: objectType = 0 (00000)
	// 4 bits: samplingFrequencyIndex = 4 (0100)
	// 4 bits: channelConfiguration = 2 (0010)
	// Binary: 00000 0100 0010 0... = 0x00 0x22 (adjusted bits)
	asc := []byte{0x02, 0x10}

	d := NewDecoder()
	_, err := d.Init2(asc)
	if err != ErrUnsupportedObjectType {
		t.Errorf("expected ErrUnsupportedObjectType, got %v", err)
	}
}

func TestDecoder_Init2_SSRObjectType(t *testing.T) {
	// ASC with object type 3 (SSR, not supported)
	// 5 bits: objectType = 3 (00011)
	// 4 bits: samplingFrequencyIndex = 4 (0100)
	// 4 bits: channelConfiguration = 2 (0010)
	// Binary: 00011 0100 0010 0... = 0x1A 0x10
	asc := []byte{0x1A, 0x10}

	d := NewDecoder()
	_, err := d.Init2(asc)
	if err != ErrUnsupportedObjectType {
		t.Errorf("expected ErrUnsupportedObjectType for SSR, got %v", err)
	}
}

func TestDecoder_Init2_MainProfile(t *testing.T) {
	// ASC with object type 1 (Main), 48000Hz, stereo
	// 5 bits: objectType = 1 (00001)
	// 4 bits: samplingFrequencyIndex = 3 (0011) = 48000Hz
	// 4 bits: channelConfiguration = 2 (0010)
	// Binary: 00001 0011 0010 0... = 0x09 0x90
	asc := []byte{0x09, 0x90}

	d := NewDecoder()
	result, err := d.Init2(asc)

	if err != nil {
		t.Fatalf("Init2 failed for Main profile: %v", err)
	}
	if result.SampleRate != 48000 {
		t.Errorf("SampleRate: got %d, want 48000", result.SampleRate)
	}
	if result.Channels != 2 {
		t.Errorf("Channels: got %d, want 2", result.Channels)
	}
	if d.objectType != 1 {
		t.Errorf("objectType: got %d, want 1 (Main)", d.objectType)
	}
}

func TestDecoder_Init2_LTPProfile(t *testing.T) {
	// ASC with object type 4 (LTP), 32000Hz, mono
	// 5 bits: objectType = 4 (00100)
	// 4 bits: samplingFrequencyIndex = 5 (0101) = 32000Hz
	// 4 bits: channelConfiguration = 1 (0001)
	// Binary: 00100 0101 0001 0... = 0x22 0x88
	asc := []byte{0x22, 0x88}

	d := NewDecoder()
	result, err := d.Init2(asc)

	if err != nil {
		t.Fatalf("Init2 failed for LTP profile: %v", err)
	}
	if result.SampleRate != 32000 {
		t.Errorf("SampleRate: got %d, want 32000", result.SampleRate)
	}
	if result.Channels != 1 {
		t.Errorf("Channels: got %d, want 1", result.Channels)
	}
	if d.objectType != 4 {
		t.Errorf("objectType: got %d, want 4 (LTP)", d.objectType)
	}
}

func TestDecoder_Init2_InvalidSampleRate(t *testing.T) {
	// ASC with invalid sample rate index (13, 14, or 15 with no explicit frequency)
	// Using index 13 which maps to 0 Hz
	// 5 bits: objectType = 2 (00010)
	// 4 bits: samplingFrequencyIndex = 13 (1101)
	// 4 bits: channelConfiguration = 2 (0010)
	// Binary: 00010 1101 0010 0... = 0x16 0x90
	asc := []byte{0x16, 0x90}

	d := NewDecoder()
	_, err := d.Init2(asc)
	if err != ErrInvalidSampleRate {
		t.Errorf("expected ErrInvalidSampleRate, got %v", err)
	}
}

func TestDecoder_Init2_ExplicitSampleRate(t *testing.T) {
	// ASC with explicit sample rate (sfIndex = 15, then 24-bit sample rate)
	// 5 bits: objectType = 2 (00010)
	// 4 bits: samplingFrequencyIndex = 15 (1111)
	// 24 bits: explicit sample rate = 44100 = 0x00AC44
	// 4 bits: channelConfiguration = 2 (0010)
	//
	// Build as 64-bit left-aligned: 0x1780562210000000
	// Byte 0: 0x17, Byte 1: 0x80, Byte 2: 0x56, Byte 3: 0x22, Byte 4: 0x10
	asc := []byte{0x17, 0x80, 0x56, 0x22, 0x10}

	d := NewDecoder()
	result, err := d.Init2(asc)

	if err != nil {
		t.Fatalf("Init2 failed with explicit sample rate: %v", err)
	}
	if result.SampleRate != 44100 {
		t.Errorf("SampleRate: got %d, want 44100", result.SampleRate)
	}
	if result.Channels != 2 {
		t.Errorf("Channels: got %d, want 2", result.Channels)
	}
}

func TestDecoder_Init2_FilterBankInitialized(t *testing.T) {
	asc := []byte{0x12, 0x10} // AAC-LC, 44100Hz, stereo

	d := NewDecoder()
	_, err := d.Init2(asc)
	if err != nil {
		t.Fatalf("Init2 failed: %v", err)
	}

	// Filter bank should be initialized (marker set)
	if d.fb == nil {
		t.Error("filter bank not initialized after Init2")
	}
}

func TestDecoder_Init2_ChannelConfigs(t *testing.T) {
	tests := []struct {
		name     string
		asc      []byte
		channels uint8
	}{
		// Mono: channels = 1
		// 00010 0100 0001 0... = 0x12 0x08
		{"mono", []byte{0x12, 0x08}, 1},
		// Stereo: channels = 2
		// 00010 0100 0010 0... = 0x12 0x10
		{"stereo", []byte{0x12, 0x10}, 2},
		// Surround 5.0: channels = 5
		// 00010 0100 0101 0... = 0x12 0x28
		{"5.0", []byte{0x12, 0x28}, 5},
		// Surround 5.1: channels = 6
		// 00010 0100 0110 0... = 0x12 0x30
		{"5.1", []byte{0x12, 0x30}, 6},
		// 7.1: channels = 7
		// 00010 0100 0111 0... = 0x12 0x38
		{"7.1", []byte{0x12, 0x38}, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder()
			result, err := d.Init2(tt.asc)
			if err != nil {
				t.Fatalf("Init2 failed: %v", err)
			}
			if result.Channels != tt.channels {
				t.Errorf("Channels: got %d, want %d", result.Channels, tt.channels)
			}
		})
	}
}

func TestDecoder_Init2_SampleRates(t *testing.T) {
	tests := []struct {
		name       string
		sfIndex    uint8
		sampleRate uint32
	}{
		{"96kHz", 0, 96000},
		{"88.2kHz", 1, 88200},
		{"64kHz", 2, 64000},
		{"48kHz", 3, 48000},
		{"44.1kHz", 4, 44100},
		{"32kHz", 5, 32000},
		{"24kHz", 6, 24000},
		{"22.05kHz", 7, 22050},
		{"16kHz", 8, 16000},
		{"12kHz", 9, 12000},
		{"11.025kHz", 10, 11025},
		{"8kHz", 11, 8000},
		{"7.35kHz", 12, 7350},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build ASC: objectType=2 (LC), sfIndex=tt.sfIndex, channels=2
			// 5 bits: objectType = 2 (00010)
			// 4 bits: sfIndex
			// 4 bits: channels = 2 (0010)
			//
			// Byte 0: 0001 0[sfIndex high 3 bits]
			// Byte 1: [sfIndex low 1 bit][channels 4 bits][GASpec...]
			byte0 := 0x10 | (tt.sfIndex >> 1)
			byte1 := ((tt.sfIndex & 0x01) << 7) | 0x10 // channels=2 in upper nibble position

			d := NewDecoder()
			result, err := d.Init2([]byte{byte0, byte1})
			if err != nil {
				t.Fatalf("Init2 failed: %v", err)
			}
			if result.SampleRate != tt.sampleRate {
				t.Errorf("SampleRate: got %d, want %d", result.SampleRate, tt.sampleRate)
			}
		})
	}
}
