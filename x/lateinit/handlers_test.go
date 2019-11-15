package lateinit

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
	"github.com/iov-one/weave/weavetest/assert"
)

func TestHandlers(t *testing.T) {
	var (
		aliceCond   = weavetest.NewCondition()
		bobCond     = weavetest.NewCondition()
		charlieCond = weavetest.NewCondition()
	)

	cases := map[string]struct {
		Conditions  []weave.Condition
		Tx          weave.Tx
		BlockHeight int64
		WantErr     *errors.Error
		AfterTest   func(t *testing.T, db weave.KVStore)
	}{
		"an entity can be initialized": {
			Conditions: []weave.Condition{aliceCond},
			Tx: &weavetest.Tx{
				Msg: &ExecuteInitMsg{
					Metadata: &weave.Metadata{Schema: 1},
					InitID:   "init-init",
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				b := NewExecutedInitBucket()
				var e ExecutedInit
				if err := b.One(db, []byte("init-init"), &e); err != nil {
					t.Fatalf("cannot get an entity: %s", err)
				}
				if got, want := e.InitID, "init-init"; got != want {
					t.Fatalf("wan %q, got %q", want, got)
				}
			},
		},
		"correct authentication is required": {
			Conditions: []weave.Condition{bobCond},
			Tx: &weavetest.Tx{
				Msg: &ExecuteInitMsg{
					Metadata: &weave.Metadata{Schema: 1},
					InitID:   "init-init",
				},
			},
			WantErr: errors.ErrUnauthorized,
		},
		"existing entity cannot be initialized or modified": {
			Conditions: []weave.Condition{charlieCond},
			Tx: &weavetest.Tx{
				Msg: &ExecuteInitMsg{
					Metadata: &weave.Metadata{Schema: 1},
					InitID:   "existing-init",
				},
			},
			WantErr: errors.ErrState,
		},
		"migration for a different chain cannot be executed": {
			Conditions: []weave.Condition{charlieCond},
			Tx: &weavetest.Tx{
				Msg: &ExecuteInitMsg{
					Metadata: &weave.Metadata{Schema: 1},
					InitID:   "staging-init",
				},
			},
			WantErr: errors.ErrChain,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			defer withNewRegister()()

			db := store.MemStore()
			migration.MustInitPkg(db, "lateinit")

			b := NewExecutedInitBucket()

			MustRegister(
				"init-init",
				"testchain-123",
				aliceCond.Address(),
				[]byte("init-init"),
				b,
				&ExecutedInit{
					Metadata: &weave.Metadata{Schema: 1},
					InitID:   "init-init",
				},
			)

			_, err := b.Put(db, []byte("existing-init"), &ExecutedInit{
				Metadata: &weave.Metadata{Schema: 1},
				InitID:   "existing-init",
			})
			if err != nil {
				t.Fatal(err)
			}

			MustRegister(
				"existing-init",
				"testchain-123",
				charlieCond.Address(),
				[]byte("existing-init"),
				b,
				&ExecutedInit{
					Metadata: &weave.Metadata{Schema: 1},
					InitID:   "existing-init",
				},
			)

			MustRegister(
				"staging-init",
				"stagingchain-942",
				charlieCond.Address(),
				[]byte("staging-init"),
				b,
				&ExecutedInit{
					Metadata: &weave.Metadata{Schema: 1},
					InitID:   "staging-init",
				},
			)

			rt := app.NewRouter()
			auth := &weavetest.CtxAuth{Key: "auth"}
			RegisterRoutes(rt, auth)

			ctx := weave.WithHeight(context.Background(), tc.BlockHeight)
			ctx = weave.WithChainID(ctx, "testchain-123")
			ctx = auth.SetConditions(ctx, tc.Conditions...)
			ctx = weave.WithBlockTime(ctx, time.Now())

			cache := db.CacheWrap()
			if _, err := rt.Check(ctx, cache, tc.Tx); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected check error: want %q, got %+v", tc.WantErr, err)
			}
			cache.Discard()
			if _, err := rt.Deliver(ctx, db, tc.Tx); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected deliver error: want %q, got %+v", tc.WantErr, err)
			}

			if tc.AfterTest != nil {
				tc.AfterTest(t, db)
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

func TestOnlyValidEntityCanBeRegistered(t *testing.T) {
	err := reg.Register(
		"an id",
		"testchain-123",
		nil,
		[]byte("123456789"),
		NewExecutedInitBucket(),
		&ExecutedInit{})

	assert.FieldError(t, err, "Metadata", errors.ErrMetadata)
	assert.FieldError(t, err, "InitID", errors.ErrEmpty)
}
