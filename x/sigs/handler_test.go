package sigs

import (
	"context"
	"testing"

	"github.com/iov-one/weave/errors"
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
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		WantSequences  []*UserData
	}{
		"great success": {
			InitData: []*UserData{
				{Pubkey: key1, Sequence: 1},
				{Pubkey: key2, Sequence: 9},
			},
			Msg: BumpSequenceMsg{
				Pubkey:    key1,
				Increment: 1,
			},
			WantSequences: []*UserData{
				{Pubkey: key1, Sequence: 2},
				{Pubkey: key2, Sequence: 9},
			},
		},
		"message with a missing public key is invalid": {
			Msg:          BumpSequenceMsg{Pubkey: nil, Increment: 1},
			WantCheckErr: errors.ErrInvalidMsg,
		},
		"message with a negative sequence increment is invalid": {
			InitData: []*UserData{
				{Pubkey: key1, Sequence: 1},
			},
			Msg:          BumpSequenceMsg{Pubkey: key1, Increment: -1},
			WantCheckErr: errors.ErrInvalidMsg,
		},
		"message with a zero sequence increment is invalid": {
			InitData: []*UserData{
				{Pubkey: key1, Sequence: 1},
			},
			Msg:          BumpSequenceMsg{Pubkey: key1, Increment: 0},
			WantCheckErr: errors.ErrInvalidMsg,
		},
		"user that we increment the sequence of must exist": {
			InitData:     nil,
			Msg:          BumpSequenceMsg{Pubkey: key1, Increment: 421},
			WantCheckErr: errors.ErrNotFound,
		},
		"sequence increment value must not be greater than 1000": {
			InitData: []*UserData{
				{Pubkey: key1, Sequence: 4},
			},
			Msg:          BumpSequenceMsg{Pubkey: key1, Increment: 1001},
			WantCheckErr: errors.ErrInvalidMsg,
		},
		"sequence increment value can be 1000": {
			InitData: []*UserData{
				{Pubkey: key1, Sequence: 4},
			},
			Msg: BumpSequenceMsg{Pubkey: key1, Increment: 1000},
			WantSequences: []*UserData{
				{Pubkey: key1, Sequence: 1004},
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			bucket := NewBucket()
			db := store.MemStore()

			for i, data := range tc.InitData {
				obj := orm.NewSimpleObj(data.Pubkey.Address(), data)
				if err := bucket.Save(db, obj); err != nil {
					t.Fatalf("cannot save %d user: %s", i, err)
				}
			}

			h := bumpSequenceHandler{b: bucket}
			ctx := context.Background()
			tx := weavetest.Tx{Msg: &tc.Msg}

			cache := db.CacheWrap()
			if _, err := h.Check(ctx, cache, &tx); !tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected check error: %+v", err)
			}
			cache.Discard()

			if tc.WantCheckErr != nil {
				// If we expect an error than it makes no sense to continue the flow.
				return
			}

			if _, err := h.Deliver(ctx, db, &tx); !tc.WantCheckErr.Is(err) {
				t.Fatalf("unexpected check error: %+v", err)
			}
			if tc.WantDeliverErr == nil {
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
