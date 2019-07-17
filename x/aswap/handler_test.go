package aswap

import (
	"context"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
)

var (
	blockNow          = time.Now()
	defaultSequenceId = weavetest.SequenceID(1)
	alice             = weavetest.NewCondition()
	bob               = weavetest.NewCondition()
	pete              = weavetest.NewCondition()
	swapAmount        = coin.NewCoin(0, 1, "TEST")
	preimage          = make([]byte, 32)
	preimageHash      = HashBytes(preimage)

	bank   = cash.NewBucket()
	ctrl   = cash.NewController(bank)
	bucket = NewBucket()

	r             = app.NewRouter()
	authenticator = &weavetest.CtxAuth{Key: "auth"}
	auth          = x.ChainAuth(authenticator)
)

func init() {
	RegisterRoutes(r, auth, ctrl)
}

func TestCreateHandler(t *testing.T) {
	initialCoins, err := coin.CombineCoins(coin.NewCoin(1, 1, "TEST"))
	assert.Nil(t, err)

	cases := map[string]struct {
		setup          func(ctx weave.Context, db weave.KVStore) weave.Context
		check          func(t *testing.T, db weave.KVStore)
		wantCheckErr   *errors.Error
		wantDeliverErr *errors.Error
		exp            Swap
		mutator        func(db *CreateMsg)
	}{
		"Happy Path": {
			setup: func(ctx weave.Context, db weave.KVStore) weave.Context {
				setBalance(t, db, alice.Address(), initialCoins)
				return authenticator.SetConditions(ctx, alice)
			},

			wantDeliverErr: nil,
			wantCheckErr:   nil,
			mutator:        nil,
			check: func(t *testing.T, db weave.KVStore) {
				var swap Swap
				err := bucket.One(db, defaultSequenceId, &swap)
				assert.Nil(t, err)
				coins := checkBalance(t, db, swap.Address)
				amt, err := coin.CombineCoins(swapAmount)
				assert.Nil(t, err)
				assert.Equal(t, true, coins.Equals(amt))
			},
		},
		"happy path, timeout can be in the past": {
			setup: func(ctx weave.Context, db weave.KVStore) weave.Context {
				setBalance(t, db, alice.Address(), initialCoins)
				return authenticator.SetConditions(ctx, alice)
			},
			mutator: func(msg *CreateMsg) {
				msg.Timeout = weave.AsUnixTime(time.Now().Add(-1000 * time.Hour))
			},
		},
		"Invalid Msg": {
			wantDeliverErr: errors.ErrInput,
			wantCheckErr:   errors.ErrInput,
			mutator: func(msg *CreateMsg) {
				msg.PreimageHash = nil
			},
		},
		"Invalid Auth": {
			setup: func(ctx weave.Context, db weave.KVStore) weave.Context {
				return authenticator.SetConditions(ctx, pete)
			},
			wantDeliverErr: errors.ErrUnauthorized,
			wantCheckErr:   errors.ErrUnauthorized,
		},
		"Empty account": {
			setup: func(ctx weave.Context, db weave.KVStore) weave.Context {
				return authenticator.SetConditions(ctx, alice)
			},
			wantDeliverErr: errors.ErrEmpty,
			wantCheckErr:   nil,
		},
	}

	for name, spec := range cases {
		createMsg := &CreateMsg{
			Metadata:     &weave.Metadata{Schema: 1},
			Source:       alice.Address(),
			Destination:  bob.Address(),
			PreimageHash: preimageHash,
			Amount:       []*coin.Coin{&swapAmount},
			Timeout:      weave.AsUnixTime(time.Now()),
		}
		t.Run(name, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "aswap", "cash")

			ctx := weave.WithHeight(context.Background(), 500)
			ctx = weave.WithBlockTime(ctx, blockNow)
			if spec.setup != nil {
				ctx = spec.setup(ctx, db)
			}
			if spec.mutator != nil {
				spec.mutator(createMsg)
			}
			cache := db.CacheWrap()

			tx := &weavetest.Tx{Msg: createMsg}
			if _, err := r.Check(ctx, cache, tx); !spec.wantCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.wantCheckErr, err)
			}

			cache.Discard()

			res, err := r.Deliver(ctx, cache, tx)
			if !spec.wantDeliverErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.wantDeliverErr, err)
			}

			if res != nil {
				err := bucket.Has(cache, res.Data)
				assert.Nil(t, err)
			}

			if spec.check != nil {
				spec.check(t, cache)
			}

		})
	}

}

