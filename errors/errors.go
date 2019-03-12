package errors

import (
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// Global error registry, codes 1-99 are reserved for global errors, 0 is reserved for non-errors
var (

	// ErrInternal represents a general case issue that cannot be
	// categorized as any of the below cases.
	// We start as 1 as 0 is reserved for non-errors
	ErrInternal = Register(1, "internal")

	// ErrUnauthorized is used whenever a request without sufficient
	// authorization is handled.
	ErrUnauthorized = Register(2, "unauthorized")

	// ErrNotFound is used when a requested operation cannot be completed
	// due to missing data.
	ErrNotFound = Register(3, "not found")

	// ErrInvalidMsg is returned whenever an event is invalid and cannot be
	// handled.
	ErrInvalidMsg = Register(4, "invalid message")

	// ErrInvalidModel is returned whenever a message is invalid and cannot
	// be used (ie. persisted).
	ErrInvalidModel = Register(5, "invalid model")

	// ErrDuplicate is returned when there is a record already that has the same
	// unique key/index used
	ErrDuplicate = Register(6, "duplicate")

	// ErrHuman is returned when application reaches a code path which should not
	// ever be reached if the code was written as expected by the framework
	ErrHuman = Register(7, "coding error")

	// ErrCannotBeModified is returned when something that is considered immutable
	// gets modified
	ErrCannotBeModified = Register(8, "cannot be modified")

	// ErrEmpty is returned when a value fails a not empty assertion
	ErrEmpty = Register(9, "value is empty")

	// ErrInvalidState is returned when an object is in invalid state
	ErrInvalidState = Register(10, "invalid state")

	// ErrInvalidType is returned whenever the type is not what was expected
	ErrInvalidType = Register(11, "invalid type")

	// ErrInsufficientAmount is returned when an amount of currency is insufficient, e.g. funds/fees
	ErrInsufficientAmount = Register(12, "insufficient amount")

	// ErrInvalidAmount stands for invalid amount of whatever
	ErrInvalidAmount = Register(13, "invalid amount")

	// ErrInvalidInput stands for general input problems indication
	ErrInvalidInput = Register(14, "invalid input")

	// ErrExpired stands for expired entities, normally has to do with block height expirations
	ErrExpired = Register(15, "expired")

	// ErrOverflow s returned when a computation cannot be completed
	// because the result value exceeds the type.
	ErrOverflow = Register(16, "an operation cannot be completed due to value overflow")

	// ErrPanic is only set when we recover from a panic, so we know to redact potentially sensitive system info
	ErrPanic = Register(111222, "panic")
)

// TMError is the tendermint abci return type with stack trace
type TMError interface {
	stackTracer
	ABCICode() uint32
	ABCILog() string
}

// stackTracer from pkg/errors
type stackTracer interface {
	error
	StackTrace() errors.StackTrace
}

// Register returns an error instance that should be used as the base for
// creating error instances during runtime.
//
// Popular root errors are declared in this package, but extensions may want to
// declare custom codes. This function ensures that no error code is used
// twice. Attempt to reuse an error code results in panic.
//
// Use this function only during a program startup phase.
func Register(code uint32, description string) *Error {
	if e, ok := usedCodes[code]; ok {
		panic(fmt.Sprintf("error with code %d is already registered: %q", code, e.desc))
	}
	err := &Error{
		code: code,
		desc: description,
	}
	usedCodes[err.code] = err
	return err
}

// usedCodes is keeping track of used codes to ensure uniqueness.
var usedCodes = map[uint32]*Error{}

// Error represents a root error.
//
// Weave framework is using root error to categorize issues. Each instance
// created during the runtime should wrap one of the declared root errors. This
// allows error tests and returning all errors to the client in a safe manner.
//
// All popular root errors are declared in this package. If an extension has to
// declare a custom root error, always use Register function to ensure
// error code uniqueness.
type Error struct {
	code uint32
	desc string
}

// Error returns the stored description
func (e Error) Error() string { return e.desc }

// ABCILog returns the stored description, same as Error()
func (e Error) ABCILog() string { return e.desc }

// ABCICode returns the associated ABCICode
func (e Error) ABCICode() uint32 { return e.code }

// New returns a new error. Returned instance is having the root cause set to
// this error. Below two lines are equal
//   e.New("my description")
//   Wrap(e, "my description")
// Allows sprintf format and vararg
func (e *Error) New(description string) error {
	return Wrap(e, description)
}

// Newf is basically New with formatting capabilities
func (e *Error) Newf(description string, args ...interface{}) error {
	return e.New(fmt.Sprintf(description, args...))
}

// Is check if given error instance is of a given kind/type. This involves
// unwrapping given error using the Cause method if available.
//
// Any non weave implementation of an error is tested positive for being
// ErrInternal.
func (kind *Error) Is(err error) bool {
	type causer interface {
		Cause() error
	}

	// Reflect usage is necessary to correctly compare with
	// a nil implementation of an error.
	if kind == nil {
		if err == nil {
			return true
		}
		return reflect.ValueOf(err).IsNil()
	}

	for {
		if err == kind {
			return true
		}

		if c, ok := err.(causer); ok {
			err = c.Cause()
		} else {

			// As a last check, figure out if what we compare is an
			// internal error with a non weave error.
			if kind == ErrInternal {
				if _, ok := err.(*Error); !ok {
					return true
				}
			}

			return false
		}
	}
}

// Wrap extends given error with an additional information.
//
// If the wrapped error does not provide ABCICode method (ie. stdlib errors),
// it will be labeled as internal error.
//
// If err is nil, this returns nil, avoiding the need for an if statement when
// wrapping a error returned at the end of a function
func Wrap(err error, description string) TMError {
	if err == nil {
		return nil
	}

	// take ABCICode from wrapped error, or default ErrInternal
	code := ErrInternal.code
	if p, ok := err.(coder); ok {
		code = p.ABCICode()
	}

	// this will not fire on wrapping a wrappedError,
	// but only on wrapping a registered Error, or stdlib error
	st, ok := err.(stackTracer)
	if !ok {
		st = errors.WithStack(err).(stackTracer)
	}

	return &wrappedError{
		parent: st,
		msg:    description,
		code:   code,
	}
}

// Wrapf extends given error with an additional information.
//
// This function works like Wrap function with additional funtionality of
// formatting the input as specified.
func Wrapf(err error, format string, args ...interface{}) TMError {
	desc := fmt.Sprintf(format, args...)
	return Wrap(err, desc)
}

type wrappedError struct {
	// This error layer description.
	msg string
	// The underlying error that triggered this one.
	parent stackTracer
	// The abci code, inherited from the parent
	code uint32
}

type coder interface {
	ABCICode() uint32
}

func (e *wrappedError) StackTrace() errors.StackTrace {
	if e.parent == nil {
		return nil
	}
	return e.parent.StackTrace()
}

func (e *wrappedError) Error() string {
	// if we have a real error code, show all logs recursively
	if e.parent == nil {
		return e.msg
	}
	return fmt.Sprintf("%s: %s", e.msg, e.parent.Error())
}

func (e *wrappedError) ABCICode() uint32 {
	return e.code
}

func (e *wrappedError) ABCILog() string {
	return e.Error()
}

func (e *wrappedError) Cause() error {
	return e.parent
}

func matchesFile(f errors.Frame, substrs ...string) bool {
	file, _ := fileLine(f)
	for _, sub := range substrs {
		if strings.Contains(file, sub) {
			return true
		}
	}
	return false
}

func fileLine(f errors.Frame) (string, int) {
	// this looks a bit like magic, but follows example here:
	// https://github.com/pkg/errors/blob/v0.8.1/stack.go#L14-L27
	// as this is where we get the Frames
	pc := uintptr(f) - 1
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown", 0
	}
	return fn.FileLine(pc)
}

