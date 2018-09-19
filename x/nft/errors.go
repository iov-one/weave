package nft

import (
	stderrors "errors"
	"github.com/iov-one/weave/errors"
)

// nft reserves 500~550
const (
	CodeUnsupportedTokenType uint32 = 500
	CodeInvalidID            uint32 = 501
)

var (
	errUnsupportedTokenType = stderrors.New("Unsupported token type")
	errInvalidID            = stderrors.New("Id is invalid")
)

// ErrUnsupportedTokenType is when the type passed does not match the expected token type.
func ErrUnsupportedTokenType() error {
	return errors.WithCode(errUnsupportedTokenType, CodeUnsupportedTokenType)
}

func ErrInvalidID() error {
	return errors.WithCode(errInvalidID, CodeInvalidID)
}