func TestReleaseHandler(t *testing.T) {
	initialCoins, err := coin.CombineCoins(coin.NewCoin(1, 1, "TEST"))
	assert.Nil(t, err)

	cases := map[string]struct {
		setup          func(ctx weave.Context, db weave.KVStore) weave.Context
		check          func(t *testing.T, db weave.KVStore)
		wantCheckErr   *errors.Error
		wantDeliverErr *errors.Error
		exp            Swap
		mutator        func(db *ReleaseMsg)
	}{
		"Happy Path, includes no auth check": {
			wantDeliverErr: nil,
			wantCheckErr:   nil,
			mutator:        nil,
			check: func(t *testing.T, db weave.KVStore) {
				err := bucket.Has(db, defaultSequenceId)
				assert.IsErr(t, errors.ErrNotFound, err)
				coins := checkBalance(t, db, bob.Address())
				amt, err := coin.CombineCoins(swapAmount)
				assert.Nil(t, err)
				assert.Equal(t, true, coins.Equals(amt))
			},
		},
		"Invalid Msg": {
			wantDeliverErr: errors.ErrInput,
			wantCheckErr:   errors.ErrInput,
			mutator: func(msg *ReleaseMsg) {
				msg.Preimage = nil
			},
		},
		"Invalid SwapID": {
			wantDeliverErr: errors.ErrNotFound,
			wantCheckErr:   errors.ErrNotFound,
			mutator: func(msg *ReleaseMsg) {
				msg.SwapID = weavetest.SequenceID(2)
			},
		},
		"Invalid Preimage": {
			wantDeliverErr: errors.ErrUnauthorized,
			wantCheckErr:   errors.ErrUnauthorized,
			mutator: func(msg *ReleaseMsg) {
				msg.Preimage = make([]byte, 32)
				msg.Preimage[0] = 1
			},
		},
		"Expired": {
			setup: func(ctx weave.Context, db weave.KVStore) weave.Context {
				return weave.WithBlockTime(ctx, time.Now().Add(10*time.Hour))
			},
			wantDeliverErr: errors.ErrState,
			wantCheckErr:   errors.ErrState,
		},
	}

	for name, spec := range cases {
		createMsg := &CreateMsg{
			Metadata:     &weave.Metadata{Schema: 1},
			Source:       alice.Address(),
			Destination:  bob.Address(),
			PreimageHash: preimageHash,
			Amount:       []*coin.Coin{&swapAmount},
			Timeout:      weave.AsUnixTime(time.Now().Add(time.Hour)),
		}
		t.Run(name, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "aswap", "cash")

			ctx := weave.WithHeight(context.Background(), 500)
			ctx = weave.WithBlockTime(ctx, blockNow)
			// setup a swap
			createCtx := authenticator.SetConditions(ctx, alice)
			setBalance(t, db, alice.Address(), initialCoins)
			tx := &weavetest.Tx{Msg: createMsg}
			_, err = r.Deliver(createCtx, db, tx)
			assert.Nil(t, err)

			releaseMsg := &ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				SwapID:   defaultSequenceId,
				Preimage: preimage,
			}

			if spec.setup != nil {
				ctx = spec.setup(ctx, db)
			}
			if spec.mutator != nil {
				spec.mutator(releaseMsg)
			}
			cache := db.CacheWrap()

			tx = &weavetest.Tx{Msg: releaseMsg}
			if _, err := r.Check(ctx, cache, tx); !spec.wantCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.wantCheckErr, err)
			}

			cache.Discard()

			if _, err := r.Deliver(ctx, cache, tx); !spec.wantDeliverErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.wantDeliverErr, err)
			}
			if spec.check != nil {
				spec.check(t, cache)
			}

		})
	}

}

