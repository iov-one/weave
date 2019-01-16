package werrors

import (
	"fmt"
	"log"
	"runtime"
)

// E builds an error value from its arguments.
// There must be at least one argument given or E panics. The type of each
// argument determines its meaning. If more than one argument of a given type
// is presented, only the last one is recorded.
//
// The types are:
//
// 	Code
//		The class of error, such as validation error.
// 	string
//		Treated as an error message.
// 	error
//		The underlying error that triggered this one.
//
func E(args ...interface{}) error {
	if len(args) == 0 {
		panic("call to werrors.E with no arguments")
	}
	var err Error
	for _, arg := range args {
		switch arg := arg.(type) {
		case Code:
			err.Code = arg
		case string:
			err.Msg = arg
		case *Error:
			err.Parent = arg
			// Inherit the error code, but prioritize an overwrite.
			if err.Code == 0 {
				err.Code = arg.Code
			}
		case error:
			err.Parent = arg
		default:
			_, file, line, _ := runtime.Caller(1)
			log.Printf("errors.E: bad call from %s:%d: %v", file, line, args)
			return fmt.Errorf("unknown type %T, value %v in error call", arg, arg)
		}
	}

	return &err
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
		return fmt.Sprintf("%s: %s", e.Code, e.Msg)
	}
	return fmt.Sprintf("%s: %s\n\t%s", e.Code, e.Msg, e.Parent.Error())
}

func (e *Error) ABCICode() uint32 {
	// Most outside error code should be the most precise, so prioritize it.
	if e.Code != 0 {
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
	// First 100 error codes are reserved for internal errors. We do not
	// want to expose internal error or implementation details, therefore
	// those are providing partial information.
	switch {
	case e.Code == 0:
		return "internal error"
	case e.Code <= 100:
		return e.Code.String() + " error"
	default:
		return e.Error()
	}
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
