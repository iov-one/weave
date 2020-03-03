package account

import (
	"github.com/iov-one/weave/errors"
	"testing"
)

func TestValidateBroker(t *testing.T) {
	cases := map[string]struct {
		token   []byte
		wantErr *errors.Error
	}{
		"success email": {
			token:   []byte("orkun@iov.one"),
			wantErr: nil,
		},
		"success bech32 address": {
			token:   []byte("bech32:tiov16hzpmhecd65u993lasmexrdlkvhcxtlnf7f4ws"),
			wantErr: nil,
		},
		"success hex address": {
			token:   []byte("D5C41DDF386EA9C2963FEC37930DBFB32F832FF3"),
			wantErr: nil,
		},
		"failure wrong email format": {
			token:   []byte("orkun@-1iov.one.com"),
			wantErr: errors.ErrInput,
		},
		"failure missing address format": {
			token:   []byte("tiov16hzpmhecd65u993lasmexrdlkvhcxtlnf7f4ws"),
			wantErr: errors.ErrInput,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := validateBroker(tc.token); !tc.wantErr.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
			}
		})
	}
}
