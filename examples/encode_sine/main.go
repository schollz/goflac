package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/schollz/goflac"
)

func main() {
	// Generate a sine wave WAV file
	var wavBuf bytes.Buffer
	frequency := 440.0 // A4 note (440 Hz)
	duration := 2.0    // 2 seconds
	sampleRate := uint32(44100)
	channels := uint16(2)
	bitsPerSample := uint16(16)

	fmt.Printf("Generating sine wave: %.0f Hz, %.1f seconds, %d Hz sample rate\n",
		frequency, duration, sampleRate)

	err := goflac.GenerateSineWAV(&wavBuf, frequency, duration, sampleRate, channels, bitsPerSample)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating sine wave: %v\n", err)
		os.Exit(1)
	}

	// Save WAV file
	wavFile := "sine.wav"
	if err := os.WriteFile(wavFile, wavBuf.Bytes(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing WAV file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated WAV file: %s (%d bytes)\n", wavFile, wavBuf.Len())

	// Read the WAV file
	wavReader, err := goflac.NewWAVReader(bytes.NewReader(wavBuf.Bytes()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading WAV: %v\n", err)
		os.Exit(1)
	}

	// Read samples
	samples, err := wavReader.ReadSamples()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading samples: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Read %d samples from %d channels\n", len(samples[0]), len(samples))

	// Encode to FLAC
	var flacBuf bytes.Buffer
	encoder, err := goflac.NewEncoder(&flacBuf, sampleRate, uint8(channels), uint8(bitsPerSample))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating encoder: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Encoding to FLAC...")
	err = encoder.Encode(samples)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding FLAC: %v\n", err)
		os.Exit(1)
	}

	// Save FLAC file
	flacFile := "sine.flac"
	flacData := flacBuf.Bytes()
	if err := os.WriteFile(flacFile, flacData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing FLAC file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated FLAC file: %s (%d bytes)\n", flacFile, len(flacData))
	compressionRatio := float64(len(flacData)) * 100 / float64(wavBuf.Len())
	fmt.Printf("Compression ratio: %.2f%%\n", compressionRatio)
	fmt.Println("\nSuccess! Pure Go FLAC encoding complete.")
}
