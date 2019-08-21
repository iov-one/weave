package orm

import (
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestValidateSequence(t *testing.T) {
	cases := map[string]struct {
		bytes   []byte
		wantErr *errors.Error
	}{
		"success": {
			bytes:   []byte{0, 1, 2, 3, 4, 5, 6, 7},
			wantErr: nil,
		},
		"success with sequence": {
			bytes:   weavetest.SequenceID(12345),
			wantErr: nil,
		},
		"failure missing": {
			bytes:   nil,
			wantErr: errors.ErrEmpty,
		},
		"failure invalid length": {
			bytes:   []byte{0, 1},
			wantErr: errors.ErrInput,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			err := ValidateSequence(tc.bytes)
			if !tc.wantErr.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
			}
			if tc.wantErr != nil {
				return
			}
		})
	}
}
