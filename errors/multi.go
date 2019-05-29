package errors

import (
	"fmt"
	"strings"
)

const multiErrCode uint32 = 100

var _ coder = (*multiErr)(nil)

type multiErr []error

// IsEmpty returns true if there are no errors registered,
func (e multiErr) IsEmpty() bool {
	return len(e) == 0
}

// With returns a new multiErr instance with the given error added.
// Nil values are ignored and multiErr flattened.
func (e multiErr) With(source error) multiErr {
	switch err := source.(type) {
	case nil:
		return e
	case multiErr:
		return e.append(err...)
	}
	return e.append(source)
}

// append copies values into a new array to not let stdlib append modify the the original one
func (e multiErr) append(errs ...error) multiErr {
	r := make(multiErr, len(e), len(e)+len(errs))
	copy(r, e)
	return append(r, errs...)
}

// Error satisfies the error interface and returns a serialized version of the content.
func (e multiErr) Error() string {
	if e.IsEmpty() {
		return ""
	}

	errs := make([]string, len(e))
	for i, err := range e {
		errs[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d errors occurred:\n\t%s\n\n",
		len(e), strings.Join(errs, "\n\t"))
}

// ABCICode returns 100
func (e multiErr) ABCICode() uint32 {
	return multiErrCode
}

// Contains returns true when the given error instance is element of this multiErr.
func (e multiErr) Contains(err *Error) bool {
	for _, v := range e {
		if err.Is(v) {
			return true
		}
	}
	return false
}
