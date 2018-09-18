package nft

import (
	stderrors "errors"
	"github.com/iov-one/weave/errors"
)

// nft reserves 500~550
const (
	CodeUnsupportedTokenType uint32 = 500
	CodeInvalidID            uint32 = 501
	CodeInvalidApprovalCount uint32 = 502
	CodeInvalidDataLength    uint32 = 503
)

var (
	errUnsupportedTokenType = stderrors.New("Unsupported token type")
	errInvalidID            = stderrors.New("Id is invalid")
	errInvalidApprovalCount = stderrors.New("Invalid approval count")
)

// ErrUnsupportedTokenType is when the type passed does not match the expected token type.
func ErrUnsupportedTokenType() error {
	return errors.WithCode(errUnsupportedTokenType, CodeUnsupportedTokenType)
}

func ErrInvalidID() error {
	return errors.WithCode(errInvalidID, CodeInvalidID)
}
func ErrInvalidApprovalCount() error {
	return errors.WithCode(errInvalidApprovalCount, CodeInvalidApprovalCount)
}
