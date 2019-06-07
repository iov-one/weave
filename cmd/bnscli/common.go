package main

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/iov-one/weave/cmd/bnsd/app"
)

// sequenceID returns a sequence value encoded as implemented in the orm
// package.
func sequenceID(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}

// fromSequence transforms given binary representation of a sequence value into
// a decimal form. fromSequence is the opposite of the sequenceID function.
func fromSequence(b []byte) (uint64, error) {
	if len(b) != 8 {
		return 0, errors.New("sequence must be 8 bytes")
	}
	return binary.BigEndian.Uint64(b), nil
}

// writeTx serialize the transaction using a protocol buffer. First bytes
// written contain the information how much space the transaction takes.
// Size information is required to be able to stream the messages:
// https://developers.google.com/protocol-buffers/docs/techniques#streaming
func writeTx(w io.Writer, tx *app.Tx) (int, error) {
	b, err := tx.Marshal()
	if err != nil {
		return 0, err
	}

	var size [txHeaderSize]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(b)))

	if n, err := w.Write(size[:]); err != nil {
		return n, err
	}
	if n, err := w.Write(b); err != nil {
		return n + txHeaderSize, err
	}
	return txHeaderSize + len(b), nil
}

func readTx(r io.Reader) (*app.Tx, int, error) {
	// When serialized using writeTx function, first bytes contain
	// information about the actual size of the transaction message.
	var size [txHeaderSize]byte
	if n, err := r.Read(size[:txHeaderSize]); err != nil {
		return nil, n, err
	}
	msgSize := binary.BigEndian.Uint32(size[:])
	raw := make([]byte, msgSize)
	if n, err := io.ReadFull(r, raw); err != nil {
		return nil, n + txHeaderSize, err
	}

	var tx app.Tx
	if err := tx.Unmarshal(raw); err != nil {
		return nil, int(msgSize + txHeaderSize), err
	}
	return &tx, int(msgSize + txHeaderSize), nil
}

const txHeaderSize = 4
