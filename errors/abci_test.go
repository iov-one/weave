package errors

import (
	"io"
	"testing"
)

func TestABCInfo(t *testing.T) {
	cases := map[string]struct {
		err      error
		debug    bool
		wantCode uint32
		wantLog  string
	}{
		"plain weave error": {
			err:      ErrNotFound,
			debug:    false,
			wantLog:  "not found",
			wantCode: ErrNotFound.code,
		},
		"wrapped weave error": {
			err:      Wrap(Wrap(ErrNotFound, "foo"), "bar"),
			debug:    false,
			wantLog:  "bar: foo: not found",
			wantCode: ErrNotFound.code,
		},
		"nil is empty message": {
			err:      nil,
			debug:    false,
			wantLog:  "",
			wantCode: 0,
		},
		"nil weave error is not an error": {
			err:      (*Error)(nil),
			debug:    false,
			wantLog:  "",
			wantCode: 0,
		},
		"stdlib is generic message": {
			err:      io.EOF,
			debug:    false,
			wantLog:  "internal error",
			wantCode: 1,
		},
		"stdlib returns error message in debug mode": {
			err:      io.EOF,
			debug:    true,
			wantLog:  "EOF",
			wantCode: 1,
		},
		"wrapped stdlib is only a generic message": {
			err:      Wrap(io.EOF, "cannot read file"),
			debug:    false,
			wantLog:  "internal error",
			wantCode: 1,
		},
		"wrapped stdlib is a full message in debug mode": {
			err:      Wrap(io.EOF, "cannot read file"),
			debug:    true,
			wantLog:  "cannot read file: EOF",
			wantCode: 1,
		},
		"custom error": {
			err:      customErr{},
			debug:    false,
			wantLog:  "custom",
			wantCode: 999,
		},
		"custom error in debug mode": {
			err:      customErr{},
			debug:    true,
			wantLog:  "custom",
			wantCode: 999,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			code, log := ABCIInfo(tc.err, tc.debug)
			if code != tc.wantCode {
				t.Errorf("want %d code, got %d", tc.wantCode, code)
			}
			if log != tc.wantLog {
				t.Errorf("want %q log, got %q", tc.wantLog, log)
			}
		})
	}
}

// customErr is a custom implementation of an error that provides an ABCICode
// method.
type customErr struct{}

func (customErr) ABCICode() uint32 { return 999 }

func (customErr) Error() string { return "custom" }
