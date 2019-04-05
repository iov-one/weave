package gconf

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/store"
)

func TestGenesisInitializer(t *testing.T) {
	const genesis = `
		{
			"gconf": {
				"a-string": "hello",
				"an-int": 321,
				"string-coin": "4 IOV",
				"struct-coin": {"whole": 7, "ticker": "IOV"},
				"an-address": "d2a1f84143a9754057e42db6d6c9f986fe0ff673"
			}
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

	if got := String(db, "a-string"); got != "hello" {
		t.Fatalf("unexpected string: %q", got)
	}

	if got := Int(db, "an-int"); got != 321 {
		t.Fatalf("unexpected int: %v", got)
	}

	wantAddr := hexDecode(t, "d2a1f84143a9754057e42db6d6c9f986fe0ff673")
	if got := Address(db, "an-address"); !wantAddr.Equals(got) {
		t.Fatalf("unexpected address: %v", got)
	}

	if got := Coin(db, "string-coin"); !coin.NewCoin(4, 0, "IOV").Equals(got) {
		t.Fatalf("unexpected coin: %v", got)
	}
	if got := Coin(db, "struct-coin"); !coin.NewCoin(7, 0, "IOV").Equals(got) {
		t.Fatalf("unexpected coin: %v", got)
	}
}
