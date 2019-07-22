package errors

import (
	"fmt"

	"github.com/pkg/errors"
)

// Field returns an error instance that wraps the original error with
// additional information. It returns `nil` if provided error is `nil`.
// Use this function to create an error instance describing a field/attribute
// error.
// This function might attach a stack trace information.
//
// Use Go naming for the field name. For example, UserName or MaxAge. When the
// error is for a nested field, use dot notation to constrct the path. For
// example, User.Age or User.Name. When the path includes an iterable, use the
// element index starting with 0 as the name, for example Tags.0 or
// Profiles.2.ID
func Field(fieldName string, err error, description string, args ...interface{}) error {
	if isNilErr(err) {
		return nil
	}

	// If this error does not carry the stacktrace information yet, attach
	// one. This should be done only once per error at the lowest frame
	// possible (most inner wrap).
	if stackTrace(err) == nil {
		err = errors.WithStack(err)
	}

	if len(args) > 0 {
		description = fmt.Sprintf(description, args...)
	}

	return &fieldError{
		parent: err,
		field:  fieldName,
		desc:   description,
	}
}

// AppendField is a shortcut function to club together error(s) with a given
// field error.
//
// Use Go naming for the field name. For example, UserName or MaxAge. When the
// error is for a nested field, use dot notation to constrct the path. For
// example, User.Age or User.Name. When the path includes an iterable, use the
// element index starting with 0 as the name, for example Tags.0 or
// Profiles.2.ID
func AppendField(errorsOrNil error, fieldName string, fieldErrOrNil error) error {
	return Append(errorsOrNil, Field(fieldName, fieldErrOrNil, ""))
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
			// Unpacker is a superset of causer. If Unpack() can be
			// called, then we already work on all children of this
			// error. No need to test for causer as it must not
			// return an error that was not part of the Unpack()
			// result.
			return res
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
