package goflac

import (
	"bytes"
	"os"
	"testing"
)

func TestEncodeSineWave(t *testing.T) {
	// Generate a sine wave WAV file
	var wavBuf bytes.Buffer
	frequency := 440.0 // A4 note
	duration := 1.0    // 1 second
	sampleRate := uint32(44100)
	channels := uint16(2)
	bitsPerSample := uint16(16)

	err := GenerateSineWAV(&wavBuf, frequency, duration, sampleRate, channels, bitsPerSample)
	if err != nil {
		t.Fatalf("Failed to generate sine wave: %v", err)
	}

	// Save WAV for debugging
	if err := os.WriteFile("/tmp/test_sine.wav", wavBuf.Bytes(), 0644); err != nil {
		t.Logf("Warning: Could not save test WAV file: %v", err)
	}

	// Read the WAV file
	wavReader, err := NewWAVReader(bytes.NewReader(wavBuf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to read WAV: %v", err)
	}

	// Read samples
	samples, err := wavReader.ReadSamples()
	if err != nil {
		t.Fatalf("Failed to read samples: %v", err)
	}

	if len(samples) != int(channels) {
		t.Fatalf("Expected %d channels, got %d", channels, len(samples))
	}

	// Encode to FLAC
	var flacBuf bytes.Buffer
	encoder, err := NewEncoder(&flacBuf, sampleRate, uint8(channels), uint8(bitsPerSample))
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}

	err = encoder.Encode(samples)
	if err != nil {
		t.Fatalf("Failed to encode FLAC: %v", err)
	}

	// Save FLAC file
	flacData := flacBuf.Bytes()
	if err := os.WriteFile("/tmp/test_sine.flac", flacData, 0644); err != nil {
		t.Logf("Warning: Could not save test FLAC file: %v", err)
	}

	// Verify FLAC header
	if len(flacData) < 4 {
		t.Fatal("FLAC output too short")
	}

	if string(flacData[0:4]) != "fLaC" {
		t.Fatalf("Invalid FLAC header: %s", string(flacData[0:4]))
	}

	t.Logf("Successfully encoded %d samples (%d channels) to %d bytes FLAC",
		len(samples[0]), len(samples), len(flacData))
	t.Logf("Compression ratio: %.2f%%", float64(len(flacData))*100/float64(wavBuf.Len()))
}

func TestEncoder_InvalidParameters(t *testing.T) {
	var buf bytes.Buffer

	// Test invalid channels
	_, err := NewEncoder(&buf, 44100, 0, 16)
	if err == nil {
		t.Error("Expected error for 0 channels")
	}

	_, err = NewEncoder(&buf, 44100, 9, 16)
	if err == nil {
		t.Error("Expected error for 9 channels")
	}

	// Test invalid bits per sample
	_, err = NewEncoder(&buf, 44100, 2, 0)
	if err == nil {
		t.Error("Expected error for 0 bits per sample")
	}

	_, err = NewEncoder(&buf, 44100, 2, 33)
	if err == nil {
		t.Error("Expected error for 33 bits per sample")
	}
}

func TestWAVReader(t *testing.T) {
	// Generate a simple WAV
	var wavBuf bytes.Buffer
	err := GenerateSineWAV(&wavBuf, 440.0, 0.1, 44100, 1, 16)
	if err != nil {
		t.Fatalf("Failed to generate WAV: %v", err)
	}

	// Read it back
	wavReader, err := NewWAVReader(bytes.NewReader(wavBuf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to read WAV: %v", err)
	}

	if wavReader.Channels() != 1 {
		t.Errorf("Expected 1 channel, got %d", wavReader.Channels())
	}

	if wavReader.SampleRate() != 44100 {
		t.Errorf("Expected 44100 sample rate, got %d", wavReader.SampleRate())
	}

	if wavReader.BitsPerSample() != 16 {
		t.Errorf("Expected 16 bits per sample, got %d", wavReader.BitsPerSample())
	}

	samples, err := wavReader.ReadSamples()
	if err != nil {
		t.Fatalf("Failed to read samples: %v", err)
	}

	expectedSamples := 4410 // 0.1 seconds * 44100 Hz
	if len(samples[0]) != expectedSamples {
		t.Errorf("Expected %d samples, got %d", expectedSamples, len(samples[0]))
	}
}

func TestBitWriter(t *testing.T) {
	bw := newBitWriter()

	// Test writing exact bytes
	bw.writeBits(0xFF, 8)
	bw.writeBits(0x00, 8)
	bw.writeBits(0xAA, 8)

	result := bw.bytes()
	expected := []byte{0xFF, 0x00, 0xAA}

	if len(result) != len(expected) {
		t.Fatalf("Expected %d bytes, got %d", len(expected), len(result))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("Byte %d: expected 0x%02X, got 0x%02X", i, expected[i], result[i])
		}
	}
}

func TestBitWriter_PartialBits(t *testing.T) {
	bw := newBitWriter()

	// Write 4 bits twice to make a byte
	bw.writeBits(0x0F, 4) // 1111
	bw.writeBits(0x00, 4) // 0000

	result := bw.bytes()
	if len(result) != 1 {
		t.Fatalf("Expected 1 byte, got %d", len(result))
	}

	if result[0] != 0xF0 {
		t.Errorf("Expected 0xF0, got 0x%02X", result[0])
	}
}

func TestBitWriter_AlignToByte(t *testing.T) {
	bw := newBitWriter()

	bw.writeBits(0x07, 3) // 111
	bw.alignToByte()      // Should pad with 00000

	result := bw.bytes()
	if len(result) != 1 {
		t.Fatalf("Expected 1 byte, got %d", len(result))
	}

	if result[0] != 0xE0 {
		t.Errorf("Expected 0xE0, got 0x%02X", result[0])
	}
}
