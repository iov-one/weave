package msgfee

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
)

func TestFeeDecorator(t *testing.T) {
	cases := map[string]struct {
		InitFees       []MsgFee
		Tx             weave.Tx
		Handler        weave.Handler
		WantCheckErr   error
		WantCheckFee   coin.Coin
		WantDeliverErr error
		WantDeliverFee coin.Coin
	}{
		"message fee with no previous fee": {
			InitFees: []MsgFee{
				{MsgPath: "foo/bar", Fee: coin.NewCoin(0, 1234, "DOGE")},
			},
			Handler:        &handlerMock{},
			Tx:             &txMock{msg: &msgMock{path: "foo/bar"}},
			WantCheckFee:   coin.NewCoin(0, 1234, "DOGE"),
			WantDeliverFee: coin.NewCoin(0, 1234, "DOGE"),
		},
		"message fee with addded to existing value with the same currency": {
			InitFees: []MsgFee{
				{MsgPath: "foo/bar", Fee: coin.NewCoin(0, 22, "BTC")},
			},
			Handler: &handlerMock{
				checkRes:   weave.CheckResult{RequiredFee: coin.NewCoin(1, 0, "BTC")},
				deliverRes: weave.DeliverResult{RequiredFee: coin.NewCoin(1, 0, "BTC")},
			},
			Tx:             &txMock{msg: &msgMock{path: "foo/bar"}},
			WantCheckFee:   coin.NewCoin(1, 22, "BTC"),
			WantDeliverFee: coin.NewCoin(1, 22, "BTC"),
		},
		"delivery failure": {
			InitFees: []MsgFee{
				{MsgPath: "foo/bar", Fee: coin.NewCoin(0, 1234, "DOGE")},
			},
			Handler: &handlerMock{
				deliverErr: errors.ErrUnauthorized,
			},
			Tx:             &txMock{msg: &msgMock{path: "foo/bar"}},
			WantCheckFee:   coin.NewCoin(0, 1234, "DOGE"),
			WantDeliverErr: errors.ErrUnauthorized,
			WantDeliverFee: coin.Coin{},
		},
		"check failure": {
			InitFees: []MsgFee{
				{MsgPath: "foo/bar", Fee: coin.NewCoin(0, 1234, "DOGE")},
			},
			Handler: &handlerMock{
				checkErr: errors.ErrUnauthorized,
			},
			Tx:             &txMock{msg: &msgMock{path: "foo/bar"}},
			WantCheckErr:   errors.ErrUnauthorized,
			WantDeliverFee: coin.NewCoin(0, 1234, "DOGE"),
		},
		"no fee for the transaction message": {
			InitFees:       []MsgFee{},
			Handler:        &handlerMock{},
			Tx:             &txMock{msg: &msgMock{path: "foo/bar"}},
			WantCheckFee:   coin.Coin{},
			WantDeliverFee: coin.Coin{},
		},
		"message fee with a different ticker than the existing fee": {
			InitFees: []MsgFee{
				{MsgPath: "foo/bar", Fee: coin.NewCoin(0, 1234, "DOGE")},
			},
			Handler: &handlerMock{
				checkRes:   weave.CheckResult{RequiredFee: coin.NewCoin(1, 0, "BTC")},
				deliverRes: weave.DeliverResult{RequiredFee: coin.NewCoin(1, 0, "BTC")},
			},
			Tx:             &txMock{msg: &msgMock{path: "foo/bar"}},
			WantCheckErr:   coin.ErrInvalidCurrency,
			WantCheckFee:   coin.NewCoin(1, 0, "BTC"),
			WantDeliverErr: coin.ErrInvalidCurrency,
			WantDeliverFee: coin.NewCoin(1, 0, "BTC"),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			decorator := NewFeeDecorator()
			bucket := NewMsgFeeBucket()
			db := store.MemStore()

			for _, f := range tc.InitFees {
				if i, err := bucket.Create(db, &f); err != nil {
					t.Fatalf("cannot create #%d transaction fee: %s", i, err)
				}
			}

			cres, err := decorator.Check(nil, db, tc.Tx, tc.Handler)
			if !errors.Is(tc.WantCheckErr, err) {
				t.Fatalf("check returned an unexpected error: %v", err)
			}
			if !tc.WantCheckFee.Equals(cres.RequiredFee) {
				t.Fatalf("unexpected check fee: %v", cres.RequiredFee)
			}

			dres, err := decorator.Deliver(nil, db, tc.Tx, tc.Handler)
			if !errors.Is(tc.WantDeliverErr, err) {
				t.Fatalf("deliver returned an unexpected error: %v", err)
			}
			if !tc.WantDeliverFee.Equals(dres.RequiredFee) {
				t.Fatalf("unexpected deliver fee: %v", dres.RequiredFee)
			}
		})
	}
}

var helpers x.TestHelpers

type txMock struct {
	weave.Tx

	msg weave.Msg
	err error
}

func (tx *txMock) GetMsg() (weave.Msg, error) {
	return tx.msg, tx.err
}

type msgMock struct {
	weave.Msg
	path string
}

func (m *msgMock) Path() string {
	return m.path
}

type handlerMock struct {
	checkRes weave.CheckResult
	checkErr error

	deliverRes weave.DeliverResult
	deliverErr error
}

var _ weave.Handler = (*handlerMock)(nil)

func (m *handlerMock) Check(weave.Context, weave.KVStore, weave.Tx) (weave.CheckResult, error) {
	return m.checkRes, m.checkErr
}

func (m *handlerMock) Deliver(weave.Context, weave.KVStore, weave.Tx) (weave.DeliverResult, error) {
	return m.deliverRes, m.deliverErr
}
