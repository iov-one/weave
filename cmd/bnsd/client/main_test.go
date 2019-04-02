package client

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/x/cash"
	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	nm "github.com/tendermint/tendermint/node"
	rpctest "github.com/tendermint/tendermint/rpc/test"
	tm "github.com/tendermint/tendermint/types"
)

// configuration for genesis
var initBalance = coin.Coin{
	Whole:  100200300,
	Ticker: "TOOL",
}

// adjust this to get debug output
var logger = log.NewNopLogger() // log.NewTMLogger()

// useful values for test cases
var node *nm.Node
var faucet *PrivateKey

func getChainID() string {
	return rpctest.GetConfig().ChainID()
}

func TestMain(m *testing.M) {
	faucet = GenPrivateKey()

	// TODO: check out config file...
	config := rpctest.GetConfig()
	config.Moniker = "SetInTestMain"

	// set up our application
	admin := faucet.PublicKey().Address()
	app, err := initApp(config, admin)
	if err != nil {
		panic(err) // what else to do???
	}

	// run the app inside a tendermint instance
	node = rpctest.StartTendermint(app)
	time.Sleep(100 * time.Millisecond) // time to setup app context
	code := m.Run()

	// and shut down proper at the end
	node.Stop()
	node.Wait()
	os.Exit(code)
}

func initApp(config *cfg.Config, addr weave.Address) (abci.Application, error) {
	opts := &server.Options{
		MinFee: coin.Coin{},
		Home:   config.RootDir,
		Logger: logger,
		Debug:  false,
	}
	bcp, err := app.GenerateApp(opts)
	if err != nil {
		return nil, err
	}

	// generate genesis file...
	err = initGenesis(config.GenesisFile(), addr)
	return bcp, err
}

func initGenesis(filename string, addr weave.Address) error {
	doc, err := tm.GenesisDocFromFile(filename)
	if err != nil {
		return err
	}
	appState, err := json.Marshal(map[string]interface{}{
		"cash": []interface{}{
			map[string]interface{}{
				"address": addr,
				"coins":   coin.Coins{&initBalance},
			},
		},
		"gconf": map[string]interface{}{
			cash.GconfCollectorAddress: "fake-collector-address",
			cash.GconfMinimalFee:       coin.Coin{}, // no fee
		},
	})
	if err != nil {
		return fmt.Errorf("serialize state: %s", err)
	}
	doc.AppState = appState
	return doc.SaveAs(filename)
}
