package errors

import (
	"fmt"
	"strings"
)

type Multi interface {
	Add(err error)
	AddNamed(name string, err error)
	Named(name string) error
	Is(err error) bool
}

// multiErr is a default implementation of errors.Multi.
// It does not support flattening in order to maintain consistent
// behaviour of named errors
type multiErr struct {
	errors []error
	errorNames map[string]int
}

func (me *multiErr) isEmpty() bool {
	return len(me.errors) == 0
}

// Add adds an error to this multiErr
func (me *multiErr) Add(err error) {
	me.errors = append(me.errors, err)
}

// AddNamed adds an error that could later be retrieved by name
func (me *multiErr) AddNamed(name string, err error) {
	me.errors = append(me.errors, err)
	me.errorNames[name] = len(me.errors) - 1
}

// Named returns a named error or nil
func (me *multiErr) Named(name string) error {
	index, ok := me.errorNames[name]
	if ok {
		return me.errors[index]
	}
	return nil
}

func(me *multiErr) Error() string {
	if len(me.errors) == 1 {
		return fmt.Sprintf("1 error occurred:\n\t* %s\n\n", me.errors[0])
	}

	points := make([]string, len(me.errors))
	for i, err := range me.errors {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d errors occurred:\n\t%s\n\n",
		len(me.errors), strings.Join(points, "\n\t"))
}

func (me *multiErr) Is(err error) error {
	if merr, ok := err.(multiErr); ok {

	}
}

var _ coder = (*multiErr)(nil)
var _ causer = (*multiErr)(nil)
var _ Multi = (*multiErr)(nil)
var _ error = (*multiErr)(nil)



// ABCICode returns the error code of a first error consistent with fail-fast approach or falls back to
// internalError code if the error does not satisfy the coder interface.
// Returns success code if the multiErr is empty
func (me *multiErr) ABCICode() uint32 {
	if me.isEmpty() {
		return SuccessABCICode
	}

	c, ok := me.errors[0].(coder)
	if ok {
		return c.ABCICode()
	}
	return internalABCICode
}

// Cause returns the cause of a first error consistent with ABCICode
// if the interface is not satisfied - it returns errInternal
// in case multiErr is empty - we return a nil
func (me *multiErr) Cause() error {
	if me.isEmpty() {
		return nil
	}
	c, ok := me.errors[0].(causer)
	if ok {
		return c.Cause()
	}
	return errInternal
}


