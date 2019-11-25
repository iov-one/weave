package datamigration

import (
	"context"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestHandler(t *testing.T) {
	var aliceCond = weavetest.NewCondition()

	cases := map[string]struct {
		Conds   []weave.Condition
		Tx      weave.Tx
		WantErr *errors.Error
	}{
		"successful migration": {
			Conds: []weave.Condition{aliceCond},
			Tx: &weavetest.Tx{
				Msg: &ExecuteMigrationMsg{
					Metadata:    &weave.Metadata{Schema: 1},
					MigrationID: "first migration",
				},
			},
			WantErr: nil,
		},
		"missing auth": {
			Conds: []weave.Condition{},
			Tx: &weavetest.Tx{
				Msg: &ExecuteMigrationMsg{
					Metadata:    &weave.Metadata{Schema: 1},
					MigrationID: "first migration",
				},
			},
			WantErr: errors.ErrUnauthorized,
		},
		"missing auth and wrong chain": {
			Conds: []weave.Condition{},
			Tx: &weavetest.Tx{
				Msg: &ExecuteMigrationMsg{
					Metadata:    &weave.Metadata{Schema: 1},
					MigrationID: "second migration",
				},
			},
			WantErr: errors.ErrUnauthorized,
		},
		"wrong chain migration": {
			Conds: []weave.Condition{aliceCond},
			Tx: &weavetest.Tx{
				Msg: &ExecuteMigrationMsg{
					Metadata:    &weave.Metadata{Schema: 1},
					MigrationID: "second migration",
				},
			},
			WantErr: errors.ErrChain,
		},
		"executed migration cannot be executed again": {
			Conds: []weave.Condition{aliceCond},
			Tx: &weavetest.Tx{
				Msg: &ExecuteMigrationMsg{
					Metadata:    &weave.Metadata{Schema: 1},
					MigrationID: "zero migration",
				},
			},
			WantErr: errors.ErrState,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			defer withNewRegister()()

			db := store.MemStore()
			migration.MustInitPkg(db, "datamigration")

			MustRegister(
				"zero migration",
				Migration{
					ChainID:         "testchain",
					RequiredSigners: []weave.Address{aliceCond.Address()},
					Migrate:         func(context.Context, weave.KVStore) error { panic("never to be called") },
				},
			)

			MustRegister(
				"first migration",
				Migration{
					ChainID:         "testchain",
					RequiredSigners: []weave.Address{aliceCond.Address()},
					Migrate:         func(context.Context, weave.KVStore) error { return nil },
				},
			)
			MustRegister(
				"second migration",
				Migration{
					ChainID:         "staging",
					RequiredSigners: []weave.Address{aliceCond.Address()},
					Migrate:         func(context.Context, weave.KVStore) error { return nil },
				},
			)
			MustRegister(
				"third migration",
				Migration{
					ChainID:         "testchain",
					RequiredSigners: []weave.Address{aliceCond.Address()},
					Migrate:         func(context.Context, weave.KVStore) error { return errors.ErrHuman },
				},
			)

			rt := app.NewRouter()
			auth := &weavetest.CtxAuth{Key: "auth"}
			RegisterRoutes(rt, auth)

			ctx := weave.WithHeight(context.Background(), 100)
			ctx = weave.WithChainID(ctx, "testchain")
			ctx = auth.SetConditions(ctx, tc.Conds...)
			ctx = weave.WithBlockTime(ctx, time.Now())

			b := NewExecutedMigrationBucket()
			if _, err := b.Put(db, []byte("zero migration"), &ExecutedMigration{Metadata: &weave.Metadata{}}); err != nil {
				t.Fatalf("cannot register migration execution: %s", err)
			}

			cache := db.CacheWrap()
			if _, err := rt.Check(ctx, cache, tc.Tx); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected check error: %s", err)
			}
			cache.Discard()
			if _, err := rt.Deliver(ctx, db, tc.Tx); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected deliver error: %s", err)
			}
		})
	}
}

// withNewRegister is a test helper that modifies the reference of the global
// initialization register. To ensure that each test is running using a custom
// register, overwrite the global register reference with an empty instance.
//
// It is the callers responsibility to call the returned cleanup function in
// order to set back the original register reference.
//
// This function is not safe for concurrent use.
func withNewRegister() func() {
	old := reg
	reg = newRegister()
	return func() { reg = old }
}
