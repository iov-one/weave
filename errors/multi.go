package errors

import (
	"fmt"
	"strings"
)

var _ coder = (*multiErr)(nil)
var _ causer = (*multiErr)(nil)
var _ Multi = (*multiErr)(nil)

// Multi is an interface for multiErr to avoid exposing actual implementation
// to the rest of the app.
type Multi interface {
	// Add adds an error to this multiErr
	Add(err error) Multi
	// AddNamed adds an error that could later be retrieved by name
	AddNamed(name string, err error) Multi
	// Named returns a named error or nil
	Named(name string) error
	// IsEmpty returns true if there are no actual errors registered,
	// this would also return true when registering a bunch of nil errors
	IsEmpty() bool
	error
}

// multiErr is a default implementation of errors.Multi.
// It does not support flattening in order to maintain consistent
// behaviour of named errors. The implementation is not thread-safe
type multiErr struct {
	errors     []error
	errorNames map[string][]int
}

func (me *multiErr) IsEmpty() bool {
	return len(me.errors) == 0
}

func (me *multiErr) first() error {
	if me.IsEmpty() {
		return nil
	}
	return me.errors[0]
}

func (me *multiErr) Add(err error) Multi {
	if err == nil {
		return me
	}
	me.errors = append(me.errors, err)
	return me
}

// This AddNamed implementation would overwrite a pointer to
// a named error if the same name is used twice, while keeping
// the original error in the container for matching.
func (me *multiErr) AddNamed(name string, err error) Multi {
	if err == nil {
		return me
	}
	me.errors = append(me.errors, err)
	me.errorNames[name] = append(me.errorNames[name], len(me.errors)-1)
	return me
}

func (me *multiErr) Named(name string) error {
	indices, ok := me.errorNames[name]
	if ok {
		multi := MultiAdd()
		for _, index := range indices {
			_ = multi.Add(me.errors[index])
		}
		return multi
	}
	return nil
}

func (me *multiErr) Error() string {
	if me.IsEmpty() {
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
	if me.IsEmpty() {
		return isFunc(nil)
	}

	res := false
	for _, err := range me.errors {
		res = isFunc(err)
		if res {
			return res
		}
	}
	return res
}

// ABCICode returns the error code of a first error consistent with fail-fast approach or falls back to
// internalError code if the error does not satisfy the coder interface.
// Returns success code if the multiErr is empty
func (me *multiErr) ABCICode() uint32 {
	return abciCode(me.first())
}

// Cause returns the first error consistent with ABCICode
// if the interface is not satisfied - it returns errInternal
// in case multiErr is empty - we return a nil
func (me *multiErr) Cause() error {
	if me.IsEmpty() {
		return nil
	}

	return me.first()
}

func newMulti() *multiErr {
	return &multiErr{
		errorNames: make(map[string][]int, 0),
	}
}

// MultiAdd allows to create a multiErr with an optional list of errors
func MultiAdd(errs ...error) Multi {
	mErr := newMulti()
	for _, err := range errs {
		_ = mErr.Add(err)
	}
	return mErr
}

// MultiAddNamed creates a multiErr from a named error
func MultiAddNamed(name string, err error) Multi {
	mErr := newMulti()
	return mErr.AddNamed(name, err)
}

// AsMulti is a nil-safe cast for working with multiErr
// in tests
func AsMulti(err error) Multi {
	if err == nil {
		return newMulti()
	}

	if v, ok := err.(Multi); ok {
		return v
	}

	return MultiAdd(err)
}
