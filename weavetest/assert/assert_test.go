package assert

import "testing"
import "github.com/iov-one/weave/errors"

func TestFieldErrors(t *testing.T) {
	cases := map[string]struct {
		Err      error
		Name     string
		WantErrs []*errors.Error
		WantFail int
	}{
		"ensure a single error exists": {
			Err:      errors.Field("name", errors.ErrHuman, "invalid human name"),
			Name:     "name",
			WantErrs: []*errors.Error{errors.ErrHuman},
			WantFail: 0,
		},
		"ensure no errors found": {
			Err:      errors.Field("name", errors.ErrHuman, "invalid human name"),
			Name:     "unknown name",
			WantErrs: nil,
			WantFail: 0,
		},
		"use nil to ensure no error was found": {
			Err:      errors.ErrHuman,
			Name:     "name",
			WantErrs: []*errors.Error{nil},
			WantFail: 0,
		},
		"fail if nil is expected to ensure no error was found": {
			Err:      errors.Field("name", errors.ErrHuman, "invalid human name"),
			Name:     "name",
			WantErrs: []*errors.Error{nil},
			WantFail: 1,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			mock := &tmock{TB: t}
			FieldErrors(mock, tc.Err, tc.Name, tc.WantErrs...)
			if tc.WantFail != mock.failcalls {
				t.Fatalf("want %d fail calls, got %d", tc.WantFail, mock.failcalls)
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
