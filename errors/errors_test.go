package errors

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
	cases := map[string]struct {
		err      error
		wantRoot *Error
		wantMsg  string
		wantLog  string
	}{
		"weave error": {
			err:      Wrap(ErrNotFound, "404"),
			wantRoot: ErrNotFound,
			wantMsg:  "404: " + ErrNotFound.desc,
			wantLog:  "404: " + ErrNotFound.desc,
		},
		"wrap of a weave error": {
			err:      Wrap(Wrap(ErrNotFound, "404"), "outer"),
			wantRoot: ErrNotFound,
			wantMsg:  "outer: 404: " + ErrNotFound.desc,
			wantLog:  "outer: 404: " + ErrNotFound.desc,
		},
		"wrap of an stdlib error": {
			err:      Wrap(fmt.Errorf("stdlib"), "outer"),
			wantRoot: ErrInternal,
			wantMsg:  "outer: stdlib",
			wantLog:  "outer: stdlib",
		},
		"deep wrap of a weave error": {
			err:      Wrap(Wrap(Wrap(ErrNotFound, "404"), "inner"), "outer"),
			wantRoot: ErrNotFound,
			wantMsg:  "outer: inner: 404: " + ErrNotFound.desc,
			wantLog:  "outer: inner: 404: " + ErrNotFound.desc,
		},
		"deep wrap of an stdlib error": {
			err:      Wrap(Wrap(fmt.Errorf("stdlib"), "inner"), "outer"),
			wantRoot: ErrInternal,
			wantMsg:  "outer: inner: stdlib",
			wantLog:  "outer: inner: stdlib",
		},
		"normalize panic handles strings": {
			err:      NormalizePanic("foo"),
			wantRoot: ErrPanic,
			wantMsg:  "foo: panic",
			wantLog:  "foo: panic",
		},
		"normalize panic handles errors": {
			err:      NormalizePanic(fmt.Errorf("message")),
			wantRoot: ErrPanic,
			wantMsg:  "message: panic",
			wantLog:  "message: panic",
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if code := errCode(tc.err); code != tc.wantRoot.code {
				t.Fatalf("want %d code, got %d", tc.wantRoot.code, code)
			}
			if msg := tc.err.Error(); msg != tc.wantMsg {
				t.Errorf("want %q, got %q", tc.wantMsg, msg)
			}
			if log := errLog(tc.err); log != tc.wantLog {
				t.Fatalf("want %q log message, got %s", tc.wantLog, log)
			}
		})
	}
}

func errCode(err error) uint32 {
	type coder interface {
		ABCICode() uint32
	}
	if e, ok := err.(coder); ok {
		return e.ABCICode()
	}
	// This error does not implement required interface, so return
	// something that can be spotted in a failing test
	return math.MaxUint16
}

func errLog(err error) string {
	type logger interface {
		ABCILog() string
	}
	if e, ok := err.(logger); ok {
		return e.ABCILog()
	}
	return ""
}

func TestCause(t *testing.T) {
	std := fmt.Errorf("This is stdlib error")

	cases := map[string]struct {
		err  error
		root error
	}{
		"Errors are self-causing": {
			err:  ErrNotFound,
			root: ErrNotFound,
		},
		"Wrap reveals root cause": {
			err:  ErrNotFound.New("foo"),
			root: ErrNotFound,
		},
		"Cause works for stderr as root": {
			err:  Wrap(std, "Some helpful text"),
			root: std,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if got := errors.Cause(tc.err); got != tc.root {
				t.Fatal("unexpected result")
			}
		})
	}
}

