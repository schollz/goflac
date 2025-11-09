package goflac

import (
	"encoding/binary"
	"errors"
	"io"
)

// WAVReader reads WAV file format
type WAVReader struct {
	r             io.Reader
	channels      uint16
	sampleRate    uint32
	bitsPerSample uint16
	dataSize      uint32
}

// NewWAVReader creates a new WAV reader
func NewWAVReader(r io.Reader) (*WAVReader, error) {
	w := &WAVReader{r: r}
	if err := w.readHeader(); err != nil {
		return nil, err
	}
	return w, nil
}

// readHeader reads and parses the WAV header
func (w *WAVReader) readHeader() error {
	// Read RIFF header
	riffHeader := make([]byte, 12)
	if _, err := io.ReadFull(w.r, riffHeader); err != nil {
		return err
	}

	if string(riffHeader[0:4]) != "RIFF" {
		return errors.New("not a valid WAV file: missing RIFF header")
	}

	if string(riffHeader[8:12]) != "WAVE" {
		return errors.New("not a valid WAV file: missing WAVE header")
	}

	// Read chunks until we find fmt and data
	for {
		chunkHeader := make([]byte, 8)
		if _, err := io.ReadFull(w.r, chunkHeader); err != nil {
			if err == io.EOF {
				return errors.New("unexpected end of file")
			}
			return err
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		if chunkID == "fmt " {
			if err := w.readFmtChunk(chunkSize); err != nil {
				return err
			}
		} else if chunkID == "data" {
			w.dataSize = chunkSize
			return nil
		} else {
			// Skip unknown chunk
			skip := make([]byte, chunkSize)
			if _, err := io.ReadFull(w.r, skip); err != nil {
				return err
			}
		}
	}
}

// readFmtChunk reads the format chunk
func (w *WAVReader) readFmtChunk(size uint32) error {
	if size < 16 {
		return errors.New("invalid fmt chunk size")
	}

	fmtData := make([]byte, size)
	if _, err := io.ReadFull(w.r, fmtData); err != nil {
		return err
	}

	audioFormat := binary.LittleEndian.Uint16(fmtData[0:2])
	if audioFormat != 1 {
		return errors.New("only PCM format is supported")
	}

	w.channels = binary.LittleEndian.Uint16(fmtData[2:4])
	w.sampleRate = binary.LittleEndian.Uint32(fmtData[4:8])
	w.bitsPerSample = binary.LittleEndian.Uint16(fmtData[14:16])

	return nil
}

// ReadSamples reads all PCM samples from the WAV file
func (w *WAVReader) ReadSamples() ([][]int32, error) {
	bytesPerSample := int(w.bitsPerSample / 8)
	numSamples := int(w.dataSize) / (bytesPerSample * int(w.channels))

	samples := make([][]int32, w.channels)
	for i := range samples {
		samples[i] = make([]int32, numSamples)
	}

	for i := 0; i < numSamples; i++ {
		for ch := 0; ch < int(w.channels); ch++ {
			sample, err := w.readSample()
			if err != nil {
				return nil, err
			}
			samples[ch][i] = sample
		}
	}

	return samples, nil
}

// readSample reads a single sample
func (w *WAVReader) readSample() (int32, error) {
	bytesPerSample := int(w.bitsPerSample / 8)
	buf := make([]byte, bytesPerSample)

	if _, err := io.ReadFull(w.r, buf); err != nil {
		return 0, err
	}

	var sample int32
	switch w.bitsPerSample {
	case 8:
		// 8-bit samples are unsigned
		sample = int32(buf[0]) - 128
	case 16:
		sample = int32(int16(binary.LittleEndian.Uint16(buf)))
	case 24:
		// 24-bit is stored as 3 bytes, little-endian
		val := int32(buf[0]) | int32(buf[1])<<8 | int32(buf[2])<<16
		// Sign extend
		if val&0x800000 != 0 {
			val |= ^0xFFFFFF
		}
		sample = val
	case 32:
		sample = int32(binary.LittleEndian.Uint32(buf))
	default:
		return 0, errors.New("unsupported bits per sample")
	}

	return sample, nil
}

// Channels returns the number of channels
func (w *WAVReader) Channels() uint16 {
	return w.channels
}

// SampleRate returns the sample rate
func (w *WAVReader) SampleRate() uint32 {
	return w.sampleRate
}

// BitsPerSample returns the bits per sample
func (w *WAVReader) BitsPerSample() uint16 {
	return w.bitsPerSample
}
