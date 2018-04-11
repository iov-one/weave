package errors

import (
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
)

var (
	errDecoding               = fmt.Errorf("Error decoding input")
	errTooLarge               = fmt.Errorf("Input size too large")
	errUnknownTxType          = fmt.Errorf("Tx type unknown")
	errUnauthorized           = fmt.Errorf("Unauthorized")
	errMissingSignature       = fmt.Errorf("Signature missing")
	errInvalidSignature       = fmt.Errorf("Signature invalid")
	errUnrecognizedAddress    = fmt.Errorf("Unrecognized Address")
	errUnrecognizedPermission = fmt.Errorf("Unrecognized Permission")
	errInvalidChainID         = fmt.Errorf("Invalid ChainID")
	errModifyChainID          = fmt.Errorf("Cannot modify ChainID")
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

// NormalizePanic converts a panic into a proper error
func NormalizePanic(p interface{}) error {
	if err, isErr := p.(error); isErr {
		return Wrap(err)
	}
	msg := fmt.Sprintf("panic: %v", p)
	return ErrInternal(msg)
}

// Recover takes a pointer to the returned error,
// and sets it upon panic
func Recover(err *error) {
	if r := recover(); r != nil {
		*err = NormalizePanic(r)
	}
}

// ErrUnknownTxType creates an error for unexpected transaction objects
func ErrUnknownTxType(tx interface{}) error {
	msg := fmt.Sprintf("%T", tx)
	return WithLog(msg, errUnknownTxType, CodeUnknownRequest)
}

// IsUnknownTxTypeErr returns true if an error was created with
// ErrUnknownTxType
func IsUnknownTxTypeErr(err error) bool {
	return IsSameError(errUnknownTxType, err)
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

// ErrUnrecognizedPermission is used for anything that is not
// the proper format
func ErrUnrecognizedPermission(perm []byte) error {
	msg := "(nil)"
	if len(perm) > 0 {
		msg = fmt.Sprintf("%X", perm)
	}
	return WithLog(msg, errUnrecognizedPermission, CodeUnrecognizedAddress)
}

// ErrInternal is a generic error code when we cannot return any more
// useful info
func ErrInternal(msg string) error {
	return New(msg, CodeInternalErr)
}

// IsInternalErr returns true for any error that is not classified
func IsInternalErr(err error) bool {
	return HasErrorCode(err, CodeInternalErr)
}

// ErrDecoding is generic when we cannot parse the transaction input
func ErrDecoding() error {
	return WithCode(errDecoding, CodeTxParseError)
}

// IsDecodingErr returns true for any error with a ParseError code
func IsDecodingErr(err error) bool {
	return HasErrorCode(err, CodeTxParseError)
}

// ErrTooLarge is a specific decode error when we pass the max tx size
func ErrTooLarge() error {
	return WithCode(errTooLarge, CodeTxParseError)
}

// IsTooLargeErr returns true iff an error was created
// with ErrTooLarge
func IsTooLargeErr(err error) bool {
	return IsSameError(errTooLarge, err)
}

// ErrUnauthorized is a generic denial.
// You can use a more specific cause if you wish, such as ErrInvalidSignature
func ErrUnauthorized() error {
	return WithCode(errUnauthorized, CodeUnauthorized)
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

// IsMissingSignatureErr returns true iff an error was created
// with ErrMissingSignature
func IsMissingSignatureErr(err error) bool {
	return IsSameError(errMissingSignature, err)
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

// IsInvalidChainIDErr returns true iff an error was created
// with ErrInvalidChainID
func IsInvalidChainIDErr(err error) bool {
	return IsSameError(errInvalidChainID, err)
}

// ErrModifyChainID is when someone tries to change the chainID
// after genesis
func ErrModifyChainID() error {
	return WithCode(errModifyChainID, CodeInvalidChainID)
}

// IsModifyChainIDErr returns true iff an error was created
// with ErrModifyChainID
func IsModifyChainIDErr(err error) bool {
	return IsSameError(errModifyChainID, err)
}
