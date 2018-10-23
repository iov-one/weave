package sigs

import (
	"fmt"

	"github.com/iov-one/weave/errors"
)

// ABCI Response Codes
// x/auth reserves 20 ~ 29.
const (
	CodeInvalidSequence uint32 = 20
)

var (
	errInvalidSequence = fmt.Errorf("Invalid sequence number")
)

func ErrInvalidSequence(why string, args ...interface{}) error {
	if len(args) > 0 {
		why = fmt.Sprintf(why, args...)
	}
	return errors.WithLog(why, errInvalidSequence, CodeInvalidSequence)
}
func IsInvalidSequenceErr(err error) bool {
	return errors.IsSameError(errInvalidSequence, err)
}

//------ various invalid signatures ----
// all will match IsInvalidSignatureError

func ErrMissingPubkey() error {
	invalidSig := errors.ErrInvalidSignature()
	return errors.WithLog("Missing public key", invalidSig, errors.CodeUnauthorized)
}
func ErrPubkeyAddressMismatch() error {
	invalidSig := errors.ErrInvalidSignature()
	return errors.WithLog("Pubkey and Address don't match", invalidSig, errors.CodeUnauthorized)
}

var IsInvalidSignatureErr = errors.IsInvalidSignatureErr
