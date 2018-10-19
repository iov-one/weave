package nft

import (
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
	errInvalidID            = stderrors.New("Id is invalid")
	errDuplicateEntry       = stderrors.New("Duplicate entry")
	errMissingEntry         = stderrors.New("Missing entry")
	errInvalidEntry         = stderrors.New("Invalid entry")
	errUnknownID            = stderrors.New("Unknown Id")
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

func ErrInvalidID() error {
	return errors.WithCode(errInvalidID, CodeInvalidID)
}
func ErrDuplicateEntry() error {
	return errors.WithCode(errDuplicateEntry, CodeDuplicateEntry)
}
func ErrMissingEntry() error {
	return errors.WithCode(errMissingEntry, CodeMissingEntry)
}
func ErrInvalidEntry() error {
	return errors.WithCode(errInvalidEntry, CodeInvalidEntry)
}
func ErrUnknownID() error {
	return errors.WithCode(errUnknownID, CodeUnknownID)
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
func ErrInvalidCodec() error {
	return errors.WithCode(errInvalidCodec, CodeInvalidCodec)
}
func ErrInvalidJson() error {
	return errors.WithCode(errInvalidJson, CodeInvalidJson)
}
