package gconf

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
)

func TestGenesisInitializer(t *testing.T) {
	const genesis = `
		{
			"gconf": {
				"a-string": "hello",
				"an-int": 321
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
		t.Fatalf("unexpected value: %v", got)
	}
	if got := Int(db, "an-int"); got != 321 {
		t.Fatalf("unexpected value: %v", got)
	}
}
