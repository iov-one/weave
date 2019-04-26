package client

import (
	"context"
	"fmt"
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
	fmt.Println("Starting tendermint...")
	node = rpctest.StartTendermint(app)

	// make sure tendermint is good to go before tests... a short static pause,
	// then wait for one block to come in.
	fmt.Println("Wait for first block...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	h, err := NewClient(NewLocalConnection(node)).WaitForNextBlock(ctx)
	fmt.Printf("Starting tests with block %d\n", h.Height)

	// Run tests if tendermint started properly
	var code int
	if err == nil {
		code = m.Run()
	} else {
		fmt.Printf("Failed to start tendermint: %s\n", err)
		code = 1
	}

	// and shut down proper at the end
	node.Stop()
	node.Wait()
	os.Exit(code)
}
