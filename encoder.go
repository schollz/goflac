package goflac

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// Encoder represents a FLAC stream encoder
type Encoder struct {
	w              io.Writer
	sampleRate     uint32
	channels       uint8
	bitsPerSample  uint8
	totalSamples   uint64
	blockSize      uint32
	minFrameSize   uint32
	maxFrameSize   uint32
	md5sum         [16]byte
}

// NewEncoder creates a new FLAC encoder
func NewEncoder(w io.Writer, sampleRate uint32, channels, bitsPerSample uint8) (*Encoder, error) {
	if channels == 0 || channels > 8 {
		return nil, errors.New("invalid number of channels")
	}
	if bitsPerSample == 0 || bitsPerSample > 32 {
		return nil, errors.New("invalid bits per sample")
	}

	return &Encoder{
		w:             w,
		sampleRate:    sampleRate,
		channels:      channels,
		bitsPerSample: bitsPerSample,
		blockSize:     4096, // Default block size
	}, nil
}

// WriteStreamInfo writes the FLAC stream header and STREAMINFO metadata block
func (e *Encoder) WriteStreamInfo() error {
	// Write FLAC signature
	if _, err := e.w.Write([]byte("fLaC")); err != nil {
		return err
	}

	// Write STREAMINFO metadata block
	// Last metadata block flag (1) + block type (0 = STREAMINFO)
	header := byte(0x80) // 0x80 = last metadata block
	if _, err := e.w.Write([]byte{header}); err != nil {
		return err
	}

	// Block length (34 bytes for STREAMINFO)
	length := make([]byte, 3)
	binary.BigEndian.PutUint32(append([]byte{0}, length...)[:4], 34)
	if _, err := e.w.Write(length[1:]); err != nil {
		return err
	}

	// STREAMINFO block (34 bytes)
	streamInfo := make([]byte, 34)
	
	// Min block size (16 bits)
	binary.BigEndian.PutUint16(streamInfo[0:2], uint16(e.blockSize))
	
	// Max block size (16 bits)
	binary.BigEndian.PutUint16(streamInfo[2:4], uint16(e.blockSize))
	
	// Min frame size (24 bits) - 0 for unknown
	streamInfo[4] = 0
	streamInfo[5] = 0
	streamInfo[6] = 0
	
	// Max frame size (24 bits) - 0 for unknown
	streamInfo[7] = 0
	streamInfo[8] = 0
	streamInfo[9] = 0
	
	// Sample rate (20 bits) + channels (3 bits) + bits per sample (5 bits)
	// Byte 10-11-12: sample rate (20 bits)
	streamInfo[10] = byte(e.sampleRate >> 12)
	streamInfo[11] = byte(e.sampleRate >> 4)
	streamInfo[12] = byte((e.sampleRate&0x0F)<<4) | byte((e.channels-1)<<1) | byte((e.bitsPerSample-1)>>4)
	
	// Byte 13: bits per sample (4 bits) + total samples (4 bits)
	streamInfo[13] = byte(((e.bitsPerSample-1)&0x0F)<<4) | byte(e.totalSamples>>32)
	
	// Bytes 14-17: total samples (32 bits)
	binary.BigEndian.PutUint32(streamInfo[14:18], uint32(e.totalSamples))
	
	// Bytes 18-33: MD5 signature (16 bytes) - all zeros for now
	copy(streamInfo[18:34], e.md5sum[:])

	if _, err := e.w.Write(streamInfo); err != nil {
		return err
	}

	return nil
}

