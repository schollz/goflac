package goflac

import "bytes"

// bitWriter handles writing bits to a byte buffer
type bitWriter struct {
	buf      bytes.Buffer
	current  uint64
	bitCount int
}

// newBitWriter creates a new bit writer
func newBitWriter() *bitWriter {
	return &bitWriter{}
}

// writeBits writes n bits from value to the buffer
func (bw *bitWriter) writeBits(value uint64, n int) {
	if n == 0 {
		return
	}

	// Add new bits to current
	bw.current = (bw.current << n) | (value & ((1 << n) - 1))
	bw.bitCount += n

	// Write complete bytes
	for bw.bitCount >= 8 {
		bw.bitCount -= 8
		b := byte(bw.current >> bw.bitCount)
		bw.buf.WriteByte(b)
		bw.current &= (1 << bw.bitCount) - 1
	}
}

// writeBitsSigned writes n bits from a signed value
func (bw *bitWriter) writeBitsSigned(value int64, n int) {
	// Convert signed to unsigned representation
	var uval uint64
	if value < 0 {
		uval = uint64((1 << n) + value)
	} else {
		uval = uint64(value)
	}
	bw.writeBits(uval, n)
}

// writeUTF8 writes a number in UTF-8 style encoding
func (bw *bitWriter) writeUTF8(value uint64) {
	if value < 0x80 {
		bw.writeBits(value, 8)
	} else if value < 0x800 {
		bw.writeBits(0xC0|(value>>6), 8)
		bw.writeBits(0x80|(value&0x3F), 8)
	} else if value < 0x10000 {
		bw.writeBits(0xE0|(value>>12), 8)
		bw.writeBits(0x80|((value>>6)&0x3F), 8)
		bw.writeBits(0x80|(value&0x3F), 8)
	} else if value < 0x200000 {
		bw.writeBits(0xF0|(value>>18), 8)
		bw.writeBits(0x80|((value>>12)&0x3F), 8)
		bw.writeBits(0x80|((value>>6)&0x3F), 8)
		bw.writeBits(0x80|(value&0x3F), 8)
	} else if value < 0x4000000 {
		bw.writeBits(0xF8|(value>>24), 8)
		bw.writeBits(0x80|((value>>18)&0x3F), 8)
		bw.writeBits(0x80|((value>>12)&0x3F), 8)
		bw.writeBits(0x80|((value>>6)&0x3F), 8)
		bw.writeBits(0x80|(value&0x3F), 8)
	} else if value < 0x80000000 {
		bw.writeBits(0xFC|(value>>30), 8)
		bw.writeBits(0x80|((value>>24)&0x3F), 8)
		bw.writeBits(0x80|((value>>18)&0x3F), 8)
		bw.writeBits(0x80|((value>>12)&0x3F), 8)
		bw.writeBits(0x80|((value>>6)&0x3F), 8)
		bw.writeBits(0x80|(value&0x3F), 8)
	} else {
		bw.writeBits(0xFE|(value>>36), 8)
		bw.writeBits(0x80|((value>>30)&0x3F), 8)
		bw.writeBits(0x80|((value>>24)&0x3F), 8)
		bw.writeBits(0x80|((value>>18)&0x3F), 8)
		bw.writeBits(0x80|((value>>12)&0x3F), 8)
		bw.writeBits(0x80|((value>>6)&0x3F), 8)
		bw.writeBits(0x80|(value&0x3F), 8)
	}
}

// alignToByte pads with zeros to align to byte boundary
func (bw *bitWriter) alignToByte() {
	if bw.bitCount > 0 {
		bw.writeBits(0, 8-bw.bitCount)
	}
}

// bytes returns the written bytes
func (bw *bitWriter) bytes() []byte {
	return bw.buf.Bytes()
}
