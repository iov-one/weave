package gov

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestMerger(t *testing.T) {

	alice := weavetest.NewCondition().Address()
	bobby := weavetest.NewCondition().Address()
	charlie := weavetest.NewCondition().Address()

	specs := map[string]struct {
		diff           []Elector
		expValidateErr *errors.Error
		expElectors    []Elector
		expTotalWeight uint64
	}{

		"Add elector": {
			diff:           []Elector{{Address: charlie, Weight: 2}},
			expElectors:    []Elector{{alice, 1}, {bobby, 10}, {Address: charlie, Weight: 2}},
			expTotalWeight: 13,
		},
		"Remove elector": {
			diff:           []Elector{{Address: bobby, Weight: 0}},
			expElectors:    []Elector{{alice, 1}},
			expTotalWeight: 1,
		},
		"Update elector": {
			diff:           []Elector{{Address: alice, Weight: 2}},
			expElectors:    []Elector{{alice, 2}, {bobby, 10}},
			expTotalWeight: 12,
		},
		"Reject duplicates in diff": {
			diff:           []Elector{{Address: alice, Weight: 2}, {Address: alice, Weight: 3}},
			expValidateErr: errors.ErrDuplicate,
		},
		"Reject existing values": {
			diff:           []Elector{{Address: alice, Weight: 1}},
			expValidateErr: errors.ErrDuplicate,
		},
		"Reject no existing values": {
			diff:           []Elector{{Address: charlie, Weight: 0}},
			expValidateErr: errors.ErrNotFound,
		},
	}

	source := []Elector{{alice, 1}, {bobby, 10}}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			m := newMerger(source)
			// when merge
			if err := m.merge(spec.diff); !spec.expValidateErr.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
			}
			if spec.expValidateErr != nil {
				return
			}
			// then
			gotElectors, gotTotalWeight := m.serialize()
			sortByAddress(spec.expElectors)
			if exp, got := spec.expElectors, gotElectors; !reflect.DeepEqual(got, exp) {
				t.Errorf("expected %v but got %v", exp, got)
			}
			if exp, got := spec.expTotalWeight, gotTotalWeight; exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}

}
