package multisig

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestCreateContractHandler(t *testing.T) {
	alice := weavetest.NewCondition().Address()
	bobby := weavetest.NewCondition().Address()
	cindy := weavetest.NewCondition().Address()

	cases := map[string]struct {
		Msg            weave.Msg
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
	}{
		"successfully create a contract": {
			Msg: &CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Participants: []*Participant{
					{Weight: 1, Signature: alice},
					{Weight: 2, Signature: bobby},
					{Weight: 3, Signature: cindy},
				},
				ActivationThreshold: 2,
				AdminThreshold:      3,
			},
		},
		"cannot create a contract without participants": {
			Msg: &CreateMsg{
				Metadata:            &weave.Metadata{Schema: 1},
				Participants:        []*Participant{},
				ActivationThreshold: 2,
				AdminThreshold:      3,
			},
			WantCheckErr: errors.ErrMsg,
		},
		"cannot create if activation threshold is too high": {
			Msg: &CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Participants: []*Participant{
					{Weight: 1, Signature: alice},
					{Weight: 2, Signature: bobby},
					{Weight: 3, Signature: cindy},
				},
				ActivationThreshold: 7, // higher than total
				AdminThreshold:      3,
			},
			WantCheckErr: errors.ErrMsg,
		},
		"can create if admin threshold is higher than total participants power": {
			Msg: &CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Participants: []*Participant{
					{Weight: 1, Signature: alice},
					{Weight: 2, Signature: bobby},
					{Weight: 3, Signature: cindy},
				},
				ActivationThreshold: 2,
				AdminThreshold:      maxWeightValue,
			},
		},
		"cannot create if activation threshold is higher than admin threshold": {
			Msg: &CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Participants: []*Participant{
					{Weight: 2, Signature: alice},
					{Weight: 2, Signature: bobby},
				},
				ActivationThreshold: 2,
				AdminThreshold:      1,
			},
			WantCheckErr: errors.ErrMsg,
		},
	}

	auth := &weavetest.Auth{
		Signer: weavetest.NewCondition(), // Any signer will do.
	}
	rt := app.NewRouter()
	RegisterRoutes(rt, auth)

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "multisig")
			ctx := context.Background()
			tx := &weavetest.Tx{Msg: tc.Msg}

			cache := db.CacheWrap()
			if _, err := rt.Check(ctx, cache, tx); !tc.WantCheckErr.Is(err) {
				t.Logf("want: %+v", tc.WantCheckErr)
				t.Logf(" got: %+v", err)
				t.Fatalf("check (%T)", tc.Msg)
			}
			cache.Discard()
			if tc.WantCheckErr != nil {
				// Failed checks are causing the message to be ignored.
				return
			}

			if _, err := rt.Deliver(ctx, db, tx); !tc.WantDeliverErr.Is(err) {
				t.Logf("want: %+v", tc.WantDeliverErr)
				t.Logf(" got: %+v", err)
				t.Fatalf("delivery (%T)", tc.Msg)
			}
		})
	}
}

