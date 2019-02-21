package errors

import (
	stderrors "errors"
	"fmt"

	"github.com/pkg/errors"
)

// ABCI Response Codes
// Base SDK reserves 0 ~ 99.
const (
	CodeInternalErr         uint32 = 1
	CodeTxParseError               = 2
	CodeUnauthorized               = 3
	CodeUnknownRequest             = 4
	CodeUnrecognizedAddress        = 5
	CodeInvalidChainID             = 6
	CodePanic                      = 111222 // TODO: use maxint or such?
)

var (
	errDecoding              = stderrors.New("Error decoding input")
	errTooLarge              = stderrors.New("Input size too large")
	errUnknownTxType         = stderrors.New("Tx type unknown")
	errMissingSignature      = stderrors.New("Signature missing")
	errInvalidSignature      = stderrors.New("Signature invalid")
	errUnrecognizedAddress   = stderrors.New("Unrecognized Address")
	errUnrecognizedCondition = stderrors.New("Unrecognized Condition")
	errInvalidChainID        = stderrors.New("Invalid ChainID")
	errModifyChainID         = stderrors.New("Cannot modify ChainID")
)

// IsSameError returns true if these errors have the same root cause.
// pattern is the expected error type and should always be non-nil
// err may be anything and returns true if it is a wrapped version of pattern
func IsSameError(pattern error, err error) bool {
	return err != nil && (errors.Cause(err) == errors.Cause(pattern))
}

// HasErrorCode checks if this error would return the named error code
func HasErrorCode(err error, code uint32) bool {
	if tm, ok := err.(coder); ok {
		return tm.ABCICode() == code
	}
	return code == CodeInternalErr
}

// NormalizePanic converts a panic into a redacted error
//
// We want the whole stack trace for logging
// but should show nothing over the ABCI interface....
func NormalizePanic(p interface{}) error {
	// TODO, handle this better??? for stack traces
	// if err, isErr := p.(error); isErr {
	// 	return Wrap(err, "normalized panic")
	// }
	msg := fmt.Sprintf("panic: %v", p)
	return Error{code: CodePanic, desc: msg}
}

// Redact will replace all panic errors with a generic message
func Redact(err error) error {
	if HasErrorCode(err, ErrPanic.code) {
		return ErrInternal
	}
	return err
}

// Recover takes a pointer to the returned error,
// and sets it upon panic
func Recover(err *error) {
	if r := recover(); r != nil {
		*err = NormalizePanic(r)
	}
}

// ErrUnrecognizedAddress may be used for empty addresses, or
// badly formatted addresses
func ErrUnrecognizedAddress(addr []byte) error {
	msg := "(nil)"
	if len(addr) > 0 {
		msg = fmt.Sprintf("%X", addr)
	}
	return WithLog(msg, errUnrecognizedAddress, CodeUnrecognizedAddress)
}

// IsUnrecognizedAddressErr returns true iff an error was created
// with ErrUnrecognizedAddress
func IsUnrecognizedAddressErr(err error) bool {
	return IsSameError(errUnrecognizedAddress, err)
}

// ErrUnrecognizedCondition is used for anything that is not
// the proper format
func ErrUnrecognizedCondition(cond []byte) error {
	msg := "(nil)"
	if len(cond) > 0 {
		msg = fmt.Sprintf("%X", cond)
	}
	return WithLog(msg, errUnrecognizedCondition, CodeUnrecognizedAddress)
}

// IsUnrecognizedConditionErr returns true iff an error was created
// with ErrUnrecognizedCondition
func IsUnrecognizedConditionErr(err error) bool {
	return IsSameError(errUnrecognizedCondition, err)
}

// ErrDecoding is generic when we cannot parse the transaction input
func ErrDecoding() error {
	return WithCode(errDecoding, CodeTxParseError)
}

// ErrTooLarge is a specific decode error when we pass the max tx size
func ErrTooLarge() error {
	return WithCode(errTooLarge, CodeTxParseError)
}

// IsUnauthorizedErr is generic helper for any unauthorized errors,
// also specific sub-types
func IsUnauthorizedErr(err error) bool {
	return HasErrorCode(err, CodeUnauthorized)
}

// ErrMissingSignature is returned when no signature is present
func ErrMissingSignature() error {
	return WithCode(errMissingSignature, CodeUnauthorized)
}

// ErrInvalidSignature is when the signature doesn't match
// (bad key, bad nonce, bad chainID)
func ErrInvalidSignature() error {
	return WithCode(errInvalidSignature, CodeUnauthorized)
}

// IsInvalidSignatureErr returns true iff an error was created
// with ErrInvalidSignature
func IsInvalidSignatureErr(err error) bool {
	return IsSameError(errInvalidSignature, err)
}

// ErrInvalidChainID is when the chainID is the wrong format
func ErrInvalidChainID(chainID string) error {
	return WithLog(chainID, errInvalidChainID, CodeInvalidChainID)
}

// ErrModifyChainID is when someone tries to change the chainID
// after genesis
func ErrModifyChainID() error {
	return WithCode(errModifyChainID, CodeInvalidChainID)
}

func WithType(err error, obj interface{}) error {
	return Wrap(err, fmt.Sprintf("%T", obj))
}