// EncodeFrame encodes a single FLAC frame
func (e *Encoder) EncodeFrame(samples [][]int32, frameNumber uint64) error {
	if len(samples) != int(e.channels) {
		return errors.New("sample count mismatch with channels")
	}

	blockSize := len(samples[0])
	for i := 1; i < len(samples); i++ {
		if len(samples[i]) != blockSize {
			return errors.New("all channels must have same block size")
		}
	}

	buf := newBitWriter()

	// Frame header sync code (14 bits): 0b11111111111110
	buf.writeBits(0x3FFE, 14)

	// Reserved (1 bit) + blocking strategy (1 bit)
	// 0 = fixed-blocksize stream
	buf.writeBits(0, 1)
	buf.writeBits(0, 1)

	// Block size in inter-channel samples (4 bits)
	blockSizeCode := getBlockSizeCode(uint32(blockSize))
	buf.writeBits(uint64(blockSizeCode), 4)

	// Sample rate (4 bits)
	sampleRateCode := getSampleRateCode(e.sampleRate)
	buf.writeBits(uint64(sampleRateCode), 4)

	// Channel assignment (4 bits)
	// 0b0000-0b0111 = independent channels
	buf.writeBits(uint64(e.channels-1), 4)

	// Sample size (3 bits)
	sampleSizeCode := getSampleSizeCode(e.bitsPerSample)
	buf.writeBits(uint64(sampleSizeCode), 3)

	// Reserved (1 bit)
	buf.writeBits(0, 1)

	// Frame or sample number (UTF-8 coded)
	buf.writeUTF8(frameNumber)

	// Block size if code == 0b0110 or 0b0111
	if blockSizeCode == 0x06 {
		buf.writeBits(uint64(blockSize-1), 8)
	} else if blockSizeCode == 0x07 {
		buf.writeBits(uint64(blockSize-1), 16)
	}

	// Sample rate if code needs it
	if sampleRateCode == 0x0C {
		buf.writeBits(uint64(e.sampleRate/1000), 8)
	} else if sampleRateCode == 0x0D {
		buf.writeBits(uint64(e.sampleRate), 16)
	} else if sampleRateCode == 0x0E {
		buf.writeBits(uint64(e.sampleRate/10), 16)
	}

	// Header CRC-8
	headerBytes := buf.bytes()
	crc8 := calculateCRC8(headerBytes)
	buf.writeBits(uint64(crc8), 8)

	// Encode subframes for each channel
	for ch := 0; ch < int(e.channels); ch++ {
		if err := e.encodeSubframe(buf, samples[ch]); err != nil {
			return err
		}
	}

	// Byte align
	buf.alignToByte()

	// Frame CRC-16
	frameBytes := buf.bytes()
	crc16 := calculateCRC16(frameBytes)
	buf.writeBits(uint64(crc16), 16)

	// Write to output
	if _, err := e.w.Write(buf.bytes()); err != nil {
		return err
	}

	return nil
}

// encodeSubframe encodes a single subframe using fixed prediction
func (e *Encoder) encodeSubframe(buf *bitWriter, samples []int32) error {
	// For simplicity, use fixed predictor order 2
	order := 2

	// Subframe header: 0 (padding) + subframe type (6 bits) + wasted bits flag (1 bit)
	buf.writeBits(0, 1)
	// Subframe type: 0b001xxx for FIXED predictor (xxx = order)
	buf.writeBits(0x08|uint64(order), 6)
	buf.writeBits(0, 1) // No wasted bits

	// Write unencoded warm-up samples
	for i := 0; i < order; i++ {
		buf.writeBitsSigned(int64(samples[i]), int(e.bitsPerSample))
	}

	// Calculate residuals
	residuals := make([]int32, len(samples)-order)
	for i := order; i < len(samples); i++ {
		predicted := fixedPredict(samples, i, order)
		residuals[i-order] = samples[i] - predicted
	}

	// Encode residuals using Rice coding
	return e.encodeResidual(buf, residuals)
}

// fixedPredict performs fixed linear prediction
func fixedPredict(samples []int32, pos, order int) int32 {
	switch order {
	case 0:
		return 0
	case 1:
		return samples[pos-1]
	case 2:
		return 2*samples[pos-1] - samples[pos-2]
	case 3:
		return 3*samples[pos-1] - 3*samples[pos-2] + samples[pos-3]
	case 4:
		return 4*samples[pos-1] - 6*samples[pos-2] + 4*samples[pos-3] - samples[pos-4]
	default:
		return 0
	}
}

// encodeResidual encodes residuals using Rice coding
func (e *Encoder) encodeResidual(buf *bitWriter, residuals []int32) error {
	// Residual coding method: 0b00 = partitioned Rice coding
	buf.writeBits(0, 2)

	// Partition order (4 bits) - 0 = single partition
	partitionOrder := uint64(0)
	buf.writeBits(partitionOrder, 4)

	// Find optimal Rice parameter
	riceParam := findOptimalRiceParameter(residuals)

	// Rice parameter (4 or 5 bits depending on coding method)
	buf.writeBits(uint64(riceParam), 4)

	// Encode residuals
	for _, r := range residuals {
		encodeRice(buf, r, riceParam)
	}

	return nil
}