func TestUpdateContractHandler(t *testing.T) {
	aliceCond := weavetest.NewCondition()
	alice := aliceCond.Address()
	bobbyCond := weavetest.NewCondition()
	bobby := bobbyCond.Address()
	cindyCond := weavetest.NewCondition()
	cindy := cindyCond.Address()

	cases := map[string]struct {
		Msg            weave.Msg
		Conditions     []weave.Condition
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
	}{
		"successfully update a contract": {
			Conditions: []weave.Condition{
				cindyCond,
			},
			Msg: &UpdateMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ContractID: weavetest.SequenceID(1),
				Participants: []*Participant{
					{Weight: 1, Signature: alice},
					{Weight: 2, Signature: bobby},
					{Weight: 3, Signature: cindy},
				},
				ActivationThreshold: 2,
				AdminThreshold:      3,
			},
		},
		"successfully update a contract with combined signature power": {
			Conditions: []weave.Condition{
				// Together they provide power 3 which is
				// enough to run admin functionalities.
				aliceCond,
				bobbyCond,
			},
			Msg: &UpdateMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ContractID: weavetest.SequenceID(1),
				Participants: []*Participant{
					{Weight: 1, Signature: alice},
					{Weight: 2, Signature: bobby},
					{Weight: 3, Signature: cindy},
				},
				ActivationThreshold: 1,
				AdminThreshold:      1,
			},
		},
		"cannot update a contract without participants": {
			Conditions: []weave.Condition{
				cindyCond,
			},
			Msg: &UpdateMsg{
				Metadata:            &weave.Metadata{Schema: 1},
				ContractID:          weavetest.SequenceID(1),
				Participants:        []*Participant{},
				ActivationThreshold: 2,
				AdminThreshold:      3,
			},
			WantCheckErr: errors.ErrMsg,
		},
		"cannot update if activation threshold is too high": {
			Conditions: []weave.Condition{
				cindyCond,
			},
			Msg: &UpdateMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ContractID: weavetest.SequenceID(1),
				Participants: []*Participant{
					{Weight: 1, Signature: alice},
					{Weight: 2, Signature: bobby},
					{Weight: 3, Signature: cindy},
				},
				ActivationThreshold: 100,
				AdminThreshold:      3,
			},
			WantCheckErr: errors.ErrMsg,
		},
		"can update if admin threshold is higher than total participants power": {
			Conditions: []weave.Condition{
				cindyCond,
			},
			Msg: &UpdateMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ContractID: weavetest.SequenceID(1),
				Participants: []*Participant{
					{Weight: 1, Signature: alice},
					{Weight: 2, Signature: bobby},
					{Weight: 3, Signature: cindy},
				},
				ActivationThreshold: 1,
				AdminThreshold:      maxWeightValue,
			},
		},
		"cannot update if activation threshold is higher than admin threshold": {
			Conditions: []weave.Condition{
				cindyCond,
			},
			Msg: &UpdateMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ContractID: weavetest.SequenceID(1),
				Participants: []*Participant{
					{Weight: 2, Signature: alice},
					{Weight: 2, Signature: bobby},
				},
				ActivationThreshold: 2,
				AdminThreshold:      1,
			},
			WantCheckErr: errors.ErrMsg,
		},
		"admin power is required to update a contract": {
			Conditions: []weave.Condition{
				// Bobby is only power 2 and power 3 is required.
				bobbyCond,
			},
			Msg: &UpdateMsg{
				Metadata:   &weave.Metadata{Schema: 1},
				ContractID: weavetest.SequenceID(1),
				Participants: []*Participant{
					{Weight: 1, Signature: alice},
					{Weight: 2, Signature: bobby},
					{Weight: 3, Signature: cindy},
				},
				ActivationThreshold: 2,
				AdminThreshold:      2,
			},
			WantCheckErr: errors.ErrUnauthorized,
		},
	}

	auth := &weavetest.CtxAuth{Key: "auth"}
	rt := app.NewRouter()
	RegisterRoutes(rt, auth)

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "multisig")

			ctx := context.Background()
			ctx = auth.SetConditions(ctx, tc.Conditions...)
			tx := &weavetest.Tx{Msg: tc.Msg}

			b := NewContractBucket()

			key, err := contractSeq.NextVal(db)
			if err != nil {
				t.Fatalf("cannot acquire ID: %s", err)
			}
			_, err = b.Put(db, key, &Contract{
				Metadata: &weave.Metadata{Schema: 1},
				Participants: []*Participant{
					{Weight: 1, Signature: alice},
					{Weight: 2, Signature: bobby},
					{Weight: 3, Signature: cindy},
				},
				ActivationThreshold: 2,
				AdminThreshold:      3,
				Address:             MultiSigCondition(key).Address(),
			})
			assert.Nil(t, err)

			cache := db.CacheWrap()
			if _, err := rt.Check(ctx, cache, tx); !tc.WantCheckErr.Is(err) {
				t.Logf("want: %+v", tc.WantCheckErr)
				t.Logf(" got: %+v", err)
				t.Fatalf("check (%T)", tc.Msg)
			}
			cache.Discard()
			if tc.WantCheckErr != nil {
				// Failed checks are causing the message to be ignored.
				return
			}

			if _, err := rt.Deliver(ctx, db, tx); !tc.WantDeliverErr.Is(err) {
				t.Logf("want: %+v", tc.WantDeliverErr)
				t.Logf(" got: %+v", err)
				t.Fatalf("delivery (%T)", tc.Msg)
			}
		})
	}
}
