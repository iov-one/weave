package client

import (
	"os"
	"testing"

	"github.com/tendermint/tendermint/abci/example/kvstore"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/rpc/test"
)

// useful values for test cases
var node *nm.Node

func TestMain(m *testing.M) {
	config := rpctest.GetConfig()
	// this is just for fun :)
	config.Moniker = "WeaveClientTest"
	// we must set these two to ensure that app.key is indexed (IndexTags non-empty overrides IndexAllTags)
	config.TxIndex.IndexTags = ""
	config.TxIndex.IndexAllTags = true

	// run the default kvstore app inside a tendermint instance
	app := kvstore.NewKVStoreApplication()
	code := TestWithTendermint(app, func(n *nm.Node) {
		node = n
	}, m)
	os.Exit(code)
}
