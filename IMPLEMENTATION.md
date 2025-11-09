# FLAC Encoder Implementation Notes

## Architecture

This implementation follows the libflac architecture as specified in the FLAC format documentation.

### Key Components

1. **Stream Structure**
   - Magic signature: "fLaC" (4 bytes)
   - Metadata blocks (at least STREAMINFO)
   - Audio frames containing compressed data

2. **STREAMINFO Metadata Block** (34 bytes)
   - Min/max block size
   - Min/max frame size
   - Sample rate, channels, bit depth
   - Total samples
   - MD5 signature (currently zeros)

3. **Frame Structure**
   - Frame header with sync code (0x3FFE)
   - Subframes (one per channel)
   - Padding to byte boundary
   - CRC-16 checksum

4. **Subframe Encoding**
   - Fixed linear prediction (order 0-4)
   - Warm-up samples (unencoded)
   - Residual coding using Rice/Golomb

### Prediction

Fixed linear predictors are used based on previous samples:
- Order 0: Constant (0)
- Order 1: s[i-1]
- Order 2: 2*s[i-1] - s[i-2]
- Order 3: 3*s[i-1] - 3*s[i-2] + s[i-3]
- Order 4: 4*s[i-1] - 6*s[i-2] + 4*s[i-3] - s[i-4]

Currently using order 2 for all encoding.

### Rice Coding

Residuals are encoded using Rice/Golomb coding:
1. Convert signed to unsigned (zigzag encoding)
2. Split into quotient and remainder
3. Encode quotient in unary (0s followed by 1)
4. Encode remainder in binary (k bits)

The Rice parameter k is chosen based on mean absolute residual value.

### CRC Protection

- **CRC-8**: Frame header protection (polynomial 0x07)
- **CRC-16**: Full frame protection (polynomial 0x8005)

## Performance

Typical compression ratios:
- Sine waves: 55-60% of WAV size
- Speech: 40-60% of WAV size
- Music: 50-70% of WAV size

## Limitations

Current implementation:
- Fixed predictor only (no LPC)
- Single partition for residual coding
- No mid-side or left-side stereo coding
- Block size fixed at 4096 samples
- No MD5 signature calculation
- No seeking support

## Future Improvements

1. **Better Compression**
   - Linear Predictive Coding (LPC)
   - Adaptive predictor order selection
   - Stereo decorrelation (mid-side encoding)
   - Partition order optimization

2. **Features**
   - Variable block size
   - MD5 signature calculation
   - SEEKTABLE metadata
   - Additional metadata blocks (tags, cue sheets)

3. **Performance**
   - Parallel encoding of frames
   - SIMD optimizations
   - Memory pooling

## References

- FLAC Format: https://xiph.org/flac/format.html
- FLAC API: https://xiph.org/flac/api/
- libflac: https://github.com/xiph/flac

## Testing

Run tests with:
```bash
go test -v
```

Run example:
```bash
cd examples/encode_sine
go run main.go
```

Verify output with external tools (if available):
```bash
flac --test output.flac
metaflac --list output.flac
```
