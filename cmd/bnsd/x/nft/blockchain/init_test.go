package blockchain

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
			"blockchains": [
				{
					"id": "id-123",
					"owner": "6f776e65722d3132333030303030303030303030",
					"chain": {
						"chain_id": "chain-123",
						"network_id": "network-123",
						"name": "name-123",
						"enabled": true,
						"production": true,
						"main_ticker_id": "main-ticker-id-123"
					},
					"iov": {
						"codec": "happy_little_tree",
						"codec_config": "{\"any\": [\"foo\"]}"
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
	token, err := AsBlockchain(obj)
	if err != nil {
		t.Fatalf("returned object is not a token: %s", err)
	}

	chain := token.GetChain()
	if want, got := "chain-123", chain.ChainID; want != got {
		t.Fatalf("want %q, got %q", want, got)
	}
	if want, got := "network-123", chain.NetworkID; want != got {
		t.Fatalf("want %q, got %q", want, got)
	}
	if want, got := true, chain.Enabled; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
	if want, got := true, chain.Production; want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}
