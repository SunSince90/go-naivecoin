package main

import "encoding/binary"

// FromInt64ToBytes casts an int64 number to a slice of bytes ([]byte).
//
// TODO: try to do this with go 1.17 generics, so we can also do int32 as well.
func FromInt64ToBytes(val int64) []byte {
	bytesVal := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytesVal, uint64(val))
	return bytesVal
}
