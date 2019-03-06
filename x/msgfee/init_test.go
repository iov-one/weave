package msgfee

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/store"
)

func TestGenesis(t *testing.T) {
	const genesis = `
{
	"msgfee": [
		{
			"msg_path": "foo/bar",
			"fee": {"whole": 1, "fractional": 2, "ticker": "DOGE"}
		},
		{
			"msg_path": "a/b",
			"fee": {"whole": 2, "fractional": 0, "ticker": "ETH"}
		}
	]
}
	`
	var opts weave.Options
	if err := json.Unmarshal([]byte(genesis), &opts); err != nil {
		t.Fatalf("cannot unmarshal genesis: %s", err)
	}

	db := store.MemStore()
	var ini Initializer
	if err := ini.FromGenesis(opts, db); err != nil {
		t.Fatalf("cannot load genesis: %s", err)
	}

	bucket := NewMsgFeeBucket()
	fee, err := bucket.MessageFee(db, "foo/bar")
	if err != nil {
		t.Fatalf("cannot fetch fee: %s", err)
	}
	if !fee.Equals(coin.NewCoin(1, 2, "DOGE")) {
		t.Fatalf("got an unexpected fee value: %s", fee)
	}

	fee, err = bucket.MessageFee(db, "a/b")
	if err != nil {
		t.Fatalf("cannot fetch fee: %s", err)
	}
	if !fee.Equals(coin.NewCoin(2, 0, "ETH")) {
		t.Fatalf("got an unexpected fee value: %s", fee)
	}
}
