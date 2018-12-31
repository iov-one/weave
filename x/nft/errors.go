package nft

import (
	"encoding/hex"
	stderrors "errors"

	"github.com/iov-one/weave/errors"
)

// nft and subpackages reserves 500~600
const (
	CodeUnsupportedTokenType uint32 = 500
	CodeInvalidID            uint32 = 501
	CodeDuplicateEntry       uint32 = 502
	CodeMissingEntry         uint32 = 503
	CodeInvalidEntry         uint32 = 504
	CodeUnknownID            uint32 = 505
	CodeInvalidLength        uint32 = 506
	CodeInvalidHost          uint32 = 507
	CodeInvalidPort          uint32 = 508
	CodeInvalidProtocol      uint32 = 509
	CodeInvalidCodec         uint32 = 510
	CodeInvalidJson          uint32 = 511
)

var (
	errUnsupportedTokenType = stderrors.New("Unsupported token type")
	errInvalidID            = stderrors.New("ID is invalid")
	errDuplicateEntry       = stderrors.New("Duplicate entry")
	errMissingEntry         = stderrors.New("Missing entry")
	errInvalidEntry         = stderrors.New("Invalid entry")
	errUnknownID            = stderrors.New("Unknown ID")
	errInvalidLength        = stderrors.New("Invalid length")
	errInvalidHost          = stderrors.New("Invalid host")
	errInvalidPort          = stderrors.New("Invalid port")
	errInvalidProtocol      = stderrors.New("Invalid protocol")
	errInvalidCodec         = stderrors.New("Invalid codec")
	errInvalidJson          = stderrors.New("Invalid json")
)

// ErrUnsupportedTokenType is when the type passed does not match the expected token type.
func ErrUnsupportedTokenType() error {
	return errors.WithCode(errUnsupportedTokenType, CodeUnsupportedTokenType)
}

func ErrInvalidID(id []byte) error {
	return errors.WithLog(printableID(id), errInvalidID, CodeInvalidID)
}
func ErrDuplicateEntry(id []byte) error {
	return errors.WithLog(printableID(id), errDuplicateEntry, CodeDuplicateEntry)
}
func ErrMissingEntry() error {
	return errors.WithCode(errMissingEntry, CodeMissingEntry)
}
func ErrInvalidEntry(id []byte) error {
	return errors.WithLog(printableID(id), errInvalidEntry, CodeInvalidEntry)
}
func ErrUnknownID(id []byte) error {
	return errors.WithLog(printableID(id), errUnknownID, CodeUnknownID)
}
func ErrInvalidLength() error {
	return errors.WithCode(errInvalidLength, CodeInvalidLength)
}
func ErrInvalidHost() error {
	return errors.WithCode(errInvalidHost, CodeInvalidHost)
}
func ErrInvalidPort() error {
	return errors.WithCode(errInvalidPort, CodeInvalidPort)
}
func ErrInvalidProtocol() error {
	return errors.WithCode(errInvalidProtocol, CodeInvalidProtocol)
}
func ErrInvalidCodec(codec string) error {
	return errors.WithLog(codec, errInvalidCodec, CodeInvalidCodec)
}
func ErrInvalidJson() error {
	return errors.WithCode(errInvalidJson, CodeInvalidJson)
}

// id's are stored as bytes, but most are ascii text
// if in ascii, just convert to string
// if not, hex-encode it and prefix with 0x
func printableID(id []byte) string {
	if len(id) == 0 {
		return "<nil>"
	}
	if isSafeAscii(id) {
		return string(id)
	}
	return "0x" + hex.EncodeToString(id)
}

// require all bytes between 0x20 and 0x7f
func isSafeAscii(id []byte) bool {
	for _, c := range id {
		if c < 0x20 || c > 0x7f {
			return false
		}
	}
	return true
}
