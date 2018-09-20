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
)

var (
	errUnsupportedTokenType = stderrors.New("Unsupported token type")
	errInvalidID            = stderrors.New("Id is invalid")
	errDuplicateEntry       = stderrors.New("Duplicate entry")
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
