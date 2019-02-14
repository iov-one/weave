package feedist

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

func TestRevenueValidate(t *testing.T) {
	addr := weave.Address("f427d624ed29c1fae0e2")

	cases := map[string]struct {
		model   Revenue
		wantErr error
	}{
		"valid model": {
			model: Revenue{
				Admin: addr,
				Recipients: []*Recipient{
					{Weight: 1, Address: addr},
				},
			},
			wantErr: nil,
		},
		"admin address must be present": {
			model: Revenue{
				Admin: nil,
				Recipients: []*Recipient{
					{Weight: 1, Address: addr},
				},
			},
			wantErr: errors.InvalidModelErr,
		},
		"at least one recipient must be given": {
			model: Revenue{
				Admin:      addr,
				Recipients: []*Recipient{},
			},
			wantErr: errors.InvalidModelErr,
		},
		"recipient weight must be greater than zero": {
			model: Revenue{
				Admin: addr,
				Recipients: []*Recipient{
					{Weight: 0, Address: addr},
				},
			},
			wantErr: errors.InvalidModelErr,
		},
		"recipient must have a valid address": {
			model: Revenue{
				Admin: addr,
				Recipients: []*Recipient{
					{Weight: 2, Address: []byte("zzz")},
				},
			},
			wantErr: errors.InvalidModelErr,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.model.Validate(); !errors.Is(tc.wantErr, err) {
				t.Logf("want %q", tc.wantErr)
				t.Logf("got %q", err)
				t.Fatal("unexpected validation result")
			}
		})
	}
}
