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
				Destinations: []*Destination{
					{Weight: 1, Address: addr},
				},
				Address: addr,
			},
			wantErr: nil,
		},
		"revenue address is required": {
			model: Revenue{
				Metadata: &weave.Metadata{Schema: 1},
				Admin:    addr,
				Destinations: []*Destination{
					{Weight: 1, Address: addr},
				},
				Address: nil,
			},
			wantErr: errors.ErrEmpty,
		},
		"admin address must be present": {
			model: Revenue{
				Metadata: &weave.Metadata{Schema: 1},
				Admin:    nil,
				Destinations: []*Destination{
					{Weight: 1, Address: addr},
				},
				Address: addr,
			},
			wantErr: errors.ErrEmpty,
		},
		"at least one destination must be given": {
			model: Revenue{
				Metadata:     &weave.Metadata{Schema: 1},
				Admin:        addr,
				Destinations: []*Destination{},
				Address:      addr,
			},
			wantErr: errors.ErrModel,
		},
		"destination weight must be greater than zero": {
			model: Revenue{
				Metadata: &weave.Metadata{Schema: 1},
				Admin:    addr,
				Destinations: []*Destination{
					{Weight: 0, Address: addr},
				},
				Address: addr,
			},
			wantErr: errors.ErrModel,
		},
		"destination must have a valid address": {
			model: Revenue{
				Metadata: &weave.Metadata{Schema: 1},
				Admin:    addr,
				Destinations: []*Destination{
					{Weight: 2, Address: []byte("zzz")},
				},
				Address: addr,
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

func TestValidDestinations(t *testing.T) {
	cases := map[string]struct {
		destinations []*Destination
		baseErr      *errors.Error
		want         *errors.Error
	}{
		"all good": {
			destinations: []*Destination{
				{Address: weave.Address("f427d624ed29c1fae0e2"), Weight: 1},
				{Address: weave.Address("aa27d624ed29c1fae0e2"), Weight: 2},
			},
			baseErr: errors.ErrModel,
			want:    nil,
		},
		"destination address not unique": {
			destinations: []*Destination{
				{Address: weave.Address("f427d624ed29c1fae0e2"), Weight: 1},
				{Address: weave.Address("f427d624ed29c1fae0e2"), Weight: 1},
			},
			baseErr: errors.ErrMsg,
			want:    errors.ErrMsg,
		},
		"too many destinations": {
			destinations: createDestinations(maxDestinations + 1),
			baseErr:      errors.ErrModel,
			want:         errors.ErrModel,
		},
		"weight too big": {
			destinations: []*Destination{
				{Address: weave.Address("f427d624ed29c1fae0e2"), Weight: math.MaxInt32 - 1},
			},
			baseErr: errors.ErrMsg,
			want:    errors.ErrMsg,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := validateDestinations(tc.destinations, tc.baseErr); !tc.want.Is(err) {
				t.Fatalf("%+v", err)
			}
		})
	}
}

func createDestinations(amount int) []*Destination {
	rs := make([]*Destination, amount)
	for i := range rs {
		rs[i] = &Destination{
			Address: weavetest.SequenceID(uint64(i)),
			Weight:  int32(i%100 + 1),
		}
	}
	return rs
}
