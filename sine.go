package goflac

import (
	"encoding/binary"
	"io"
	"math"
)

// GenerateSineWAV generates a WAV file with a sine wave
func GenerateSineWAV(w io.Writer, frequency float64, duration float64, sampleRate uint32, channels uint16, bitsPerSample uint16) error {
	// Calculate parameters
	numSamples := uint32(duration * float64(sampleRate))
	byteRate := sampleRate * uint32(channels) * uint32(bitsPerSample/8)
	blockAlign := channels * (bitsPerSample / 8)
	dataSize := numSamples * uint32(channels) * uint32(bitsPerSample/8)

	// Write RIFF header
	if _, err := w.Write([]byte("RIFF")); err != nil {
		return err
	}
	fileSize := 36 + dataSize
	if err := binary.Write(w, binary.LittleEndian, uint32(fileSize)); err != nil {
		return err
	}
	if _, err := w.Write([]byte("WAVE")); err != nil {
		return err
	}

	// Write fmt chunk
	if _, err := w.Write([]byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(16)); err != nil { // fmt chunk size
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(1)); err != nil { // PCM format
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, channels); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, sampleRate); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, bitsPerSample); err != nil {
		return err
	}

	// Write data chunk header
	if _, err := w.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, dataSize); err != nil {
		return err
	}

	// Generate and write sine wave samples
	amplitude := float64(int32(1<<(bitsPerSample-1)) - 1)
	for i := uint32(0); i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		value := amplitude * math.Sin(2*math.Pi*frequency*t)

		for ch := uint16(0); ch < channels; ch++ {
			switch bitsPerSample {
			case 8:
				sample := uint8(int32(value) + 128)
				if err := binary.Write(w, binary.LittleEndian, sample); err != nil {
					return err
				}
			case 16:
				sample := int16(value)
				if err := binary.Write(w, binary.LittleEndian, sample); err != nil {
					return err
				}
			case 24:
				sample := int32(value)
				bytes := []byte{
					byte(sample),
					byte(sample >> 8),
					byte(sample >> 16),
				}
				if _, err := w.Write(bytes); err != nil {
					return err
				}
			case 32:
				sample := int32(value)
				if err := binary.Write(w, binary.LittleEndian, sample); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
