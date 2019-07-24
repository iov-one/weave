package msgfee

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestNewAntispamFeeDecoratorZero(t *testing.T) {
	d := NewAntispamFeeDecorator(coin.Coin{})
	if d != nil {
		t.Fatalf("zero fee must return a nil decorator: %v", d)
	}
}

func TestNewAntispamFeeDecorator(t *testing.T) {
	cases := map[string]struct {
		ReqFee         coin.Coin
		AntiSpamFee    coin.Coin
		Tx             weave.Tx
		Handler        *weavetest.Handler
		CheckErr       *errors.Error
		DeliverErr     *errors.Error
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		WantCheckFee   coin.Coin
	}{
		"anti-spam fee is less than initial fee": {
			ReqFee:       coin.NewCoin(0, 1234, "DOGE"),
			Handler:      &weavetest.Handler{},
			Tx:           &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckFee: coin.NewCoin(0, 1234, "DOGE"),
			AntiSpamFee:  coin.NewCoin(0, 1233, "DOGE"),
		},
		"anti-spam fee is equal to initial fee": {
			ReqFee:       coin.NewCoin(0, 1234, "DOGE"),
			Handler:      &weavetest.Handler{},
			Tx:           &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckFee: coin.NewCoin(0, 1234, "DOGE"),
			AntiSpamFee:  coin.NewCoin(0, 1234, "DOGE"),
		},
		"anti-spam fee is more than initial fee": {
			ReqFee:       coin.NewCoin(0, 1234, "DOGE"),
			Handler:      &weavetest.Handler{},
			Tx:           &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckFee: coin.NewCoin(0, 1235, "DOGE"),
			AntiSpamFee:  coin.NewCoin(0, 1235, "DOGE"),
		},
		"anti-spam fee is zero": {
			ReqFee:       coin.NewCoin(0, 1234, "DOGE"),
			Handler:      &weavetest.Handler{},
			Tx:           &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckFee: coin.NewCoin(0, 1234, "DOGE"),
			AntiSpamFee:  coin.NewCoin(0, 0, "DOGE"),
		},
		"anti-spam fee is zero with different currencies": {
			ReqFee:       coin.NewCoin(0, 1234, "DOGE"),
			Handler:      &weavetest.Handler{},
			Tx:           &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckFee: coin.NewCoin(0, 1234, "DOGE"),
			AntiSpamFee:  coin.NewCoin(0, 0, "GATO"),
		},
		"anti-spam has different currency": {
			ReqFee:       coin.NewCoin(0, 1234, "DOGE"),
			Handler:      &weavetest.Handler{},
			Tx:           &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckFee: coin.NewCoin(0, 1234, "DOGE"),
			AntiSpamFee:  coin.NewCoin(0, 1235, "GATO"),
			WantCheckErr: errors.ErrCurrency,
		},
		"deliver err propagates": {
			ReqFee:         coin.NewCoin(0, 1234, "DOGE"),
			Handler:        &weavetest.Handler{},
			Tx:             &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckFee:   coin.NewCoin(0, 1234, "DOGE"),
			AntiSpamFee:    coin.NewCoin(0, 1235, "GATO"),
			WantCheckErr:   errors.ErrCurrency,
			DeliverErr:     errors.ErrNotFound,
			WantDeliverErr: errors.ErrNotFound,
		},
		"check err propagates": {
			ReqFee:       coin.NewCoin(0, 1234, "DOGE"),
			Handler:      &weavetest.Handler{},
			Tx:           &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foo/bar"}},
			WantCheckFee: coin.NewCoin(0, 1234, "DOGE"),
			AntiSpamFee:  coin.NewCoin(0, 1235, "GATO"),
			CheckErr:     errors.ErrNotFound,
			WantCheckErr: errors.ErrNotFound,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			decorator := NewAntispamFeeDecorator(tc.AntiSpamFee)
			tc.Handler.CheckResult = weave.CheckResult{
				RequiredFee: tc.ReqFee,
			}

			if tc.DeliverErr != nil {
				tc.Handler.DeliverErr = tc.DeliverErr
			}
			if tc.CheckErr != nil {
				tc.Handler.CheckErr = tc.CheckErr
			}

			cres, err := decorator.Check(nil, nil, tc.Tx, tc.Handler)
			if !tc.WantCheckErr.Is(err) {
				t.Fatalf("check returned an unexpected error: %v", err)
			}
			if tc.WantCheckErr == nil && !tc.WantCheckFee.Equals(cres.RequiredFee) {
				t.Fatalf("unexpected check fee: %v", cres.RequiredFee)
			}

			if _, err := decorator.Deliver(nil, nil, tc.Tx, tc.Handler); !tc.WantDeliverErr.Is(err) {
				t.Fatalf("deliver returned an unexpected error: %v", err)
			}

		})
	}
}

func BenchmarkAntispamFeeDecorator(b *testing.B) {
	cases := map[string]struct {
		Next    weave.Handler
		Fee     coin.Coin
		WantFee coin.Coin
		WantErr *errors.Error
	}{
		"zero fee with nil decorator": {
			Next: &weavetest.Handler{
				CheckResult: weave.CheckResult{RequiredFee: coin.NewCoin(0, 0, "")},
			},
			WantErr: nil,
			WantFee: coin.NewCoin(0, 0, ""),
		},
		"zero fee with when a fee is required": {
			Fee: coin.NewCoin(2, 4, "IOV"),
			Next: &weavetest.Handler{
				CheckResult: weave.CheckResult{RequiredFee: coin.NewCoin(0, 0, "")},
			},
			WantErr: errors.ErrEmpty,
		},
		"sufficient non zero fee": {
			Fee: coin.NewCoin(2, 4, "IOV"),
			Next: &weavetest.Handler{
				CheckResult: weave.CheckResult{RequiredFee: coin.NewCoin(5, 6, "IOV")},
			},
			WantFee: coin.NewCoin(5, 6, "IOV"),
			WantErr: nil,
		},
		"coin type differs": {
			Fee: coin.NewCoin(2, 4, "IOV"),
			Next: &weavetest.Handler{
				CheckResult: weave.CheckResult{RequiredFee: coin.NewCoin(1, 1, "DOGE")},
			},
			WantErr: errors.ErrCurrency,
		},
	}

	for benchName, bc := range cases {
		db := store.MemStore()
		ctx := context.Background()

		decorator := NewAntispamFeeDecorator(bc.Fee)

		b.Run(benchName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				res, err := decorator.Check(ctx, db, &weavetest.Tx{}, bc.Next)
				if !bc.WantErr.Is(err) {
					b.Fatalf("unexpected error: %s", err)
				}
				if err == nil && !res.RequiredFee.Equals(bc.WantFee) {
					b.Fatalf("invalid decorator fee: %s", res.RequiredFee)
				}
			}

			// Deliver decoraror is pass through.
		})
	}
}