func TestReturnHandler(t *testing.T) {
	initialCoins, err := coin.CombineCoins(swapAmount)
	assert.Nil(t, err)

	cases := map[string]struct {
		setup          func(ctx weave.Context, db weave.KVStore) weave.Context
		check          func(t *testing.T, db weave.KVStore)
		wantCheckErr   *errors.Error
		wantDeliverErr *errors.Error
		exp            Swap
		mutator        func(db *ReturnMsg)
	}{
		"Happy Path, includes no auth check": {
			setup: func(ctx weave.Context, db weave.KVStore) weave.Context {
				return weave.WithBlockTime(ctx, blockNow.Add(2*time.Hour))
			},
			wantDeliverErr: nil,
			wantCheckErr:   nil,
			mutator:        nil,
			check: func(t *testing.T, db weave.KVStore) {
				err := bucket.Has(db, defaultSequenceId)
				assert.IsErr(t, errors.ErrNotFound, err)
				coins := checkBalance(t, db, alice.Address())
				amt, err := coin.CombineCoins(swapAmount)
				assert.Nil(t, err)
				assert.Equal(t, true, coins.Equals(amt))
			},
		},
		"Invalid Msg": {
			wantDeliverErr: errors.ErrInput,
			wantCheckErr:   errors.ErrInput,
			mutator: func(msg *ReturnMsg) {
				msg.SwapID = nil
			},
		},
		"Invalid SwapID": {
			wantDeliverErr: errors.ErrNotFound,
			wantCheckErr:   errors.ErrNotFound,
			mutator: func(msg *ReturnMsg) {
				msg.SwapID = weavetest.SequenceID(2)
			},
		},
		"Not Expired": {
			setup: func(ctx weave.Context, db weave.KVStore) weave.Context {
				return weave.WithBlockTime(ctx, blockNow)
			},
			wantDeliverErr: errors.ErrState,
			wantCheckErr:   errors.ErrState,
		},
	}

	for name, spec := range cases {
		createMsg := &CreateMsg{
			Metadata:     &weave.Metadata{Schema: 1},
			Source:       alice.Address(),
			Destination:  bob.Address(),
			PreimageHash: preimageHash,
			Amount:       []*coin.Coin{&swapAmount},
			Timeout:      weave.AsUnixTime(blockNow.Add(time.Hour)),
		}
		t.Run(name, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "aswap", "cash")

			ctx := weave.WithHeight(context.Background(), 500)
			ctx = weave.WithBlockTime(ctx, blockNow)
			// setup a swap
			createCtx := authenticator.SetConditions(ctx, alice)
			setBalance(t, db, alice.Address(), initialCoins)
			tx := &weavetest.Tx{Msg: createMsg}
			_, err = r.Deliver(createCtx, db, tx)
			assert.Nil(t, err)

			returnMsg := &ReturnMsg{
				Metadata: &weave.Metadata{Schema: 1},
				SwapID:   defaultSequenceId,
			}

			if spec.setup != nil {
				ctx = spec.setup(ctx, db)
			}
			if spec.mutator != nil {
				spec.mutator(returnMsg)
			}
			cache := db.CacheWrap()

			tx = &weavetest.Tx{Msg: returnMsg}
			if _, err := r.Check(ctx, cache, tx); !spec.wantCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.wantCheckErr, err)
			}

			cache.Discard()

			if _, err := r.Deliver(ctx, cache, tx); !spec.wantDeliverErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.wantDeliverErr, err)
			}
			if spec.check != nil {
				spec.check(t, cache)
			}

		})
	}

}

func setBalance(t testing.TB, db weave.KVStore, addr weave.Address, coins coin.Coins) {
	t.Helper()

	acct, err := cash.WalletWith(addr, coins...)
	assert.Nil(t, err)
	err = bank.Save(db, acct)
	assert.Nil(t, err)
}

func checkBalance(t testing.TB, db weave.KVStore, addr weave.Address) coin.Coins {
	t.Helper()

	acct, err := bank.Get(db, addr)
	assert.Nil(t, err)
	coins := cash.AsCoins(acct)
	return coins
}
