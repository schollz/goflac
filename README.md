# goflac

A pure Go FLAC (Free Lossless Audio Codec) encoder with no C dependencies. This implementation follows the libflac architecture and provides a simple API for encoding PCM audio to FLAC format.

## Features

- **Pure Go**: No CGO or libc dependencies
- **FLAC Encoding**: Full FLAC stream encoder implementation
- **Fixed Prediction**: Uses fixed linear predictors for compression
- **Rice Coding**: Implements Rice/Golomb coding for residual encoding
- **WAV Support**: Built-in WAV file reader for easy testing
- **Sine Wave Generator**: Includes utility for generating test audio

## Installation

```bash
go get github.com/schollz/goflac
```

## Usage

### Encoding Audio to FLAC

```go
package main

import (
    "bytes"
    "os"
    "github.com/schollz/goflac"
)

func main() {
    // Prepare your PCM samples (2D array: [channels][samples])
    samples := [][]int32{
        {0, 100, 200, 300, ...}, // Channel 0
        {0, 100, 200, 300, ...}, // Channel 1
    }
    
    // Create encoder
    var output bytes.Buffer
    encoder, err := goflac.NewEncoder(&output, 44100, 2, 16)
    if err != nil {
        panic(err)
    }
    
    // Encode samples
    err = encoder.Encode(samples)
    if err != nil {
        panic(err)
    }
    
    // Save to file
    os.WriteFile("output.flac", output.Bytes(), 0644)
}
```

### Converting WAV to FLAC

```go
package main

import (
    "bytes"
    "os"
    "github.com/schollz/goflac"
)

func main() {
    // Read WAV file
    wavData, _ := os.ReadFile("input.wav")
    wavReader, _ := goflac.NewWAVReader(bytes.NewReader(wavData))
    
    // Read samples
    samples, _ := wavReader.ReadSamples()
    
    // Create encoder with WAV parameters
    var output bytes.Buffer
    encoder, _ := goflac.NewEncoder(
        &output,
        wavReader.SampleRate(),
        uint8(wavReader.Channels()),
        uint8(wavReader.BitsPerSample()),
    )
    
    // Encode
    encoder.Encode(samples)
    
    // Save FLAC
    os.WriteFile("output.flac", output.Bytes(), 0644)
}
```

## Examples

See the `examples/encode_sine` directory for a complete example that generates a sine wave and encodes it to FLAC:

```bash
cd examples/encode_sine
go run main.go
```

This will generate `sine.wav` and `sine.flac` files demonstrating the encoding process.

## Implementation Details

This implementation follows the FLAC specification and libflac architecture:

### FLAC Frame Structure
- **Stream Header**: "fLaC" signature followed by metadata blocks
- **STREAMINFO Block**: Contains sample rate, channels, bit depth, and total samples
- **Audio Frames**: Contains encoded audio data with frame header and subframes

### Encoding Process
1. **Framing**: Audio is divided into blocks (default 4096 samples)
2. **Prediction**: Fixed linear prediction is applied (order 2)
3. **Residual Encoding**: Prediction residuals are encoded using Rice coding
4. **CRC Protection**: Frame headers (CRC-8) and frames (CRC-16) are checksummed

### Compression
Typical compression ratios:
- Sine waves: ~55-60% of original WAV size
- Music: ~50-70% of original WAV size (varies by content)

## Testing

Run the test suite:

```bash
go test -v
```

The tests include:
- Sine wave generation and encoding
- WAV file reading
- Bit writer operations
- Parameter validation

## Technical Specifications

Supported formats:
- **Sample Rates**: Any valid rate (common: 44100, 48000, 96000 Hz)
- **Channels**: 1-8 channels
- **Bit Depth**: 8, 12, 16, 20, 24, 32 bits per sample
- **Block Size**: 4096 samples (configurable)

## License

[Add your license here]

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## References

- [FLAC Format Specification](https://xiph.org/flac/format.html)
- [libflac](https://github.com/xiph/flac) - Reference implementation in C