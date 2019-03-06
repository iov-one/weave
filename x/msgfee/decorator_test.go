package msgfee

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
)

func TestFeeDecorator(t *testing.T) {
	handler := helpers.CountingHandler()
	decorator := NewFeeDecorator()
	bucket := NewMsgFeeBucket()
	db := store.MemStore()

	_, err := bucket.Create(db, &MsgFee{
		MsgPath: "foo/bar",
		Fee:     coin.NewCoinp(0, 1234, "DOGE"),
	})
	if err != nil {
		t.Fatalf("cannot create a transaction fee: %s", err)
	}

	tx := &txMock{
		msg: &msgMock{path: "foo/bar"},
	}
	if res, err := decorator.Check(nil, db, tx, handler); err != nil {
		t.Fatalf("check failed: %s", err)
	} else if !res.RequiredFee.Equals(coin.NewCoin(0, 1234, "DOGE")) {
		t.Fatalf("unexpected check fee: %v", res.RequiredFee)
	}

	if c := handler.GetCount(); c != 1 {
		t.Fatalf("want count=1, got %d", c)
	}

	if res, err := decorator.Deliver(nil, db, tx, handler); err != nil {
		t.Fatalf("check failed: %s", err)
	} else if !res.RequiredFee.Equals(coin.NewCoin(0, 1234, "DOGE")) {
		t.Fatalf("unexpected deliver fee: %v", res.RequiredFee)
	}

	if c := handler.GetCount(); c != 2 {
		t.Fatalf("want count=2, got %d", c)
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
