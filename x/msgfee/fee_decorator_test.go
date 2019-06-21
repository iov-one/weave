package msgfee

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestFeeDecorator(t *testing.T) {
	cases := map[string]struct {
		InitFees       []MsgFee
		Tx             weave.Tx
		Handler        weave.Handler
		WantCheckErr   *errors.Error
		WantCheckFee   coin.Coin
		WantDeliverErr *errors.Error
		WantDeliverFee coin.Coin
	}{
		"message fee with no previous fee": {
			InitFees: []MsgFee{
				{
					Metadata: &weave.Metadata{Schema: 1},
					MsgPath:  "foo/bar",
					Fee:      coin.NewCoin(0, 1234, "DOGE"),
				},
			},
			Handler:        &weavetest.Handler{},
			Tx:             &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckFee:   coin.NewCoin(0, 1234, "DOGE"),
			WantDeliverFee: coin.NewCoin(0, 1234, "DOGE"),
		},
		"message fee added to an existing value with the same currency": {
			InitFees: []MsgFee{
				{
					Metadata: &weave.Metadata{Schema: 1},
					MsgPath:  "foo/bar",
					Fee:      coin.NewCoin(0, 22, "BTC"),
				},
			},
			Handler: &weavetest.Handler{
				CheckResult:   weave.CheckResult{RequiredFee: coin.NewCoin(1, 0, "BTC")},
				DeliverResult: weave.DeliverResult{RequiredFee: coin.NewCoin(1, 0, "BTC")},
			},
			Tx:             &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckFee:   coin.NewCoin(1, 22, "BTC"),
			WantDeliverFee: coin.NewCoin(1, 22, "BTC"),
		},
		"delivery failure": {
			InitFees: []MsgFee{
				{
					Metadata: &weave.Metadata{Schema: 1},
					MsgPath:  "foo/bar",
					Fee:      coin.NewCoin(0, 1234, "DOGE"),
				},
			},
			Handler: &weavetest.Handler{
				DeliverErr: errors.ErrUnauthorized,
			},
			Tx:             &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckFee:   coin.NewCoin(0, 1234, "DOGE"),
			WantDeliverErr: errors.ErrUnauthorized,
			WantDeliverFee: coin.Coin{},
		},
		"check failure": {
			InitFees: []MsgFee{
				{
					Metadata: &weave.Metadata{Schema: 1},
					MsgPath:  "foo/bar",
					Fee:      coin.NewCoin(0, 1234, "DOGE"),
				},
			},
			Handler: &weavetest.Handler{
				CheckErr: errors.ErrUnauthorized,
			},
			Tx:             &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverFee: coin.NewCoin(0, 1234, "DOGE"),
		},
		"no fee for the transaction message": {
			InitFees:       []MsgFee{},
			Handler:        &weavetest.Handler{},
			Tx:             &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckFee:   coin.Coin{},
			WantDeliverFee: coin.Coin{},
		},
		"message fee with a different ticker than the existing fee": {
			InitFees: []MsgFee{
				{
					Metadata: &weave.Metadata{Schema: 1},
					MsgPath:  "foo/bar",
					Fee:      coin.NewCoin(0, 1234, "DOGE"),
				},
			},
			Handler: &weavetest.Handler{
				CheckResult:   weave.CheckResult{RequiredFee: coin.NewCoin(1, 0, "BTC")},
				DeliverResult: weave.DeliverResult{RequiredFee: coin.NewCoin(1, 0, "BTC")},
			},
			Tx:             &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckErr:   errors.ErrCurrency,
			WantCheckFee:   coin.NewCoin(1, 0, "BTC"),
			WantDeliverErr: errors.ErrCurrency,
			WantDeliverFee: coin.NewCoin(1, 0, "BTC"),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			decorator := NewFeeDecorator()
			bucket := NewMsgFeeBucket()
			db := store.MemStore(179)
			migration.MustInitPkg(db, "msgfee")

			for _, f := range tc.InitFees {
				if i, err := bucket.Create(db, &f); err != nil {
					t.Fatalf("cannot create #%d transaction fee: %s", i, err)
				}
			}

			cres, err := decorator.Check(nil, db, tc.Tx, tc.Handler)
			if !tc.WantCheckErr.Is(err) {
				t.Fatalf("check returned an unexpected error: %v", err)
			}
			if tc.WantCheckErr == nil && !tc.WantCheckFee.Equals(cres.RequiredFee) {
				t.Fatalf("unexpected check fee: %v", cres.RequiredFee)
			}

			dres, err := decorator.Deliver(nil, db, tc.Tx, tc.Handler)
			if !tc.WantDeliverErr.Is(err) {
				t.Fatalf("deliver returned an unexpected error: %v", err)
			}
			if tc.WantDeliverErr == nil && !tc.WantDeliverFee.Equals(dres.RequiredFee) {
				t.Fatalf("unexpected deliver fee: %v", dres.RequiredFee)
			}
		})
	}
}
