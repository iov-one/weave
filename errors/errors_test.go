package errors

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

func TestErrors(t *testing.T) {
	cases := map[string]struct {
		err      error
		wantRoot Error
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
			err:      Wrap(errors.New("stdlib"), "outer"),
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
			err:      Wrap(Wrap(errors.New("stdlib"), "inner"), "outer"),
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

func TestIs(t *testing.T) {
	cases := map[string]struct {
		a      error
		b      error
		wantIs bool
	}{
		"instance of the same error, even if internal": {
			a:      ErrInternal,
			b:      ErrInternal,
			wantIs: true,
		},
		"two different internal errors": {
			a:      errors.New("one"),
			b:      errors.New("two"),
			wantIs: false,
		},
		"two different coded errors": {
			a:      ErrNotFound,
			b:      ErrInvalidModel,
			wantIs: false,
		},
		"two different internal and wrapped  errors": {
			a:      Wrap(errors.New("a not found"), "where is a?"),
			b:      Wrap(ErrInternal, "b not found"),
			wantIs: false,
		},
		"two equal coded errors": {
			a:      Wrap(ErrNotFound, "a not found"),
			b:      Wrap(ErrNotFound, "b not found"),
			wantIs: true,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if got := Is(tc.a, tc.b); got != tc.wantIs {
				t.Fatal("unexpected result")
			}
		})
	}
}

func TestWrapEmpty(t *testing.T) {
	if err := Wrap(nil, "wrapping <nil>"); err != nil {
		t.Fatal(err)
	}
}
