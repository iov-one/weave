package errors

import (
	"errors"
	"reflect"
	"testing"
)

func TestAppend(t *testing.T) {
	var (
		myErrNotFound  = Wrap(ErrNotFound, "test")
		myErrState     = Wrap(ErrState, "test")
		myErrWrapMulti = Wrap(Append(ErrDuplicate, ErrDeleted), "inner")
	)

	cases := map[string]struct {
		Input []error
		// Use multiError instance as the Want value. Build it manually
		// (do not use Append) to be certain of the state.
		Want error
	}{
		"nil input": {
			Input: nil,
			Want:  nil,
		},
		"empty input": {
			Input: []error{},
			Want:  nil,
		},
		"single nil error": {
			Input: []error{nil},
			Want:  nil,
		},
		"two nil errors": {
			Input: []error{nil, nil},
			Want:  nil,
		},
		"a nil error and a non nil error": {
			Input: []error{nil, myErrNotFound},
			Want:  multiError{myErrNotFound},
		},
		"only non nil errors": {
			Input: []error{myErrState, myErrNotFound},
			Want:  multiError{myErrState, myErrNotFound},
		},
		"nested error": {
			Input: []error{
				myErrState,
				Append(myErrNotFound, ErrEmpty,
					Append(ErrDuplicate, ErrDeleted),
				),
			},
			Want: multiError{
				myErrState,
				myErrNotFound, ErrEmpty,
				ErrDuplicate, ErrDeleted,
			},
		},
		"nested wrapped error": {
			Input: []error{
				myErrState,
				Append(myErrNotFound, ErrEmpty,
					myErrWrapMulti,
				),
			},
			Want: multiError{
				myErrState,
				myErrNotFound, ErrEmpty,
				// Wrapped error cannot be flattened.
				myErrWrapMulti,
			},
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			got := Append(tc.Input...)
			if !reflect.DeepEqual(got, tc.Want) {
				t.Fatalf("unexpected result: %s", got)
			}
		})
	}
}

func TestMultierrorIs(t *testing.T) {
	cases := map[string]struct {
		Kind   *Error
		Tested error
		WantIs bool
	}{
		"empty multierror is nil": {
			Kind:   nil,
			Tested: Append(),
			WantIs: true,
		},
		"empty multierror is not any other error": {
			Kind:   ErrNotFound,
			Tested: Append(),
			WantIs: false,
		},
		"multi error is all of represented errors: not found": {
			Kind:   ErrNotFound,
			Tested: Append(ErrInput, ErrNotFound, ErrUnauthorized),
			WantIs: true,
		},
		"multi error is all of represented errors: unauthorized": {
			Kind:   ErrUnauthorized,
			Tested: Append(ErrInput, ErrNotFound, ErrUnauthorized),
			WantIs: true,
		},
		"multi error is all of represented errors but not others": {
			Kind:   ErrPanic,
			Tested: Append(ErrInput, ErrNotFound, ErrUnauthorized),
			WantIs: false,
		},
		"multi error is all of represented errors when wrapped": {
			Kind: ErrNotFound,
			Tested: Append(
				Wrap(ErrInput, "foo"),
				Wrap(ErrNotFound, "bar"),
			),
			WantIs: true,
		},
		"multi error is all of represented errors when nested": {
			Kind: ErrNotFound,
			Tested: Append(
				ErrInput,
				Append(
					ErrState,
					Append(ErrSchema, ErrNotFound),
				),
			),

			WantIs: true,
		},
		"multi error is only all of represented errors when nested": {
			Kind: ErrEmpty,
			Tested: Append(
				ErrInput,
				Append(
					ErrState,
					Append(ErrSchema, ErrNotFound),
				),
			),

			WantIs: false,
		},
		"multi error is all of represented errors when nested and wrapped": {
			Kind: ErrNotFound,
			Tested: Wrap(Append(
				ErrInput,
				Wrap(Append(
					ErrState,
					Wrap(Append(ErrSchema, ErrNotFound), "inner"),
				), "middle"),
			), "most outer"),

			WantIs: true,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if is := tc.Kind.Is(tc.Tested); is != tc.WantIs {
				t.Fatalf("unexpected result: %v", is)
			}
		})
	}
}

func TestMultiErrorMessageFormat(t *testing.T) {
	cases := map[string]struct {
		Err     error
		WantMsg string
	}{
		"nil multi error": {
			// This is an unusual case and most likely a bug in the
			// code if you do this, but let's test the result
			// nevertheless.
			Err:     (multiError)(nil),
			WantMsg: "<nil>",
		},
		"empty multi error": {
			// This is an unusual case and most likely a bug in the
			// code if you do this, but let's test the result
			// nevertheless.
			Err:     multiError{},
			WantMsg: "<nil>",
		},
		"single error": {
			Err:     Append(errors.New("my msg")),
			WantMsg: "my msg",
		},
		"two errors": {
			Err: Append(
				errors.New("first msg"),
				errors.New("second msg"),
			),
			WantMsg: `2 errors occurred:
	* first msg
	* second msg
`,
		},
		"wrapped error": {
			Err:     Append(Wrap(errors.New("my msg"), "wrapped")),
			WantMsg: "wrapped: my msg",
		},
		"two errors wrapped": {
			Err: Wrap(Append(
				errors.New("first msg"),
				errors.New("second msg"),
			), "wrapped"),
			WantMsg: `wrapped: 2 errors occurred:
	* first msg
	* second msg
`,
		},
		"nested errors": {
			Err: Append(
				errors.New("first"),
				Append(
					errors.New("second"),
					errors.New("third"),
				),
			),
			WantMsg: `3 errors occurred:
	* first
	* second
	* third
`,
		},
		"nested wrapped errors": {
			Err: Append(
				errors.New("first"),
				Wrap(Append(
					errors.New("second"),
					errors.New("third"),
				), "wrapped"),
			),
			WantMsg: `2 errors occurred:
	* first
	* wrapped: 2 errors occurred:
		* second
		* third
`,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if msg := tc.Err.Error(); msg != tc.WantMsg {
				// Print in a nice human readable form for fast comparison.
				t.Log("want error message (as printed)\n", tc.WantMsg)
				t.Log("got error message (as printed)\n", msg)
				// Escape special characters for easier comparison.
				t.Logf("want: %q", tc.WantMsg)
				t.Logf(" got: %q", msg)
				t.Fatal("unexpected message")
			}
		})
	}
}

func TestMultiErrorABCICodeIsRestricted(t *testing.T) {
	// Ensure thath the multi error code is restricted and cannot by
	// registered by another error instance.
	assertPanics(t, func() {
		_ = Register(multiErrorABCICode, "my error")
	})
}

func assertPanics(t testing.TB, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("panic expected")
		}
	}()
	fn()
}
