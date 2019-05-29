package errors

import (
	stdlib "errors"
	"fmt"
	"github.com/pkg/errors"
	"testing"
)

func TestCause(t *testing.T) {
	std := stdlib.New("this is a stdlib error")

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
			b:      MultiErr.With(ErrNotFound).With(ErrState),
			wantIs: true,
		},
		"multierr with random order": {
			a:      ErrNotFound,
			b:      MultiErr.With(ErrState).With(ErrNotFound),
			wantIs: true,
		},
		"multierr with wrapped err": {
			a:      ErrNotFound,
			b:      MultiErr.With(Wrap(ErrNotFound, "test")),
			wantIs: true,
		},
		"multierr with nil error": {
			a:      ErrNotFound,
			b:      MultiErr.With(nil),
			wantIs: false,
		},
		"multierr with different error": {
			a:      ErrNotFound,
			b:      MultiErr.With(ErrState),
			wantIs: false,
		},
		"multierr from nil": {
			a:      nil,
			b:      MultiErr.With(ErrState),
			wantIs: false,
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if got := tc.a.Is(tc.b); got != tc.wantIs {
				t.Fatalf("unexpected result - got:%v want: %v", got, tc.wantIs)
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
