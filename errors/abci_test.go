package errors

import (
	"io"
	"testing"
)

func TestABCInfo(t *testing.T) {
	cases := map[string]struct {
		err      error
		wantCode uint32
		wantLog  string
	}{
		"plain weave error": {
			err:      ErrNotFound,
			wantLog:  "not found",
			wantCode: ErrNotFound.code,
		},
		"wrapped weave error": {
			err:      Wrap(Wrap(ErrNotFound, "foo"), "bar"),
			wantLog:  "bar: foo: not found",
			wantCode: ErrNotFound.code,
		},
		"nil is empty message": {
			err:      nil,
			wantLog:  "",
			wantCode: 0,
		},
		"nil weave error is not an error": {
			err:      (*Error)(nil),
			wantLog:  "",
			wantCode: 0,
		},
		"stdlib is generic message": {
			err:      io.EOF,
			wantLog:  "internal error",
			wantCode: 1,
		},
		"wrapped stdlib is only a generic message": {
			err:      Wrap(io.EOF, "cannot read file"),
			wantLog:  "internal error",
			wantCode: 1,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			code, log := ABCIInfo(tc.err)
			if code != tc.wantCode {
				t.Errorf("want %d code, got %d", tc.wantCode, code)
			}
			if log != tc.wantLog {
				t.Errorf("want %q log, got %q", tc.wantLog, log)
			}
		})
	}
}
