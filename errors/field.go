package errors

import (
	"fmt"

	"github.com/pkg/errors"
)

// Field returns an error instance that wraps the original error with
// additional information.
// Use this function to create an error instance describing a field/attribute
// error.
func Field(fieldName string, err error, description string) error {
	if isNilErr(err) {
		return nil
	}

	// If this error does not carry the stacktrace information yet, attach
	// one. This should be done only once per error at the lowest frame
	// possible (most inner wrap).
	if stackTrace(err) == nil {
		err = errors.WithStack(err)
	}

	return &fieldError{
		parent: err,
		field:  fieldName,
		desc:   description,
	}
}

type fieldError struct {
	parent error
	field  string
	desc   string
}

func (err *fieldError) Error() string {
	if err.desc == "" {
		return fmt.Sprintf("field %q: %s", err.field, err.parent)
	}
	return fmt.Sprintf("field %q: %s: %s", err.field, err.desc, err.parent)
}

// Cause implements the causer interface.
func (err *fieldError) Cause() error {
	return err.parent
}

// Field implements fielder interface.
func (err *fieldError) Field() string {
	return err.field
}

// FieldErrors returns the list of all errors that are created for the given
// field name.
// An error must be implementing a fielder interface and return a matching
// field name in order to pass the test and be included in the result set.
func FieldErrors(err error, fieldName string) []error {
	if isNilErr(err) {
		return nil
	}

	var res []error
	for {
		if err == nil {
			return res
		}

		if f, ok := err.(fielder); ok {
			if f.Field() == fieldName {
				return append(res, err)
			}
		}

		if u, ok := err.(unpacker); ok {
			for _, e := range u.Unpack() {
				res = append(res, FieldErrors(e, fieldName)...)
			}
		}

		if c, ok := err.(causer); ok {
			err = c.Cause()
		} else {
			return res
		}
	}
}

type fielder interface {
	// Field returns the field name that this error is created for.
	Field() string
}