func trimInternal(st errors.StackTrace) errors.StackTrace {
	// trim our internal parts here
	// manual error creation, or runtime for caught panics
	for matchesFile(st[0],
		// where we create errors
		"weave/errors/errors.go",
		// runtime are added on panics
		"/runtime/",
		// _test is defined in coverage tests, causing failure
		"/_test/") {
		st = st[1:]
	}
	// trim out outer wrappers (runtime.goexit and test library if present)
	for l := len(st) - 1; matchesFile(st[l], "/runtime/", "src/testing/testing.go"); l-- {
		st = st[:l]
	}
	return st
}

func writeSimpleFrame(s io.Writer, f errors.Frame) {
	file, line := fileLine(f)
	// cut file at "github.com/"
	// TODO: generalize better for other hosts?
	chunks := strings.SplitN(file, "github.com/", 2)
	if len(chunks) == 2 {
		file = chunks[1]
	}
	fmt.Fprintf(s, " [%s:%d]", file, line)
}

// Format works like pkg/errors, with additions.
// %s is just the error message
// %+v is the full stack trace
// %v appends a compressed [filename:line] where the error
//    was created
//
// Inspired by https://github.com/pkg/errors/blob/v0.8.1/errors.go#L162-L176
func (e *wrappedError) Format(s fmt.State, verb rune) {
	// normal output here....
	if verb != 'v' {
		fmt.Fprintf(s, e.ABCILog())
		return
	}
	// work with the stack trace... whole or part
	stack := trimInternal(e.StackTrace())
	if s.Flag('+') {
		fmt.Fprintf(s, "%+v\n", stack)
		fmt.Fprintf(s, e.ABCILog())
	} else {
		fmt.Fprintf(s, e.ABCILog())
		writeSimpleFrame(s, stack[0])
	}
}

// hasErrorCode checks if this error would return the named error code
// only used internally for Redact to avoid issues that have to do with
// Is expecting the same reference for ErrInternal
func hasErrorCode(err error, code uint32) bool {
	if tm, ok := err.(coder); ok {
		return tm.ABCICode() == code
	}
	return code == ErrInternal.ABCICode()
}

// NormalizePanic converts a panic into a redacted error
//
// We want the whole stack trace for logging
// but should show nothing over the ABCI interface....
func NormalizePanic(p interface{}) error {
	// TODO, handle this better??? for stack traces
	// if err, isErr := p.(error); isErr {
	// 	return Wrap(err, "normalized panic")
	// }
	return ErrPanic.Newf("%v", p)
}

// Redact will replace all panic errors with a generic message
func Redact(err error) error {
	if hasErrorCode(err, ErrPanic.code) {
		return ErrInternal
	}
	return err
}

// Recover takes a pointer to the returned error,
// and sets it upon panic
func Recover(err *error) {
	if r := recover(); r != nil {
		*err = NormalizePanic(r)
	}
}

// WithType is a helper to augment an error with a corresponding type message
func WithType(err error, obj interface{}) error {
	return Wrap(err, fmt.Sprintf("%T", obj))
}
