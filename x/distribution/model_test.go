package distribution

import (
	"encoding/binary"
	"math"
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
			wantErr: errors.ErrInvalidModel,
		},
		"at least one recipient must be given": {
			model: Revenue{
				Admin:      addr,
				Recipients: []*Recipient{},
			},
			wantErr: errors.ErrInvalidModel,
		},
		"recipient weight must be greater than zero": {
			model: Revenue{
				Admin: addr,
				Recipients: []*Recipient{
					{Weight: 0, Address: addr},
				},
			},
			wantErr: errors.ErrInvalidModel,
		},
		"recipient must have a valid address": {
			model: Revenue{
				Admin: addr,
				Recipients: []*Recipient{
					{Weight: 2, Address: []byte("zzz")},
				},
			},
			wantErr: errors.ErrInvalidModel,
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

func TestValidRecipients(t *testing.T) {
	cases := map[string]struct {
		recipients []*Recipient
		baseErr    errors.Error
		want       error
	}{
		"all good": {
			recipients: []*Recipient{
				{Address: weave.Address("f427d624ed29c1fae0e2"), Weight: 1},
				{Address: weave.Address("aa27d624ed29c1fae0e2"), Weight: 2},
			},
			baseErr: errors.ErrInvalidModel,
			want:    nil,
		},
		"recipient address not unique": {
			recipients: []*Recipient{
				{Address: weave.Address("f427d624ed29c1fae0e2"), Weight: 1},
				{Address: weave.Address("f427d624ed29c1fae0e2"), Weight: 1},
			},
			baseErr: errors.ErrInvalidMsg,
			want:    errors.ErrInvalidMsg,
		},
		"too many recipients": {
			recipients: createRecipients(maxRecipients + 1),
			baseErr:    errors.ErrInvalidModel,
			want:       errors.ErrInvalidModel,
		},
		"weight too big": {
			recipients: []*Recipient{
				{Address: weave.Address("f427d624ed29c1fae0e2"), Weight: math.MaxInt32 - 1},
			},
			baseErr: errors.ErrInvalidMsg,
			want:    errors.ErrInvalidMsg,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := validateRecipients(tc.recipients, tc.baseErr); !errors.Is(tc.want, err) {
				t.Fatalf("%+v", err)
			}
		})
	}
}

func createRecipients(amount int) []*Recipient {
	rs := make([]*Recipient, amount)
	addr := make([]byte, 8)
	for i := range rs {
		binary.BigEndian.PutUint64(addr, uint64(i))
		rs[i] = &Recipient{Address: addr, Weight: int32(i%100 + 1)}
	}
	return rs
}
