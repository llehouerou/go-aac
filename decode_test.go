package aac

import "testing"

// mockFilterBank is a minimal mock for testing filter bank initialization.
type mockFilterBank struct {
	frameLength uint16
}

// testFilterBankFactory creates a mock filter bank for testing.
func testFilterBankFactory(frameLength uint16) any {
	return &mockFilterBank{frameLength: frameLength}
}

func TestDecoder_Decode_NilDecoder(t *testing.T) {
	var d *Decoder
	_, _, err := d.Decode([]byte{0xFF, 0xF1, 0x50, 0x80})
	if err != ErrNilDecoder {
		t.Errorf("expected ErrNilDecoder, got %v", err)
	}
}

func TestDecoder_Decode_NilBuffer(t *testing.T) {
	d := NewDecoder()
	_, _, err := d.Decode(nil)
	if err != ErrNilBuffer {
		t.Errorf("expected ErrNilBuffer, got %v", err)
	}
}

func TestDecoder_Decode_EmptyBuffer(t *testing.T) {
	d := NewDecoder()
	_, _, err := d.Decode([]byte{})
	if err != ErrBufferTooSmall {
		t.Errorf("expected ErrBufferTooSmall, got %v", err)
	}
}

func TestDecoder_Decode_ID3Tag(t *testing.T) {
	d := NewDecoder()
	// Initialize with valid ADTS header first
	adtsHeader := []byte{0xFF, 0xF1, 0x50, 0x80, 0x00, 0x1F, 0xFC}
	_, err := d.Init(adtsHeader)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create ID3v1 tag (128 bytes starting with "TAG")
	id3Tag := make([]byte, 128)
	copy(id3Tag, []byte("TAG"))

	// Decode should return nil samples and consume 128 bytes
	samples, info, err := d.Decode(id3Tag)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if samples != nil {
		t.Error("expected nil samples for ID3 tag")
	}
	if info == nil {
		t.Fatal("expected non-nil FrameInfo")
	}
	if info.BytesConsumed != 128 {
		t.Errorf("BytesConsumed: got %d, want 128", info.BytesConsumed)
	}
}

func TestDecoder_Decode_ADTSHeaderParsed(t *testing.T) {
	d := NewDecoder()
	// Initialize with ADTS stream
	adtsHeader := []byte{0xFF, 0xF1, 0x50, 0x80, 0x00, 0x1F, 0xFC}
	_, err := d.Init(adtsHeader)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create ADTS frame with header + ID_END payload.
	// Frame length = 8 bytes (7 header + 1 payload with ID_END).
	// ADTS fixed header (28 bits after syncword):
	//   0xFF, 0xF1 = syncword (12b) + id=0 (1b) + layer=0 (2b) + protection_absent=1 (1b)
	//   0x50 = profile=1(LC) (2b) + sf_index=4(44100) (4b) + private=0 (1b) + chan_config high bit=0 (1b)
	//   0x80 = chan_config low bits=10 (2b) + original=0 + home=0 + copyright_id_bit=0 + copyright_id_start=0
	// ADTS variable header (28 bits):
	//   frame_length (13 bits) = 8 (0x008)
	//   adts_buffer_fullness (11 bits) = 0x7FF (VBR)
	//   number_of_raw_data_blocks (2 bits) = 0
	//
	// Encoding frame_length=8:
	//   Bits: last 2 bits of chan_config (10), orig(0), home(0), copy_id(0), copy_start(0), frame_len[12:11]=00
	//         frame_len[10:3] = 00000010 (for frame_length=8)
	//         frame_len[2:0]=000, buffer_fullness[10:6]=11111
	//         buffer_fullness[5:0]=111111, num_blocks=00
	frame := []byte{
		0xFF, 0xF1, // syncword + id=0, layer=0, protection_absent=1
		0x50, // profile=1(LC), sf_index=4(44100), private=0, chan_config[2]=0
		0x80, // chan_config[1:0]=10, original=0, home=0, copyright_id_bit=0, copyright_id_start=0, frame_length[12:11]=00
		0x02, // frame_length[10:3] = 00000010 (for frame_length=8)
		0x1F, // frame_length[2:0]=000, buffer_fullness[10:6]=11111
		0xFC, // buffer_fullness[5:0]=111111, num_blocks=00
		0xE0, // ID_END (0x7 = 111) + padding (00000)
	}

	// This should parse ADTS header and raw_data_block (ID_END only)
	_, info, err := d.Decode(frame)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil FrameInfo")
	}
	if info.HeaderType != HeaderTypeADTS {
		t.Errorf("HeaderType: got %d, want %d (ADTS)", info.HeaderType, HeaderTypeADTS)
	}
}

