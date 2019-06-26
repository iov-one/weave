package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
)

// unpackSequence process given raw string representation and does its best in
// order to decode a sequence value from the raw form. This function is
// intended to be used with data coming from stdin.
//
// Unless a format prefix is provided, value is expected to be a decimal
// number.
//
// Supported prefixes and their formats are:
// - (none): string encoded decimal number
// - hex: hex encoded binary sequence value
// - base64: base64 encoded binary  sequence value
func unpackSequence(raw string) ([]byte, error) {
	if raw == "" {
		return nil, errors.New("empty")
	}

	// By default the decimal format is used
	format := "decimal"
	chunks := strings.SplitN(raw, ":", 2)
	if len(chunks) == 2 {
		format = chunks[0]
		raw = chunks[1]
	}

	switch format {
	case "decimal":
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid decimal format: %s", err)
		}
		if n < 1 {
			return nil, errors.New("sequence value must be greater than zero")
		}
		return sequenceID(n), nil

	case "hex":
		b, err := hex.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid hex format: %s", err)
		}
		if len(b) != sequenceBinarySize {
			return nil, fmt.Errorf("sequence value must be %d bytes long", sequenceBinarySize)
		}
		return b, nil
	case "base64":
		b, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 format: %s", err)
		}
		if len(b) != sequenceBinarySize {
			return nil, fmt.Errorf("sequence value must be %d bytes long", sequenceBinarySize)
		}
		return b, nil
	default:
		return nil, fmt.Errorf("unknown %q sequence format", format)
	}

}

// sequenceID returns a sequence value encoded as implemented in the orm
// package.
func sequenceID(n uint64) []byte {
	b := make([]byte, sequenceBinarySize)
	binary.BigEndian.PutUint64(b, n)
	return b
}

// fromSequence transforms given binary representation of a sequence value into
// a decimal form. fromSequence is the opposite of the sequenceID function.
func fromSequence(b []byte) (uint64, error) {
	if len(b) != sequenceBinarySize {
		return 0, fmt.Errorf("sequence must be %d bytes", sequenceBinarySize)
	}
	return binary.BigEndian.Uint64(b), nil
}

// sequenceBinarySize is the size of a binary representation of a sequence value.
const sequenceBinarySize = 8

// writeTx serialize the transaction using a protocol buffer. First bytes
// written contain the information how much space the transaction takes.
// Size information is required to be able to stream the messages:
// https://developers.google.com/protocol-buffers/docs/techniques#streaming
func writeTx(w io.Writer, tx *bnsd.Tx) (int, error) {
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

// readTx consumes data from given reader and unpack the serialized
// transaction. This function should be used together with writeTx as
// serialized transaction is a protobuf with a custom header added.
//
// This function can be used to read from os.Stdin when nothing is being
// written to the stdin. In such case, io.EOF is returned.
func readTx(r io.Reader) (*bnsd.Tx, int, error) {
	// If the given reader is providing a stat information (ie os.Stdin)
	// then check if the data is being piped. That should prevent us from
	// waiting for a data on a reader that no one ever writes to.
	if s, ok := r.(stater); ok {
		if info, err := s.Stat(); err == nil {
			isPipe := (info.Mode() & os.ModeCharDevice) == 0
			if !isPipe {
				return nil, 0, io.EOF
			}
		}
	}

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

	var tx bnsd.Tx
	if err := tx.Unmarshal(raw); err != nil {
		return nil, int(msgSize + txHeaderSize), err
	}
	return &tx, int(msgSize + txHeaderSize), nil
}

const txHeaderSize = 4

type stater interface {
	Stat() (os.FileInfo, error)
}

// readAddressWeightPairCSV reads all address weight pairs from a csv file and calls the callback.
func readAddressWeightPairCSV(csvpath string, f func(address weave.Address, weight uint32)) error {
	fd, err := os.Open(csvpath)
	if err != nil {
		return fmt.Errorf("cannot open file: %s", err)
	}
	defer fd.Close()

	rd := csv.NewReader(fd)
	for lineNo := 1; ; lineNo++ {
		row, err := rd.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if len(row) != 2 {
			return fmt.Errorf("invalid line %d: expected 2 columns, got %d", lineNo, len(row))
		}
		address, err := weave.ParseAddress(row[0])
		if err != nil {
			return fmt.Errorf("invalid line %d: invalid address %q: %s", lineNo, row[0], err)
		}
		weight, err := strconv.ParseUint(row[1], 10, 32)
		if err != nil {
			return fmt.Errorf("invalid line %d: invalid weight (q-factor) %q: %s", lineNo, row[1], err)
		}
		f(address, uint32(weight))
	}
}
