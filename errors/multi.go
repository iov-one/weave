package errors

import (
	"fmt"
	"strings"
)

var _ coder = (*multiErr)(nil)
var _ causer = (*multiErr)(nil)
var _ Multi = (*multiErr)(nil)
var _ error = (*multiErr)(nil)

// Multi is an interface for multiErr to avoid exposing actual implementation
// to the rest of the app.
type Multi interface {
	// Add adds an error to this multiErr
	Add(err error)
	// AddNamed adds an error that could later be retrieved by name
	AddNamed(name string, err error)
	// Named returns a named error or nil
	Named(name string) error
}

// multiErr is a default implementation of errors.Multi.
// It does not support flattening in order to maintain consistent
// behaviour of named errors
type multiErr struct {
	errors     []error
	errorNames map[string]int
}

func (me *multiErr) isEmpty() bool {
	return len(me.errors) == 0
}

func (me *multiErr) first() error {
	if me.isEmpty() {
		return nil
	}
	return me.errors[0]
}

func (me *multiErr) Add(err error) {
	me.errors = append(me.errors, err)
}

func (me *multiErr) AddNamed(name string, err error) {
	me.errors = append(me.errors, err)
	me.errorNames[name] = len(me.errors) - 1
}

func (me *multiErr) Named(name string) error {
	index, ok := me.errorNames[name]
	if ok {
		return me.errors[index]
	}
	return nil
}

func (me *multiErr) Error() string {
	if me.isEmpty() {
		return ""
	}

	if len(me.errors) == 1 {
		return me.first().Error()
	}

	errs := make([]string, len(me.errors))
	for i, err := range me.errors {
		errs[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d errors occurred:\n\t%s\n\n",
		len(me.errors), strings.Join(errs, "\n\t"))
}

// is provides a helper for Error.Is to work with multiErr
func (me *multiErr) is(isFunc func(error) bool) bool {
	for _, err := range me.errors {
		res := isFunc(err)
		if res {
			return res
		}
	}
	return isFunc(nil)
}

// ABCICode returns the error code of a first error consistent with fail-fast approach or falls back to
// internalError code if the error does not satisfy the coder interface.
// Returns success code if the multiErr is empty
func (me *multiErr) ABCICode() uint32 {
	return abciCode(me.first())
}

// Cause returns the cause of a first error consistent with ABCICode
// if the interface is not satisfied - it returns errInternal
// in case multiErr is empty - we return a nil
func (me *multiErr) Cause() error {
	if me.isEmpty() {
		return nil
	}

	c, ok := me.first().(causer)
	if ok {
		return c.Cause()
	}
	return errInternal
}