// findOptimalRiceParameter finds the optimal Rice parameter
func findOptimalRiceParameter(residuals []int32) uint8 {
	if len(residuals) == 0 {
		return 0
	}

	// Calculate mean absolute value
	var sum uint64
	for _, r := range residuals {
		if r < 0 {
			sum += uint64(-r)
		} else {
			sum += uint64(r)
		}
	}
	mean := float64(sum) / float64(len(residuals))

	// Estimate optimal parameter
	if mean < 1 {
		return 0
	}
	param := uint8(math.Log2(mean))
	if param > 14 {
		param = 14
	}
	return param
}

// encodeRice encodes a signed integer using Rice coding
func encodeRice(buf *bitWriter, value int32, param uint8) {
	// Convert signed to unsigned (zigzag encoding)
	var uval uint32
	if value < 0 {
		uval = uint32(-2*value - 1)
	} else {
		uval = uint32(2 * value)
	}

	// Split into quotient and remainder
	quotient := uval >> param
	remainder := uval & ((1 << param) - 1)

	// Write quotient in unary
	for i := uint32(0); i < quotient; i++ {
		buf.writeBits(0, 1)
	}
	buf.writeBits(1, 1)

	// Write remainder in binary
	buf.writeBits(uint64(remainder), int(param))
}

// getBlockSizeCode returns the FLAC block size code
func getBlockSizeCode(blockSize uint32) uint8 {
	switch blockSize {
	case 192:
		return 0x01
	case 576:
		return 0x02
	case 1152:
		return 0x03
	case 2304:
		return 0x04
	case 4608:
		return 0x05
	case 256:
		return 0x08
	case 512:
		return 0x09
	case 1024:
		return 0x0A
	case 2048:
		return 0x0B
	case 4096:
		return 0x0C
	case 8192:
		return 0x0D
	case 16384:
		return 0x0E
	case 32768:
		return 0x0F
	default:
		if blockSize <= 256 {
			return 0x06 // 8-bit value follows
		}
		return 0x07 // 16-bit value follows
	}
}

// getSampleRateCode returns the FLAC sample rate code
func getSampleRateCode(sampleRate uint32) uint8 {
	switch sampleRate {
	case 88200:
		return 0x01
	case 176400:
		return 0x02
	case 192000:
		return 0x03
	case 8000:
		return 0x04
	case 16000:
		return 0x05
	case 22050:
		return 0x06
	case 24000:
		return 0x07
	case 32000:
		return 0x08
	case 44100:
		return 0x09
	case 48000:
		return 0x0A
	case 96000:
		return 0x0B
	default:
		if sampleRate%1000 == 0 && sampleRate/1000 < 256 {
			return 0x0C // kHz follows
		}
		if sampleRate < 65536 {
			return 0x0D // Hz follows
		}
		return 0x0E // tens of Hz follows
	}
}

// getSampleSizeCode returns the FLAC sample size code
func getSampleSizeCode(bitsPerSample uint8) uint8 {
	switch bitsPerSample {
	case 8:
		return 0x01
	case 12:
		return 0x02
	case 16:
		return 0x04
	case 20:
		return 0x05
	case 24:
		return 0x06
	case 32:
		return 0x07
	default:
		return 0x00
	}
}

// calculateCRC8 calculates CRC-8 for FLAC frame header
func calculateCRC8(data []byte) uint8 {
	crc := uint8(0)
	for _, b := range data {
		crc ^= b
		for i := 0; i < 8; i++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ 0x07
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// calculateCRC16 calculates CRC-16 for FLAC frame
func calculateCRC16(data []byte) uint16 {
	crc := uint16(0)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x8005
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// Encode encodes PCM audio data to FLAC
func (e *Encoder) Encode(samples [][]int32) error {
	if err := e.WriteStreamInfo(); err != nil {
		return err
	}

	blockSize := int(e.blockSize)
	totalBlocks := (len(samples[0]) + blockSize - 1) / blockSize

	for blockNum := 0; blockNum < totalBlocks; blockNum++ {
		start := blockNum * blockSize
		end := start + blockSize
		if end > len(samples[0]) {
			end = len(samples[0])
		}

		// Extract block samples for all channels
		blockSamples := make([][]int32, e.channels)
		for ch := 0; ch < int(e.channels); ch++ {
			blockSamples[ch] = samples[ch][start:end]
		}

		if err := e.EncodeFrame(blockSamples, uint64(blockNum)); err != nil {
			return err
		}
	}

	return nil
}
