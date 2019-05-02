package validators

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestAccountValidate(t *testing.T) {
	cases := map[string]struct {
		Accounts *Accounts
		WantErr  *errors.Error
	}{
		"valid model": {
			Accounts: &Accounts{
				Metadata: &weave.Metadata{Schema: 1},
				Addresses: [][]byte{
					weavetest.NewCondition().Address(),
					weavetest.NewCondition().Address(),
				},
			},
			WantErr: nil,
		},
		"missing metadata": {
			Accounts: &Accounts{
				Addresses: [][]byte{
					weavetest.NewCondition().Address(),
					weavetest.NewCondition().Address(),
				},
			},
			WantErr: errors.ErrMetadata,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.Accounts.Validate(); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected validation error: %s", err)
			}
		})
	}

}
