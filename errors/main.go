package errors

import (
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// TMError is the tendermint abci return type with stack trace
type TMError interface {
	stackTracer
	ABCICode() uint32
	ABCILog() string
}

// This function is deprecated. Error codes are no longer part of an error API.
//
// New creates an error with the given message and a stacktrace,
// and sets the code and log,
// overriding the state if err was already TMError
func New(log string, code uint32) error {
	// create a new error with stack trace and attach a code
	st := errors.New(log).(stackTracer)
	return tmerror{
		stackTracer: st,
		code:        code,
		log:         log,
	}
}

// WithCode adds a stacktrace if necessary and sets the code and msg,
// overriding the code if err was already TMError
func WithCode(err error, code uint32) TMError {
	// add a stack only if not present
	st, ok := err.(stackTracer)
	if !ok {
		st = errors.WithStack(err).(stackTracer)
	}
	// TODO: preserve log better???
	// and then wrap it with TMError info
	return tmerror{
		stackTracer: st,
		code:        code,
		log:         err.Error(),
	}
}

// WithLog prepends some text to the error, then calls WithCode
// It wraps the original error, so IsSameError will still match on err
//
// Since
func WithLog(prefix string, err error, code uint32) TMError {
	e2 := errors.WithMessage(err, prefix)
	return WithCode(e2, code)
}

// This function was deprecated.
//
// Wrap safely takes any error and promotes it to a TMError.
// Doing nothing on nil or an incoming TMError.
func deprecatedLegacyWrap(err error) TMError {
	// nil or TMError are no-ops
	if err == nil {
		return nil
	}
	// and check for noop
	tm, ok := err.(TMError)
	if ok {
		return tm
	}

	return WithCode(err, CodeInternalErr)
}

//////////////////////////////////////////////////
// tmerror is generic implementation of TMError

type tmerror struct {
	stackTracer
	code uint32
	log  string
}

func (t tmerror) ABCICode() uint32 {
	return t.code
}

func (t tmerror) ABCILog() string {
	return t.log
}

func (t tmerror) Cause() error {
	return errors.Cause(t.stackTracer)
}

// Stacktrace trims off the redundant lines at top and bottom
// of the stack
func (t tmerror) Stacktrace() errors.StackTrace {
	st := t.stackTracer.StackTrace()
	// trim our internal parts here
	// manual error creation, or runtime for caught panics
	for matchesFile(st[0],
		// where we create errors
		"weave/errors/common.go",
		"weave/errors/main.go",
		// runtime are added on panics
		"/runtime/",
		// _test is defined in coverage tests, causing failure
		"/_test/") {
		st = st[1:]
	}
	// trim out outer wrappers (runtime)
	for l := len(st) - 1; matchesFile(st[l], "/runtime/"); l-- {
		st = st[:l]
	}
	return st
}

func (t tmerror) Error() string {
	return t.stackTracer.Error()
}

var (
	_ causer  = tmerror{}
	_ error   = tmerror{}
	_ TMError = tmerror{}
)

// stackTracer from pkg/errors
type stackTracer interface {
	error
	StackTrace() errors.StackTrace
}

// causer from pkg/errors
type causer interface {
	Cause() error
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
	pc := uintptr(f) - 1
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown", 0
	}
	return fn.FileLine(pc)
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
func (t tmerror) Format(s fmt.State, verb rune) {
	// %+v shows all lines,
	if verb == 'v' && s.Flag('+') {
		fmt.Fprintf(s, "%+v\n", t.Stacktrace())
	}
	// always print the normal error
	fmt.Fprintf(s, "(%d) %s", t.code, t.ABCILog())
	// %v just the first
	if verb == 'v' && !s.Flag('+') {
		writeSimpleFrame(s, t.Stacktrace()[0])
	}
}
