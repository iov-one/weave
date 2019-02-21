package errors

import (
	"fmt"

	"github.com/pkg/errors"
)

// TMError is the tendermint abci return type with stack trace
type TMError interface {
	stackTracer
	ABCICode() uint32
	ABCILog() string
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
