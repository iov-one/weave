package msgfee

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/migration"
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
	],
	"conf": {
		"msgfee": {
			"fee_admin": "seq:foo/bar/1"
		}
	}
}
	`
	var opts weave.Options
	if err := json.Unmarshal([]byte(genesis), &opts); err != nil {
		t.Fatalf("cannot unmarshal genesis: %s", err)
	}

	db := store.MemStore()
	migration.MustInitPkg(db, "msgfee")
	var ini Initializer
	if err := ini.FromGenesis(opts, weave.GenesisParams{}, db); err != nil {
		t.Fatalf("cannot load genesis: %s", err)
	}

	bucket := NewMsgFeeBucket()
	var fee MsgFee
	if err := bucket.One(db, []byte("foo/bar"), &fee); err != nil {
		t.Fatalf("cannot fetch fee: %s", err)
	}
	if !fee.Fee.Equals(coin.NewCoin(1, 2, "DOGE")) {
		t.Fatalf("got an unexpected fee value: %s", fee)
	}

	if err := bucket.One(db, []byte("a/b"), &fee); err != nil {
		t.Fatalf("cannot fetch fee: %s", err)
		t.Fatalf("cannot fetch fee: %s", err)
	}
	if !fee.Fee.Equals(coin.NewCoin(2, 0, "ETH")) {
		t.Fatalf("got an unexpected fee value: %s", fee)
	}
}

func TestGenesisWithInvalidFee(t *testing.T) {
	cases := map[string]string{
		"zero fee":  `[{"msg_path": "foo/bar", "fee": {"whole": 0, "fractional": 0, "ticker": "DOGE"}}]`,
		"no ticker": `[{"msg_path": "foo/bar", "fee": {"whole": 1, "fractional": 0, "ticker": ""}}]`,
		"no path":   `[{"fee": {"whole": 1, "fractional": 1, "ticker": "DOGE"}}]`,
		"no fee":    `[{"msg_path": "foo/bar"}]`,
	}
	for testName, content := range cases {
		t.Run(testName, func(t *testing.T) {
			genesis := `{"msgfee": ` + content + `}`
			var opts weave.Options
			if err := json.Unmarshal([]byte(genesis), &opts); err != nil {
				t.Fatalf("cannot unmarshal genesis: %s", err)
			}

			db := store.MemStore()
			migration.MustInitPkg(db, "msgfee")
			var ini Initializer
			if err := ini.FromGenesis(opts, weave.GenesisParams{}, db); err == nil {
				t.Fatal("no error")
			}
		})
	}

}
