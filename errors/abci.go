package errors

import (
	"errors"
	"fmt"
	"reflect"
)

const (
	// SuccessABCICode declares an ABCI response use 0 to signal that the
	// processing was successful and no error is returned.
	SuccessABCICode = 0

	// All unclassified errors that do not provide an ABCI code are clubbed
	// under an internal error code and a generic message instead of
	// detailed error string.
	internalABCICode uint32 = 1
	internalABCILog         = "internal error"
)

// ABCIInfo returns the ABCI error information as consumed by the tendermint
// client. Returned code and log message should be used as a ABCI response.
// Any error that does not provide ABCICode information is categorized as error
// with code 1.
// When not running in a debug mode all messages of errors that do not provide
// ABCICode information are replaced with generic "internal error". Errors
// without an ABCICode information as considered internal.
func ABCIInfo(err error, debug bool) (uint32, string) {
	if errIsNil(err) {
		return SuccessABCICode, ""
	}

	// Only non-internal errors information can be exposed. Any error that
	// does not explicitly expose its state by providing and ABCI error
	// code must be silenced.
	if code := abciCode(err); code != internalABCICode {
		if debug {
			// Try to trigger full information formatting. This
			// might produce a stacktrace.
			return code, fmt.Sprintf("%+v", err)
		}
		return code, err.Error()
	}

	if debug {
		return internalABCICode, fmt.Sprintf("%+v", err)
	}

	// For internal errors hide the original error message and return
	// generic data.
	return internalABCICode, internalABCILog
}

type coder interface {
	ABCICode() uint32
}

// abciCode test if given error contains an ABCI code and returns the value of
// it if available. This function is testing for the causer interface as well
// and unwraps the error.
func abciCode(err error) uint32 {
	if errIsNil(err) {
		return SuccessABCICode
	}

	for {
		if c, ok := err.(coder); ok {
			return c.ABCICode()
		}

		if c, ok := err.(causer); ok {
			err = c.Cause()
		} else {
			return internalABCICode
		}
	}
}

// errIsNil returns true if value represented by the given error is nil.
//
// Most of the time a simple == check is enough. There is a very narrowed
// spectrum of cases (mostly in tests) where a more sophisticated check is
// required.
func errIsNil(err error) bool {
	if err == nil {
		return true
	}
	if val := reflect.ValueOf(err); val.Kind() == reflect.Ptr {
		return val.IsNil()
	}
	return false
}

// Redact replace all errors that do not initialize with a weave error with a
// generic internal error instance. This function is supposed to hide
// implementation details errors and leave only those that weave framework
// originates.
//
// This is a no-operation function when running in debug mode.
func Redact(err error, debug bool) error {
	if debug {
		return err
	}
	if ErrPanic.Is(err) {
		return errors.New(internalABCILog)
	}
	if abciCode(err) == internalABCICode {
		return errors.New(internalABCILog)
	}
	return err
}
