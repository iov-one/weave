package errors

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

// Append clubs together all provided errors. Nil values are ignored.
func Append(errs ...error) error {
	// Always build the multi error collection from scratch to avoid slice
	// modyfications.
	var res multiError

	for _, e := range errs {
		if !isNilErr(e) {
			res = append(res, e)
		}
	}
	if len(res) == 0 {
		return nil
	}
	return res
}

// multiError represents a group of errors. It "is" all of the represented
// errors.
type multiError []error

var _ unpacker = (multiError)(nil)

// Unpack implements unpacker interface.
func (e multiError) Unpack() []error {
	return e
}

// Error satisfies the error interface and returns a serialized version of the content.
func (e multiError) Error() string {
	switch len(e) {
	case 0:
		return "<nil>"
	case 1:
		return e[0].Error()
	}

	msgs := make([]string, len(e))
	for i, err := range e {
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
	type stackTracer interface {
		StackTrace() errors.StackTrace
	}

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
func (e multiError) ABCICode() uint32 {
	return multiErrorABCICode
}
