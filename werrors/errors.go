package werrors

import (
	"fmt"
)

// New returns a new error instance. Avoid using Internal code when creating
// errors.
func New(cause Code, description string) error {
	return &Error{
		Code: cause,
		Msg:  description,
	}
}

// Wrap extends given error with additional information.
//
// If the wrapped error does not provide ABCICode method (ie. stdlib errors),
// it will be labeled as internal error.
func Wrap(err error, description string) error {
	return &Error{
		Parent: err,
		Msg:    description,
	}
}

type Error struct {
	// Code represents the type of error.
	Code Code

	// This error layer description.
	Msg string

	// The underlying error that triggered this one, if any.
	Parent error
}

func (e *Error) Error() string {
	if e.Parent == nil {
		return e.Msg
	}
	return fmt.Sprintf("%s: %s", e.Msg, e.Parent.Error())
}

func (e *Error) ABCICode() uint32 {
	// Most outside error code should be the most precise, so prioritize it.
	if e.Code != Internal {
		return uint32(e.Code)
	}

	type coder interface {
		ABCICode() uint32
	}

	if p, ok := e.Parent.(coder); ok {
		return p.ABCICode()
	}
	return 0
}

func (e *Error) ABCILog() string {
	// Internal error must not be revealed as a public API message.
	// Instead, return generic description.
	if e.ABCICode() == uint32(Internal) {
		return "internal error"
	}
	return e.Error()
}

func (e *Error) Cause() error {
	type causer interface {
		Cause() error
	}
	// Casuse returns the root cause of an error, regardless how many
	// layers there are.
	if e.Parent != nil {
		if c, ok := e.Parent.(causer); ok {
			if cause := c.Cause(); cause != nil {
				return cause
			}
		}
		return e.Parent
	}
	return nil
}
