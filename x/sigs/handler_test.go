package sigs

import (
	"context"
	"math"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestBumpSequence(t *testing.T) {
	var (
		key1 = weavetest.NewKey().PublicKey()
		key2 = weavetest.NewKey().PublicKey()
	)

	cases := map[string]struct {
		// Before performing the test, initialize the database with given user data.
		InitData       []*UserData
		Msg            BumpSequenceMsg
		Signers        []weave.Condition
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		// WantSequence sequence values should be tested for being one
		// smaller than expected. This is usual transaction processing
		// will additionally increment sequence. That is why handler
		// increments it by the requested value - 1.
		WantSequences []*UserData
	}{
		"great success": {
			InitData: []*UserData{
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key1, Sequence: 1},
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key2, Sequence: 9},
			},
			Signers: []weave.Condition{key1.Condition()},
			Msg:     BumpSequenceMsg{Metadata: &weave.Metadata{Schema: 1}, Increment: 2},
			WantSequences: []*UserData{
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key1, Sequence: 2},
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key2, Sequence: 9},
			},
		},
		"incrementing sequence of the main signer": {
			InitData: []*UserData{
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key1, Sequence: 1},
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key2, Sequence: 9},
			},
			Signers: []weave.Condition{
				key2.Condition(), // Main signer.
				key1.Condition(),
			},
			Msg: BumpSequenceMsg{Metadata: &weave.Metadata{Schema: 1}, Increment: 2},
			WantSequences: []*UserData{
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key1, Sequence: 1},
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key2, Sequence: 10},
			},
		},
		"transaction with a missing signature is rejected": {
			Msg:            BumpSequenceMsg{Metadata: &weave.Metadata{Schema: 1}, Increment: 1},
			Signers:        nil,
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"message with a zero sequence increment is invalid": {
			InitData: []*UserData{
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key1, Sequence: 1},
			},
			Msg:            BumpSequenceMsg{Metadata: &weave.Metadata{Schema: 1}, Increment: 0},
			WantCheckErr:   errors.ErrMsg,
			WantDeliverErr: errors.ErrMsg,
		},
		"user that we increment the sequence of must exist": {
			InitData: []*UserData{
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key2, Sequence: 4},
			},
			Signers:        []weave.Condition{key1.Condition()},
			Msg:            BumpSequenceMsg{Metadata: &weave.Metadata{Schema: 1}, Increment: 421},
			WantCheckErr:   errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
		},
		"sequence increment value must not be greater than 1000": {
			InitData: []*UserData{
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key1, Sequence: 4},
			},
			Signers:        []weave.Condition{key1.Condition()},
			Msg:            BumpSequenceMsg{Metadata: &weave.Metadata{Schema: 1}, Increment: 1001},
			WantCheckErr:   errors.ErrMsg,
			WantDeliverErr: errors.ErrMsg,
		},
		"sequence increment value can be 1000": {
			InitData: []*UserData{
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key1, Sequence: 4},
			},
			Signers: []weave.Condition{key1.Condition()},
			Msg:     BumpSequenceMsg{Metadata: &weave.Metadata{Schema: 1}, Increment: 1000},
			WantSequences: []*UserData{
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key1, Sequence: 1003},
			},
		},
		"successful sequence increment before counter overflow": {
			InitData: []*UserData{
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key1, Sequence: math.MaxInt64 - 20},
			},
			Signers: []weave.Condition{key1.Condition()},
			Msg:     BumpSequenceMsg{Metadata: &weave.Metadata{Schema: 1}, Increment: 20},
			WantSequences: []*UserData{
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key1, Sequence: math.MaxInt64 - 1},
			},
		},
		"sequence increment value overflow": {
			InitData: []*UserData{
				{Metadata: &weave.Metadata{Schema: 1}, Pubkey: key1, Sequence: math.MaxInt64 - 20},
			},
			Signers:        []weave.Condition{key1.Condition()},
			Msg:            BumpSequenceMsg{Metadata: &weave.Metadata{Schema: 1}, Increment: 21},
			WantCheckErr:   errors.ErrOverflow,
			WantDeliverErr: errors.ErrOverflow,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			bucket := NewBucket()
			db := store.MemStore()
			migration.MustInitPkg(db, "sigs")

			for i, data := range tc.InitData {
				obj := orm.NewSimpleObj(data.Pubkey.Address(), data)
				if err := bucket.Save(db, obj); err != nil {
					t.Fatalf("cannot save %d user: %s", i, err)
				}
			}

			auth := &weavetest.CtxAuth{Key: "auth"}
			handler := bumpSequenceHandler{
				b:    bucket,
				auth: auth,
			}
			ctx := context.Background()
			ctx = auth.SetConditions(ctx, tc.Signers...)
			tx := weavetest.Tx{Msg: &tc.Msg}

			cache := db.CacheWrap()
			if _, err := handler.Check(ctx, cache, &tx); !tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected check error: %+v", err)
			}
			cache.Discard()

			if _, err := handler.Deliver(ctx, db, &tx); !tc.WantDeliverErr.Is(err) {
				t.Fatalf("unexpected check error: %+v", err)
			}
			if tc.WantDeliverErr != nil {
				// If we expect an error than it make no sense to continue the flow.
				return
			}

			for i, want := range tc.WantSequences {
				obj, err := bucket.Get(db, want.Pubkey.Address())
				if err != nil {
					t.Errorf("cannot get %d user: %s", i, err)
				}
				if obj == nil {
					t.Errorf("cannot get %d user: not found", i)
				} else if got := AsUser(obj); got.Sequence != want.Sequence {
					t.Errorf("unexpected %d sequence: want %d, got %d", i, want.Sequence, got.Sequence)
				}

			}
		})
	}
}
