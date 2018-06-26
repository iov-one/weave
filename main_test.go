package utils

import (
	"os"
	"testing"

	"github.com/tendermint/abci/example/kvstore"
	nm "github.com/tendermint/tendermint/node"
	rpctest "github.com/tendermint/tendermint/rpc/test"
)

var node *nm.Node

func TestMain(m *testing.M) {
	// TODO: use our app
	// start a tendermint node (and kvstore) in the background to test against
	app := kvstore.NewKVStoreApplication()

	// TODO: check out config file...
	config := rpctest.GetConfig()
	config.Moniker = "SetInTestMain"

	node = rpctest.StartTendermint(app)
	code := m.Run()

	// and shut down proper at the end
	node.Stop()
	node.Wait()
	os.Exit(code)
}
