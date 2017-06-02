package buffstreams

import (
	"encoding/binary"
	"math"
)

func byteArrayToUInt32(bytes []byte) (result int64, bytesRead int) {
	return binary.Varint(bytes)
}

func intToByteArray(value int64, bufferSize int) []byte {
	toWriteLen := make([]byte, bufferSize)
	binary.PutVarint(toWriteLen, value)
	return toWriteLen
}

func messageSizeToBitLength(messageSize int) int {
	bytes := float64(messageSize)
	header := math.Ceil(math.Floor(math.Log2(bytes)+1)/8.0) + 1
	return int(header)
}