func TestErrorIs(t *testing.T) {
	cases := map[string]struct {
		a      *Error
		b      error
		wantIs bool
	}{
		"instance of the same error": {
			a:      ErrNotFound,
			b:      ErrNotFound,
			wantIs: true,
		},
		"instance of the same error, even if internal": {
			a:      ErrInternal,
			b:      ErrInternal,
			wantIs: true,
		},
		"two different coded errors": {
			a:      ErrNotFound,
			b:      ErrInvalidModel,
			wantIs: false,
		},
		"successful comparison to a wrapped error": {
			a:      ErrNotFound,
			b:      errors.Wrap(ErrNotFound, "gone"),
			wantIs: true,
		},
		"unsuccessful comparison to a wrapped error": {
			a:      ErrNotFound,
			b:      errors.Wrap(ErrOverflow, "too big"),
			wantIs: false,
		},
		"not equal to stdlib error": {
			a:      ErrNotFound,
			b:      fmt.Errorf("stdlib error"),
			wantIs: false,
		},
		"not equal to a wrapped stdlib error": {
			a:      ErrNotFound,
			b:      errors.Wrap(fmt.Errorf("stdlib error"), "wrapped"),
			wantIs: false,
		},
		"internal error is equal to a stdlib error": {
			a:      ErrInternal,
			b:      fmt.Errorf("stdlib error"),
			wantIs: true,
		},
		"internal error is equal to a wrapped stdlib error": {
			a:      ErrInternal,
			b:      errors.Wrap(fmt.Errorf("stdlib error"), "w-rap"),
			wantIs: true,
		},
		"nil is nil": {
			a:      nil,
			b:      nil,
			wantIs: true,
		},
		"nil is any error nil": {
			a:      nil,
			b:      (*customError)(nil),
			wantIs: true,
		},
		"nil is not not-nil": {
			a:      nil,
			b:      ErrNotFound,
			wantIs: false,
		},
		"not-nil is not nil": {
			a:      ErrNotFound,
			b:      nil,
			wantIs: false,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if got := tc.a.Is(tc.b); got != tc.wantIs {
				t.Fatal("unexpected result")
			}
		})
	}
}

type customError struct {
}

func (customError) Error() string {
	return "custom error"
}

func TestWrapEmpty(t *testing.T) {
	if err := Wrap(nil, "wrapping <nil>"); err != nil {
		t.Fatal(err)
	}
}

func doWrap(err error) error {
	return Wrap(err, "do the do")
}

func TestStackTrace(t *testing.T) {
	cases := map[string]struct {
		err error
		// this is the text we want to see with .Log()
		log string
		// whether the Wrap call is in the stacktrace (not for pkg/errors)
		withWrap bool
	}{
		"New gives us a stacktrace": {
			err:      ErrDuplicate.New("name"),
			log:      "name: duplicate",
			withWrap: true,
		},
		"Wrapping stderr gives us a stacktrace": {
			err:      Wrap(fmt.Errorf("foo"), "standard"),
			log:      "standard: foo",
			withWrap: true,
		},
		"Wrapping pkg/errors gives us clean stacktrace": {
			err:      Wrap(errors.New("bar"), "pkg"),
			log:      "pkg: bar",
			withWrap: false,
		},
		"Wrapping inside another function is still clean": {
			err:      doWrap(fmt.Errorf("indirect")),
			log:      "do the do: indirect",
			withWrap: true,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			// make sure error returns the log
			assert.Equal(t, tc.log, tc.err.Error())

			// make sure we can get a stack trace
			st, ok := tc.err.(stackTracer)
			require.True(t, ok)
			trace := st.StackTrace()
			stack := fmt.Sprintf("%+v", trace)

			// these lines are in all traces, but we want to remove them
			wrap := "github.com/iov-one/weave/errors.Wrap\n"
			errNew := "github.com/iov-one/weave/errors.Error.New\n"
			runtime := "runtime.goexit\n"
			// this is the actual test code that must remains
			thisTest := "github.com/iov-one/weave/errors.TestStackTrace\n"
			assert.Equal(t, tc.withWrap, strings.Contains(stack, wrap))
			assert.True(t, strings.Contains(stack, thisTest))
			assert.True(t, strings.Contains(stack, runtime))

			// verify printing the error produces cleaned stack
			debug := fmt.Sprintf("%+v", tc.err)
			// include the log message
			assert.True(t, strings.Contains(debug, tc.log))
			// and the important lines of the trace
			assert.True(t, strings.Contains(debug, thisTest))
			// but not the garbage
			assert.False(t, strings.Contains(debug, wrap))
			assert.False(t, strings.Contains(debug, errNew))
			assert.False(t, strings.Contains(debug, runtime))

			// verify printing with %v gives minimal info
			medium := fmt.Sprintf("%v", tc.err)
			// include the log message
			assert.True(t, strings.HasPrefix(medium, tc.log))
			// only one line
			assert.False(t, strings.Contains(medium, "\n"))
			// contains a link to where it was created, which must be here, not the Wrap() function
			assert.True(t, strings.Contains(medium, "[iov-one/weave/errors/errors_test.go"))
		})
	}
}
