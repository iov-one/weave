//nolint
package errors

import (
	"fmt"

	"github.com/pkg/errors"
)

// ABCI Response Codes
// Base SDK reserves 0 ~ 99.
const (
	CodeInternalErr         uint32 = 1
	CodeTxParseError        uint32 = 2
	CodeUnauthorized        uint32 = 3
	CodeUnknownRequest      uint32 = 4
	CodeUnrecognizedAddress uint32 = 5
)

var (
	errDecoding            = fmt.Errorf("Error decoding input")
	errTooLarge            = fmt.Errorf("Input size too large")
	errUnknownTxType       = fmt.Errorf("Tx type unknown")
	errUnauthorized        = fmt.Errorf("Unauthorized")
	errMissingSignature    = fmt.Errorf("Signature missing")
	errInvalidSignature    = fmt.Errorf("Signature invalid")
	errUnrecognizedAddress = fmt.Errorf("Unrecognized Address")
)

// IsSameError returns true if these errors have the same root cause.
// pattern is the expected error type and should always be non-nil
// err may be anything and returns true if it is a wrapped version of pattern
func IsSameError(pattern error, err error) bool {
	return err != nil && (errors.Cause(err) == errors.Cause(pattern))
}

// HasErrorCode checks if this error would return the named error code
func HasErrorCode(err error, code uint32) bool {
	if tm, ok := err.(TMError); ok {
		return tm.ABCICode() == code
	}
	return code == CodeInternalErr
}

func ErrUnknownTxType(tx interface{}) TMError {
	msg := fmt.Sprintf("%T", tx)
	return WithLog(msg, errUnknownTxType, CodeUnknownRequest)
}
func IsUnknownTxTypeErr(err error) bool {
	return IsSameError(errUnknownTxType, err)
}

func ErrUnrecognizedAddress(addr string) TMError {
	return WithLog(addr, errUnrecognizedAddress, CodeUnrecognizedAddress)
}
func IsUnrecognizedAddressErr(err error) bool {
	return IsSameError(errUnrecognizedAddress, err)
}

// ErrInternal is a generic error code when we cannot return any more
// useful info
func ErrInternal(msg string) TMError {
	return New(msg, CodeInternalErr)
}

// IsInternalErr matches any error that is not classified
func IsInternalErr(err error) bool {
	return HasErrorCode(err, CodeInternalErr)
}

// ErrDecoding is generic when we cannot parse the transaction input
func ErrDecoding() TMError {
	return WithCode(errDecoding, CodeTxParseError)
}
func IsDecodingErr(err error) bool {
	return HasErrorCode(err, CodeTxParseError)
}

// ErrTooLarge is a specific decode error when we pass the max tx size
func ErrTooLarge() TMError {
	return WithCode(errTooLarge, CodeTxParseError)
}
func IsTooLargeErr(err error) bool {
	return IsSameError(errTooLarge, err)
}

// ErrUnauthorized is a generic denial.
// You can use a more specific cause if you wish, such as ErrInvalidSignature
func ErrUnauthorized() TMError {
	return WithCode(errUnauthorized, CodeUnauthorized)
}

// IsUnauthorizedErr is generic helper for any unauthorized errors,
// also specific sub-types
func IsUnauthorizedErr(err error) bool {
	return HasErrorCode(err, CodeUnauthorized)
}

func ErrMissingSignature() TMError {
	return WithCode(errMissingSignature, CodeUnauthorized)
}
func IsMissingSignatureErr(err error) bool {
	return IsSameError(errMissingSignature, err)
}

// ErrInvalidSignature is when the
func ErrInvalidSignature() TMError {
	return WithCode(errInvalidSignature, CodeUnauthorized)
}
func IsInvalidSignatureErr(err error) bool {
	return IsSameError(errInvalidSignature, err)
}
