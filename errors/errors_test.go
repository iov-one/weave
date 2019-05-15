package errors

import (
	"fmt"
	"strings"
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
	"github.com/pkg/errors"
)

func TestMultiErr(t *testing.T) {
	cases := map[string]func(t *testing.T){
		"Named error": func(t *testing.T) {
			name := "Test"
			err := MultiAddNamed(name, ErrEmpty)
			assert.Equal(t, err.Named(name), ErrEmpty)
			assert.Equal(t, err.Named("random"), nil)
		},
		"Named error override": func(t *testing.T) {
			name := "Test"
			err := MultiAddNamed(name, ErrEmpty)
			err.AddNamed(name, ErrState)
			assert.Equal(t, err.Named(name), ErrState)
		},
		"ABCICode is consistent with that of a normal error": func(t *testing.T) {
			err := MultiAdd(ErrEmpty)
			assert.Equal(t, err.(coder).ABCICode(), ErrEmpty.ABCICode())
		},
		"Error() works as expected": func(t *testing.T) {
			err := MultiAdd()
			assert.Equal(t, err.Error(), "")

			err.Add(ErrEmpty)
			assert.Equal(t, strings.Contains(err.Error(), ErrEmpty.Error()), true)

			err.Add(ErrState)
			assert.Equal(t, strings.Contains(err.Error(), ErrEmpty.Error()), true)
			assert.Equal(t, strings.Contains(err.Error(), ErrState.Error()), true)
			assert.Equal(t, strings.Contains(err.Error(), "2"), true)
		},
	}

	for testName, tc := range cases {
		t.Run(testName, tc)
	}
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
			err:  Wrap(ErrNotFound, "foo"),
			root: ErrNotFound,
		},
		"Cause works for stderr as root": {
			err:  Wrap(std, "Some helpful text"),
			root: std,
		},
		"multierr cause is the first error": {
			err:  MultiAdd(ErrState, ErrEmpty),
			root: ErrState,
		},
		"empty multierr cause is nil": {
			err:  MultiAdd(),
			root: nil,
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
		"two different coded errors": {
			a:      ErrNotFound,
			b:      ErrModel,
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
		"multierr with the same error": {
			a:      ErrNotFound,
			b:      MultiAdd(ErrNotFound, ErrState),
			wantIs: true,
		},
		"multierr with nil error": {
			a:      ErrNotFound,
			b:      MultiAdd(nil),
			wantIs: false,
		},
		"multierr with different error": {
			a:      ErrNotFound,
			b:      MultiAdd(ErrState),
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
