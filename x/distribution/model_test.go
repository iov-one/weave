package distribution

import (
	"math"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestRevenueValidate(t *testing.T) {
	addr := weave.Address("f427d624ed29c1fae0e2")

	cases := map[string]struct {
		model   Revenue
		wantErr *errors.Error
	}{
		"valid model": {
			model: Revenue{
				Metadata: &weave.Metadata{Schema: 1},
				Admin:    addr,
				Recipients: []*Recipient{
					{Weight: 1, Address: addr},
				},
			},
			wantErr: nil,
		},
		"admin address must be present": {
			model: Revenue{
				Metadata: &weave.Metadata{Schema: 1},
				Admin:    nil,
				Recipients: []*Recipient{
					{Weight: 1, Address: addr},
				},
			},
			wantErr: errors.ErrEmpty,
		},
		"at least one recipient must be given": {
			model: Revenue{
				Metadata:   &weave.Metadata{Schema: 1},
				Admin:      addr,
				Recipients: []*Recipient{},
			},
			wantErr: errors.ErrModel,
		},
		"recipient weight must be greater than zero": {
			model: Revenue{
				Metadata: &weave.Metadata{Schema: 1},
				Admin:    addr,
				Recipients: []*Recipient{
					{Weight: 0, Address: addr},
				},
			},
			wantErr: errors.ErrModel,
		},
		"recipient must have a valid address": {
			model: Revenue{
				Metadata: &weave.Metadata{Schema: 1},
				Admin:    addr,
				Recipients: []*Recipient{
					{Weight: 2, Address: []byte("zzz")},
				},
			},
			wantErr: errors.ErrInput,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.model.Validate(); !tc.wantErr.Is(err) {
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
		baseErr    *errors.Error
		want       *errors.Error
	}{
		"all good": {
			recipients: []*Recipient{
				{Address: weave.Address("f427d624ed29c1fae0e2"), Weight: 1},
				{Address: weave.Address("aa27d624ed29c1fae0e2"), Weight: 2},
			},
			baseErr: errors.ErrModel,
			want:    nil,
		},
		"recipient address not unique": {
			recipients: []*Recipient{
				{Address: weave.Address("f427d624ed29c1fae0e2"), Weight: 1},
				{Address: weave.Address("f427d624ed29c1fae0e2"), Weight: 1},
			},
			baseErr: errors.ErrMsg,
			want:    errors.ErrMsg,
		},
		"too many recipients": {
			recipients: createRecipients(maxRecipients + 1),
			baseErr:    errors.ErrModel,
			want:       errors.ErrModel,
		},
		"weight too big": {
			recipients: []*Recipient{
				{Address: weave.Address("f427d624ed29c1fae0e2"), Weight: math.MaxInt32 - 1},
			},
			baseErr: errors.ErrMsg,
			want:    errors.ErrMsg,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := validateRecipients(tc.recipients, tc.baseErr); !tc.want.Is(err) {
				t.Fatalf("%+v", err)
			}
		})
	}
}

func createRecipients(amount int) []*Recipient {
	rs := make([]*Recipient, amount)
	for i := range rs {
		rs[i] = &Recipient{
			Address: weavetest.SequenceID(uint64(i)),
			Weight:  int32(i%100 + 1),
		}
	}
	return rs
}
