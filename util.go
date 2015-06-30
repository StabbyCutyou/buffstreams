package buffstreams

import (
	"encoding/binary"
	"math"
)

func UInt16ToByteArray(value uint16, bufferSize int) []byte {
	toWriteLen := make([]byte, bufferSize)
	binary.LittleEndian.PutUint16(toWriteLen, value)
	return toWriteLen
}

// Formula for taking size in bytes and calculating # of bits to express that size
// http://www.exploringbinary.com/number-of-bits-in-a-decimal-integer/
func MessageSizeToBitLength(messageSize int) int {
	bytes := float64(messageSize)
	header := math.Ceil(math.Floor(math.Log2(bytes)+1) / 8.0)
	return int(header)
}
