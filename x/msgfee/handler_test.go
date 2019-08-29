package msgfee

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestSetMsgFeeHandler(t *testing.T) {
	rt := app.NewRouter()
	auth := &weavetest.CtxAuth{Key: "auth"}
	admin := weavetest.NewCondition()
	RegisterRoutes(rt, auth)

	cases := map[string]struct {
		Ctx            func() context.Context
		Tx             weave.Tx
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		WantFees       map[string]coin.Coin
	}{
		"register a new fee": {
			Ctx: func() context.Context {
				return context.WithValue(context.Background(), "auth", []weave.Condition{admin})
			},
			Tx: &weavetest.Tx{
				Msg: &SetMsgFeeMsg{
					Metadata: &weave.Metadata{Schema: 1},
					MsgPath:  "test/myfee",
					Fee:      coin.NewCoin(3, 2, "DOGE"),
				},
			},
			WantFees: map[string]coin.Coin{
				"test/one":   coin.NewCoin(1, 0, "IOV"),
				"test/myfee": coin.NewCoin(3, 2, "DOGE"),
			},
		},
		"only an admin can change the fee": {
			Ctx: func() context.Context {
				// No authentication information attached.
				return context.Background()
			},
			Tx: &weavetest.Tx{
				Msg: &SetMsgFeeMsg{
					Metadata: &weave.Metadata{Schema: 1},
					MsgPath:  "test/myfee",
					Fee:      coin.NewCoin(3, 2, "DOGE"),
				},
			},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverErr: errors.ErrUnauthorized,
		},
		"overwrite an existing fee with a new value": {
			Ctx: func() context.Context {
				return context.WithValue(context.Background(), "auth", []weave.Condition{admin})
			},
			Tx: &weavetest.Tx{
				Msg: &SetMsgFeeMsg{
					Metadata: &weave.Metadata{Schema: 1},
					MsgPath:  "test/one",
					Fee:      coin.NewCoin(3, 8, "DOGE"),
				},
			},
			WantFees: map[string]coin.Coin{
				"test/one": coin.NewCoin(3, 8, "DOGE"),
			},
		},
		"delete and existing fee": {
			Ctx: func() context.Context {
				return context.WithValue(context.Background(), "auth", []weave.Condition{admin})
			},
			Tx: &weavetest.Tx{
				Msg: &SetMsgFeeMsg{
					Metadata: &weave.Metadata{Schema: 1},
					MsgPath:  "test/one",
					Fee:      coin.NewCoin(0, 0, "XYZ"),
				},
			},
			WantFees: map[string]coin.Coin{
				"test/one": coin.NewCoin(0, 0, ""),
			},
		},
		"a fee must be non negative": {
			Ctx: func() context.Context {
				return context.WithValue(context.Background(), "auth", []weave.Condition{admin})
			},
			Tx: &weavetest.Tx{
				Msg: &SetMsgFeeMsg{
					Metadata: &weave.Metadata{Schema: 1},
					MsgPath:  "test/one",
					Fee:      coin.NewCoin(-10, 0, "IOV"),
				},
			},
			WantCheckErr:   errors.ErrAmount,
			WantDeliverErr: errors.ErrAmount,
		},
		"message path must be provided": {
			Ctx: func() context.Context {
				return context.WithValue(context.Background(), "auth", []weave.Condition{admin})
			},
			Tx: &weavetest.Tx{
				Msg: &SetMsgFeeMsg{
					Metadata: &weave.Metadata{Schema: 1},
					MsgPath:  "",
					Fee:      coin.NewCoin(10, 9421, "IOV"),
				},
			},
			WantCheckErr:   errors.ErrEmpty,
			WantDeliverErr: errors.ErrEmpty,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			migration.MustInitPkg(db, "msgfee")

			conf := Configuration{
				Owner:    nil,
				FeeAdmin: admin.Address(),
			}
			if err := gconf.Save(db, "msgfee", &conf); err != nil {
				t.Fatalf("cannot save gconf configuration: %s", err)
			}

			fees := NewMsgFeeBucket()

			// Initialize the database with a single fee.
			_, err := fees.Put(db, []byte("test/one"), &MsgFee{
				Metadata: &weave.Metadata{Schema: 1},
				MsgPath:  "test/mymsg",
				Fee:      coin.NewCoin(1, 0, "IOV"),
			})
			assert.Nil(t, err)

			cache := db.CacheWrap()
			_, err = rt.Check(tc.Ctx(), cache, tc.Tx)
			assert.IsErr(t, tc.WantCheckErr, err)
			cache.Discard()

			_, err = rt.Deliver(tc.Ctx(), db, tc.Tx)
			assert.IsErr(t, tc.WantDeliverErr, err)

			for path, amount := range tc.WantFees {
				// If expected amount is zero, we expect the fee to not be present in the store.
				if amount.IsZero() {
					var fee MsgFee
					if err := fees.One(db, []byte(path), &fee); !errors.ErrNotFound.Is(err) {
						t.Fatalf("fee for %q message was not expected, got %q", path, fee)
					}
				} else {
					var fee MsgFee
					if err := fees.One(db, []byte(path), &fee); err != nil {
						t.Fatalf("cannot fetch %q fee: %s", path, err)
					}
					if !fee.Fee.Equals(amount) {
						t.Errorf("expecetd %q message fee to be %q, got %q", path, amount, fee.Fee)
					}
				}
			}
		})
	}
}
