package aswap_test

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
	"github.com/iov-one/weave/x/aswap"
	"github.com/iov-one/weave/x/cash"
)

var (
	blockNow = time.Now()
)

func TestCreateHandler(t *testing.T) {
	alice := weavetest.NewCondition()
	bob := weavetest.NewCondition()
	pete := weavetest.NewCondition()

	swapAmount := coin.NewCoin(0, 1, "TEST")

	initialCoins, err := coin.CombineCoins(coin.NewCoin(1, 1, "TEST"))
	assert.Nil(t, err)

	bank := cash.NewBucket()
	ctrl := cash.NewController(bank)
	bucket := aswap.NewBucket()

	setBalance := func(t *testing.T, db weave.KVStore, addr weave.Address, coins coin.Coins) {
		acct, err := cash.WalletWith(addr, coins...)
		assert.Nil(t, err)
		err = bank.Save(db, acct)
		assert.Nil(t, err)
	}

	checkBalance := func(t *testing.T, db weave.KVStore, addr weave.Address) coin.Coins {
		acct, err := bank.Get(db, addr)
		assert.Nil(t, err)
		coins := cash.AsCoins(acct)
		return coins
	}

	r := app.NewRouter()
	authenticator := &weavetest.CtxAuth{"auth"}
	auth := x.ChainAuth(authenticator)
	aswap.RegisterRoutes(r, auth, ctrl)

	cases := map[string]struct {
		setup          func(ctx weave.Context, db weave.KVStore) weave.Context
		check          func(t *testing.T, db weave.KVStore)
		wantCheckErr   *errors.Error
		wantDeliverErr *errors.Error
		exp            aswap.Swap
		mutator        func(db *aswap.CreateSwapMsg)
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
				obj, err := bucket.Get(db, weavetest.SequenceID(1))
				assert.Nil(t, err)
				swap := aswap.AsSwap(obj)
				coins := checkBalance(t, db, aswap.SwapAddr(obj.Key(), swap))
				amt, err := coin.CombineCoins(swapAmount)
				assert.Nil(t, err)
				assert.Equal(t, coins.Equals(amt), true)
			},
		},
		"Invalid Msg": {
			wantDeliverErr: errors.ErrInvalidInput,
			wantCheckErr:   errors.ErrInvalidInput,
			mutator: func(msg *aswap.CreateSwapMsg) {
				msg.PreimageHash = nil
			},
		},
		"Invalid Timeout": {
			wantDeliverErr: errors.ErrInvalidInput,
			wantCheckErr:   errors.ErrInvalidInput,
			mutator: func(msg *aswap.CreateSwapMsg) {
				msg.Timeout = msg.Timeout.Add(-aswap.MinTimeout)
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
		createMsg := &aswap.CreateSwapMsg{
			Metadata:     &weave.Metadata{Schema: 1},
			Src:          alice.Address(),
			Recipient:    bob.Address(),
			PreimageHash: make([]byte, 32),
			Amount:       []*coin.Coin{&swapAmount},
			Timeout:      weave.AsUnixTime(time.Now()).Add(aswap.MinTimeout + time.Second),
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

			if _, err := r.Deliver(ctx, cache, tx); !spec.wantDeliverErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.wantDeliverErr, err)
			}
			if spec.check != nil {
				spec.check(t, cache)
			}

		})
	}

}
