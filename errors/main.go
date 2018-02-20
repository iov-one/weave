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

// Wrap safely takes any error and promotes it to a TMError.
// Doing nothing on nil or an incoming TMError.
func Wrap(err error) TMError {
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
	// TODO: remove this for production....
	return fmt.Sprintf("%+v", t.stackTracer)
	// return t.log
}

func (t tmerror) Cause() error {
	return errors.Cause(t.stackTracer)
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

// Format handles "%+v" to expose the full stack trace
// concept from pkg/errors
func (t tmerror) Format(s fmt.State, verb rune) {
	// special case also show all info
	if verb == 'v' && s.Flag('+') {
		fmt.Fprintf(s, "%+v", t.stackTracer)
	}
	// always print the normal error
	fmt.Fprintf(s, "(%d) %s", t.code, t.ABCILog())
}