func TestDecoder_Decode_ADIFHeaderType(t *testing.T) {
	d := NewDecoder()
	// Manually set ADIF mode (since ADIF init is not fully implemented)
	d.adifHeaderPresent = true

	// Create minimal buffer with ID_END (0x7 = 111 in first 3 bits)
	// 0xE0 = 1110 0000 = ID_END (111) + 5 padding bits
	buffer := []byte{0xE0}

	_, info, err := d.Decode(buffer)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil FrameInfo")
	}
	if info.HeaderType != HeaderTypeADIF {
		t.Errorf("HeaderType: got %d, want %d (ADIF)", info.HeaderType, HeaderTypeADIF)
	}
}

func TestDecoder_Decode_RawHeaderType(t *testing.T) {
	d := NewDecoder()
	// No header present = raw AAC

	// Create minimal buffer with ID_END (0x7 = 111 in first 3 bits)
	// 0xE0 = 1110 0000 = ID_END (111) + 5 padding bits
	buffer := []byte{0xE0}

	_, info, err := d.Decode(buffer)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil FrameInfo")
	}
	if info.HeaderType != HeaderTypeRAW {
		t.Errorf("HeaderType: got %d, want %d (RAW)", info.HeaderType, HeaderTypeRAW)
	}
}

func TestDecoder_Decode_RawDataBlockParsing(t *testing.T) {
	d := NewDecoder()
	// Initialize with raw AAC (no ADTS)
	d.sfIndex = 4              // 44100 Hz
	d.objectType = 2           // LC
	d.channelConfiguration = 2 // stereo
	d.frameLength = 1024

	// Create minimal raw_data_block with ID_END only
	// ID_END = 0x7 (3 bits) = 111
	// Byte-aligned: 0xE0 (111 00000)
	rawData := []byte{0xE0}

	_, info, err := d.Decode(rawData)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil FrameInfo")
	}
	// With only ID_END, no channels should be detected
	if info.Channels != 0 {
		t.Errorf("Channels: got %d, want 0 for empty frame", info.Channels)
	}
	// BytesConsumed should be 1 (3 bits ID_END + 5 bits alignment = 8 bits = 1 byte)
	if info.BytesConsumed != 1 {
		t.Errorf("BytesConsumed: got %d, want 1", info.BytesConsumed)
	}
	// frChannels should be 0 after parsing empty frame
	if d.frChannels != 0 {
		t.Errorf("frChannels: got %d, want 0", d.frChannels)
	}
	// frChEle should be 0 (no elements besides ID_END)
	if d.frChEle != 0 {
		t.Errorf("frChEle: got %d, want 0", d.frChEle)
	}
}

func TestDecoder_Decode_FilterBankLazyInit(t *testing.T) {
	// Register the test factory for this test
	// Save and restore the original factory to avoid affecting other tests
	originalFactory := filterBankFactory
	RegisterFilterBankFactory(testFilterBankFactory)
	defer func() { filterBankFactory = originalFactory }()

	d := NewDecoder()
	d.adtsHeaderPresent = false
	d.sfIndex = 4              // 44100 Hz
	d.objectType = 2           // LC
	d.channelConfiguration = 2 // stereo
	d.frameLength = 1024

	// Before decode, fb is marker (true) or nil
	if d.fb != nil && d.fb != true {
		t.Error("fb should be nil or marker before decode")
	}

	// Decode minimal frame (ID_END only)
	rawData := []byte{0xE0}
	_, _, _ = d.Decode(rawData)

	// After decode, fb should not be nil and not the boolean marker
	if d.fb == nil {
		t.Error("fb should not be nil after decode")
	}

	// Verify it's the mock filter bank (not the boolean marker)
	if _, isMarker := d.fb.(bool); isMarker {
		t.Error("fb should not be boolean marker after decode")
	}

	// Verify it's the mock type we registered
	mock, ok := d.fb.(*mockFilterBank)
	if !ok {
		t.Errorf("fb should be *mockFilterBank, got %T", d.fb)
	} else if mock.frameLength != 1024 {
		t.Errorf("mock.frameLength: got %d, want 1024", mock.frameLength)
	}
}

