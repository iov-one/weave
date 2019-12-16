package account

import (
	"testing"

	weave "github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestAccountMsgFeeDecorator(t *testing.T) {
	cases := map[string]struct {
		ReqFee         coin.Coin
		Tx             weave.Tx
		WantCheckErr   *errors.Error
		WantDeliverErr *errors.Error
		WantCheckFee   coin.Coin
	}{
		"register fee charged from first domain": {
			ReqFee: coin.NewCoin(3, 0, "IOV"),
			Tx: &weavetest.Tx{
				Msg: &RegisterAccountMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Domain:   "first",
					Name:     "myaccount",
				},
			},
			WantCheckFee: coin.NewCoin(4, 0, "IOV"),
		},
		"register fee charged from second domain": {
			ReqFee: coin.NewCoin(3, 0, "IOV"),
			Tx: &weavetest.Tx{
				Msg: &RegisterAccountMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Domain:   "second",
					Name:     "myaccount",
				},
			},
			WantCheckFee: coin.NewCoin(6, 0, "IOV"),
		},
		"operation free of charge": {
			ReqFee: coin.NewCoin(3, 0, "IOV"),
			Tx: &weavetest.Tx{
				Msg: &TransferAccountMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Domain:   "second",
					Name:     "myaccount",
					NewOwner: weavetest.NewCondition().Address(),
				},
			},
			WantCheckFee: coin.NewCoin(3, 0, "IOV"),
		},
		"mixed fee tickers": {
			ReqFee: coin.NewCoin(3, 0, "DOGE"),
			Tx: &weavetest.Tx{
				Msg: &RegisterAccountMsg{
					Metadata: &weave.Metadata{Schema: 1},
					Domain:   "second",
					Name:     "myaccount",
				},
			},
			WantCheckErr:   errors.ErrCurrency,
			WantDeliverErr: errors.ErrCurrency,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "account")

			fdomain := Domain{
				Metadata: &weave.Metadata{Schema: 1},
				Domain:   "first",
				Admin:    weavetest.NewCondition().Address(),
				MsgFees: []AccountMsgFee{
					{MsgPath: "account/register_account", Fee: coin.NewCoin(1, 0, "IOV")},
					{MsgPath: "account/delete_account", Fee: coin.NewCoin(2, 0, "IOV")},
				},
			}
			if _, err := NewDomainBucket().Put(db, []byte(fdomain.Domain), &fdomain); err != nil {
				t.Fatalf("cannot put domain: %s", err)
			}

			sdomain := Domain{
				Metadata: &weave.Metadata{Schema: 1},
				Domain:   "second",
				Admin:    weavetest.NewCondition().Address(),
				MsgFees: []AccountMsgFee{
					{MsgPath: "account/register_account", Fee: coin.NewCoin(3, 0, "IOV")},
					{MsgPath: "account/delete_account", Fee: coin.NewCoin(7, 0, "IOV")},
				},
			}
			if _, err := NewDomainBucket().Put(db, []byte(sdomain.Domain), &sdomain); err != nil {
				t.Fatalf("cannot put domain: %s", err)
			}

			decorator := NewAccountMsgFeeDecorator()

			handler := &weavetest.Handler{
				CheckResult:   weave.CheckResult{RequiredFee: tc.ReqFee},
				DeliverResult: weave.DeliverResult{RequiredFee: tc.ReqFee},
			}

			cres, err := decorator.Check(nil, db, tc.Tx, handler)
			if !tc.WantCheckErr.Is(err) {
				t.Fatalf("check returned an unexpected error: %v", err)
			}
			if tc.WantCheckErr == nil && !tc.WantCheckFee.Equals(cres.RequiredFee) {
				t.Fatalf("unexpected check fee: %v", cres.RequiredFee)
			}

			if _, err := decorator.Deliver(nil, db, tc.Tx, handler); !tc.WantDeliverErr.Is(err) {
				t.Fatalf("deliver returned an unexpected error: %v", err)
			}

		})
	}
}
