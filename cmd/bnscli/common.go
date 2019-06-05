package main

import "encoding/binary"

// sequenceID returns a sequence value encoded as implemented in the orm
// package.
func sequenceID(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}