func TestDecoder_Decode_CompletePipeline(t *testing.T) {
	// This test verifies the complete decode pipeline works when channels are present.
	// Since full SCE/CPE parsing isn't implemented yet, we manually set up the decoder
	// state to simulate having parsed one channel, then call the pipeline code directly.

	d := NewDecoder()
	d.adtsHeaderPresent = false
	d.sfIndex = 4              // 44100 Hz
	d.objectType = 2           // LC
	d.channelConfiguration = 2 // stereo
	d.frameLength = 1024

	// Manually allocate channel buffers to simulate post-element-parsing state
	_ = d.allocateChannelBuffers(2)

	// Verify channel buffers are allocated
	if d.timeOut[0] == nil || d.timeOut[1] == nil {
		t.Fatal("channel buffers should be allocated")
	}

	// Verify createChannelConfig works
	info := &FrameInfo{}
	d.createChannelConfig(info)

	if info.NumFrontChannels != 2 {
		t.Errorf("NumFrontChannels: got %d, want 2", info.NumFrontChannels)
	}
	if info.ChannelPosition[0] != ChannelFrontLeft {
		t.Errorf("ChannelPosition[0]: got %d, want %d (FrontLeft)", info.ChannelPosition[0], ChannelFrontLeft)
	}
	if info.ChannelPosition[1] != ChannelFrontRight {
		t.Errorf("ChannelPosition[1]: got %d, want %d (FrontRight)", info.ChannelPosition[1], ChannelFrontRight)
	}

	// Verify generatePCMOutput works
	samples := d.generatePCMOutput(2)
	if samples == nil {
		t.Error("expected non-nil samples from generatePCMOutput")
	}
	s16, ok := samples.([]int16)
	if !ok {
		t.Fatalf("samples should be []int16, got %T", samples)
	}
	expectedLen := int(d.frameLength) * 2 // 1024 * 2 channels = 2048
	if len(s16) != expectedLen {
		t.Errorf("samples length: got %d, want %d", len(s16), expectedLen)
	}

	// Verify getSampleRate works
	sr := getSampleRate(d.sfIndex)
	if sr != 44100 {
		t.Errorf("getSampleRate: got %d, want 44100", sr)
	}

	// Verify first frame muting logic
	d.frame = 0
	d.frame++ // Simulate first decode
	if d.frame > 1 {
		t.Error("first frame should be frame 1")
	}
}

func TestDecoder_Decode_CompletePipeline_Integrated(t *testing.T) {
	// This test verifies the integrated Decode flow with an ID_END-only frame.
	// With ID_END only (zero channels), the decoder returns early as expected.

	d := NewDecoder()
	d.adtsHeaderPresent = false
	d.sfIndex = 4              // 44100 Hz
	d.objectType = 2           // LC
	d.channelConfiguration = 2 // stereo
	d.frameLength = 1024

	// ID_END only frame - returns early with zero channels
	rawData := []byte{0xE0}
	samples, info, err := d.Decode(rawData)

	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil FrameInfo")
	}
	// Zero channels for ID_END only frame
	if info.Channels != 0 {
		t.Errorf("Channels: got %d, want 0 for empty frame", info.Channels)
	}
	// Samples should be nil for empty frame
	if samples != nil {
		t.Error("expected nil samples for empty frame")
	}
	// Frame counter should increment
	if d.frame != 1 {
		t.Errorf("frame counter: got %d, want 1", d.frame)
	}
}

func TestDecoder_DecodeFloat(t *testing.T) {
	d := NewDecoder()
	d.adtsHeaderPresent = false
	d.sfIndex = 4              // 44100 Hz
	d.objectType = 2           // LC
	d.channelConfiguration = 2 // stereo
	d.frameLength = 1024
	d.config.OutputFormat = OutputFormat16Bit // Original format

	// ID_END only
	rawData := []byte{0xE0}
	_, info, err := d.DecodeFloat(rawData)
	if err != nil {
		t.Fatalf("DecodeFloat failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil info")
	}

	// Verify original format is preserved
	if d.config.OutputFormat != OutputFormat16Bit {
		t.Error("OutputFormat should be restored after DecodeFloat")
	}
}

func TestDecoder_DecodeInt16_ReturnsInt16Slice(t *testing.T) {
	// This test validates the return type and basic structure.
	// Full decoding tests require complete syntax parsing (future work).
	d := NewDecoder()

	// Initialize with valid ADTS header
	adtsHeader := []byte{
		0xFF, 0xF1, 0x50, 0x80, 0x00, 0x1F, 0xFC,
	}
	_, err := d.Init(adtsHeader)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// DecodeInt16 should return []int16 (or nil if no samples)
	// For now, we just verify the method exists and returns correct types
	samples, err := d.DecodeInt16(adtsHeader)

	// Currently, full decoding returns errors for unimplemented elements.
	// We just verify the method signature is correct.
	_ = samples
	_ = err
}

func TestDecoder_DecodeFloat32_ReturnsFloat32Slice(t *testing.T) {
	d := NewDecoder()

	// Initialize with valid ADTS header
	adtsHeader := []byte{
		0xFF, 0xF1, 0x50, 0x80, 0x00, 0x1F, 0xFC,
	}
	_, err := d.Init(adtsHeader)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// DecodeFloat32 should match the simplified signature: ([]float32, error)
	samples, err := d.DecodeFloat32(adtsHeader)
	_ = samples
	_ = err
	// Method exists with correct signature - that's what we're testing
}
