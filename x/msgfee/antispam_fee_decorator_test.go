package msgfee_test

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/msgfee"
)

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
			decorator := msgfee.NewAntispamFeeDecorator(tc.AntiSpamFee)
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
