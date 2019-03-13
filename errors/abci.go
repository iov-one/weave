package errors

import (
	"errors"
	"reflect"
)

// ABCIInfo returns the ABCI error information as consumed by the tenderemint
// client.
// This function provides a full error infromation
func ABCIInfo(err error) (uint32, string) {
	if err == nil || reflect.ValueOf(err).IsNil() {
		return 0, ""
	}

	// All non-weave errors are returning a generic result because their
	// content is an implementation detail and must not be exposed.
	if e, ok := weaveErr(err); ok {
		return e.code, err.Error()
	}
	return genericABCICode, genericABCILog
}

const (
	genericABCICode = 1
	genericABCILog  = "internal error"
)

// isWeaveErr test if given error represents an Error provided by this package.
// This function is testing for the causer interface as well and unwraps the
// error.
func weaveErr(err error) (*Error, bool) {
	for {
		if e, ok := err.(*Error); ok {
			return e, true
		}

		if c, ok := err.(causer); ok {
			err = c.Cause()
		} else {
			return nil, false
		}
	}
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
		return errors.New(genericABCILog)
	}
	if _, ok := weaveErr(err); !ok {
		return errors.New(genericABCILog)
	}
	return err
}
