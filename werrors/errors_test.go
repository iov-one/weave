package werrors

import (
	"errors"
	"math"
	"testing"
)

func TestErrors(t *testing.T) {
	cases := map[string]struct {
		err      error
		wantCode Code
		wantMsg  string
		wantLog  string
	}{
		"weave error": {
			err:      New(NotFound, "404"),
			wantCode: NotFound,
			wantMsg:  "404",
			wantLog:  "404",
		},
		"wrap of a weave error": {
			err:      Wrap(New(NotFound, "404"), "outer"),
			wantCode: NotFound,
			wantMsg:  "outer: 404",
			wantLog:  "outer: 404",
		},
		"wrap of an stdlib error": {
			err:      Wrap(errors.New("stdlib"), "outer"),
			wantCode: Internal,
			wantMsg:  "outer: stdlib",
			wantLog:  "internal error",
		},
		"deep wrap of a weave error": {
			err:      Wrap(Wrap(New(NotFound, "404"), "inner"), "outer"),
			wantCode: NotFound,
			wantMsg:  "outer: inner: 404",
			wantLog:  "outer: inner: 404",
		},
		"deep wrap of an stdlib error": {
			err:      Wrap(Wrap(errors.New("stdlib"), "inner"), "outer"),
			wantCode: Internal,
			wantMsg:  "outer: inner: stdlib",
			wantLog:  "internal error",
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if code := errCode(tc.err); code != tc.wantCode {
				t.Fatalf("want %d code, got %d", tc.wantCode, code)
			}
			// Ensure that Code.Is is working well
			if !tc.wantCode.Is(tc.err) {
				t.Fatal("Code.Is returns unexpected result")
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

func errCode(err error) Code {
	type coder interface {
		ABCICode() uint32
	}
	if e, ok := err.(coder); ok {
		return Code(e.ABCICode())
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
