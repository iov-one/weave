package ticker

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
)

func TestInitializer(t *testing.T) {
	const genesis = `
	{
		"nfts": {
			"tickers": [
				{
					"base": {
						"id": "id-123",
						"owner": "6f776e65722d3132333030303030303030303030"
					},
					"details": {
						"blockchain_id": "chain-123"
					}
				}
			]
		}
	}
	`
	var opts weave.Options
	if err := json.Unmarshal([]byte(genesis), &opts); err != nil {
		t.Fatalf("cannot unmarshal JSON serialized genesis: %s", err)
	}

	var ini Initializer
	db := store.MemStore()
	if err := ini.FromGenesis(opts, db); err != nil {
		t.Fatalf("cannot initialize from genesis: %+v", err)
	}

	bucket := NewBucket()
	obj, err := bucket.Get(db, []byte("id-123"))
	if err != nil {
		t.Fatalf("cannot find token in the database: %s", err)
	} else if obj == nil {
		t.Fatal("cannot find token in the database: does not exist")
	}
	token, ok := obj.Value().(*TickerToken)
	if !ok {
		t.Fatalf("returned object is not a token: %T", obj.Value())
	}

	if want, got := "id-123", string(token.Base.ID); want != got {
		t.Fatalf("want %q, got %q", want, got)
	}
	if want, got := "owner-12300000000000", string(token.Base.Owner); want != got {
		t.Fatalf("want %q, got %q", want, got)
	}
	if want, got := "chain-123", string(token.Details.BlockchainID); want != got {
		t.Fatalf("want %q, got %q", want, got)
	}
}
