package errors

import (
	"fmt"

	"github.com/pkg/errors"
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
	return code == ErrInternal.ABCICode()
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
	return ErrPanic.Newf("%v", p)
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

func WithType(err error, obj interface{}) error {
	return Wrap(err, fmt.Sprintf("%T", obj))
}
