package assert

import "testing"
import "github.com/iov-one/weave/errors"

func TestIsErr(t *testing.T) {
	cases := map[string]struct {
		ErrWant  error
		ErrGot   error
		Result   bool
		WantFail bool
	}{
		"same error": {
			ErrWant:  errors.ErrEmpty,
			ErrGot:   errors.ErrEmpty,
			WantFail: false,
		},
		"compared to nil": {
			ErrWant:  nil,
			ErrGot:   errors.ErrEmpty,
			WantFail: true,
		},
		"both nil": {
			ErrWant:  nil,
			ErrGot:   nil,
			WantFail: false,
		},
		"wrapped": {
			ErrWant:  errors.ErrEmpty,
			ErrGot:   errors.Wrap(errors.ErrEmpty, "test"),
			WantFail: false,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			mock := &tmock{TB: t}
			IsErr(mock, tc.ErrWant, tc.ErrGot)
			failed := mock.failcalls > 0
			if tc.WantFail != failed {
				t.Fatalf("unlexpected failed call state: %d failures", mock.failcalls)
			}
		})
	}
}

func TestFieldErrors(t *testing.T) {
	cases := map[string]struct {
		Err      error
		Name     string
		WantErr  *errors.Error
		WantFail bool
	}{
		"ensure a single error exists and is found": {
			Err:      errors.Field("name", errors.ErrHuman, "invalid human name"),
			Name:     "name",
			WantErr:  errors.ErrHuman,
			WantFail: false,
		},
		"use nil to ensure no error was found": {
			Err:      errors.Field("name", errors.ErrHuman, "invalid human name"),
			Name:     "unknown-name",
			WantErr:  nil,
			WantFail: false,
		},
		"use nil to fail when an error was found but was not expected": {
			Err:      errors.Field("name", errors.ErrHuman, "invalid human"),
			Name:     "name",
			WantErr:  nil,
			WantFail: true,
		},
		"more than one error for a single field is not allowed, even if it is the same error type": {
			Err: errors.Append(
				errors.Field("name", errors.ErrHuman, "first"),
				errors.Field("name", errors.ErrHuman, "second"),
			),
			Name:     "name",
			WantErr:  errors.ErrHuman,
			WantFail: true, // Only one error per name is allowed when testing.
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			mock := &tmock{TB: t}
			FieldError(mock, tc.Err, tc.Name, tc.WantErr)
			failed := mock.failcalls > 0
			if tc.WantFail != failed {
				t.Fatalf("unlexpected failed call state: %d failures", mock.failcalls)
			}
		})
	}
}

// tmock mocks testing.TB and only counts failure calls. It ignores all other
// input.
type tmock struct {
	testing.TB
	failcalls int
}

func (t *tmock) Error(args ...interface{}) {
	t.TB.Log(args...)
	t.failcalls++
}

func (t *tmock) Errorf(s string, args ...interface{}) {
	t.TB.Logf(s, args...)
	t.failcalls++
}

func (t *tmock) Fatal(args ...interface{}) {
	t.TB.Log(args...)
	t.failcalls++
}

func (t *tmock) Fatalf(s string, args ...interface{}) {
	t.TB.Logf(s, args...)
	t.failcalls++
}
