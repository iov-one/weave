package client

import (
	"os"
	"testing"
	"time"

	"github.com/tendermint/tendermint/abci/example/kvstore"
	nm "github.com/tendermint/tendermint/node"
	rpctest "github.com/tendermint/tendermint/rpc/test"
)

// useful values for test cases
var node *nm.Node

func getChainID() string {
	return rpctest.GetConfig().ChainID()
}

func TestMain(m *testing.M) {
	config := rpctest.GetConfig()
	// this is just for fun :)
	config.Moniker = "WeaveClientTest"
	// we must set these two to ensure that app.key is indexed (IndexTags non-empty overrides IndexAllTags)
	config.TxIndex.IndexTags = ""
	config.TxIndex.IndexAllTags = true

	// run the default kvstore app inside a tendermint instance
	app := kvstore.NewKVStoreApplication()
	node = rpctest.StartTendermint(app)
	time.Sleep(100 * time.Millisecond) // time to setup app context
	code := m.Run()

	// and shut down proper at the end
	node.Stop()
	node.Wait()
	os.Exit(code)
}
