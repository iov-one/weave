package errors

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

// Append clubs together all provided errors. Nil values are ignored.
//
// If given error implements unpacker interface, it is flattened. All
// represented by this error container errors are directly included into the
// result set rather than through the container. This means that
//   Append(Append(err1, err2), Append(err3), err4)
// produce the same result as
//   Append(err1, err2, err3, err4)
// Because not all errors implement unpacker interface, the internal
// representation of the constructed error relation can be a tree. For example,
// the following code will result in a tree-like error chain.
//   Append(err1, Wrap(Append(err2, err3), "w"))
//
// When implementing an error that satisfies unpacker interface, keep in mind
// that Append function destroys such error and consume only contained by it
// errors. Implement unpacker interface only for error containers, that do not
// carry any additional information.
func Append(errs ...error) error {
	// Always build the multi error collection from scratch to avoid slice
	// modyfications.
	var res multiError

	for _, e := range errs {
		res = appendError(res, e)
	}
	if len(res) == 0 {
		return nil
	}
	return res
}

// appendError extends given multiError with provided error. It flattens any
// error that provides the Unpack method.
func appendError(errs multiError, e error) multiError {
	if isNilErr(e) {
		return errs
	}

	if u, ok := e.(unpacker); ok {
		for _, e := range u.Unpack() {
			errs = appendError(errs, e)
		}
		return errs
	}

	return append(errs, e)
}

// multiError represents a group of errors. It "is" all of the represented
// errors.
type multiError []error

var _ unpacker = (multiError)(nil)

// Unpack implements unpacker interface.
func (errs multiError) Unpack() []error {
	return errs
}

// Error satisfies the error interface and returns a serialized version of the content.
func (errs multiError) Error() string {
	switch len(errs) {
	case 0:
		return "<nil>"
	case 1:
		return errs[0].Error()
	}

	msgs := make([]string, len(errs))
	for i, err := range errs {
		// When dealing with a multi error, this might be a nested
		// multierror. Because Error method lacks context and cannot
		// determine what level of nesting it is in, we must parse the
		// method output, find all list items and increase the
		// indentetion by one.
		items := strings.Split(err.Error(), "\n")
		for n, it := range items {
			if isListItem(it) {
				items[n] = "\t" + it
			}
		}
		m := strings.Join(items, "\n")
		// Remove all last new line characters to avoid multiple blank
		// lines when processing multiple nested multi errors.
		m = strings.TrimRight(m, "\n")
		msgs[i] = "\n\t* " + m
	}

	return fmt.Sprintf(
		"%d errors occurred:%s\n",
		len(msgs), strings.Join(msgs, ""))
}

// StackTrace returns the first stack trace found or nil.
func (errs multiError) StackTrace() errors.StackTrace {
	for _, err := range errs {
		if st := stackTrace(err); st != nil {
			return st
		}
	}
	return nil
}

// isListItem returns true if given string represents an item of a list. This
// is true if the message is prefixed with '*' character and any number of
// whitespaces.
func isListItem(msg string) bool {
	for _, c := range msg {
		if unicode.IsSpace(c) {
			continue
		}
		return c == '*'
	}
	return false
}

// ABCICode implementes ABCI coder interface.
func (multiError) ABCICode() uint32 {
	return multiErrorABCICode
}
