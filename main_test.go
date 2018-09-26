package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bcpd/app"
	"github.com/iov-one/weave/x"

	abci "github.com/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	nm "github.com/tendermint/tendermint/node"
	rpctest "github.com/tendermint/tendermint/rpc/test"
	tm "github.com/tendermint/tendermint/types"
	"github.com/tendermint/tmlibs/log"
)

// configuration for genesis
var initBalance = x.Coin{
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
	code := m.Run()

	// and shut down proper at the end
	node.Stop()
	node.Wait()
	os.Exit(code)
}

func initApp(config *cfg.Config, addr weave.Address) (abci.Application, error) {
	bcp, err := app.GenerateApp(config.RootDir, logger, false)
	if err != nil {
		return nil, err
	}

	// generate genesis file...
	err = initGenesis(config.GenesisFile(), addr)
	return bcp, err
}

func initGenesis(filename string, addr weave.Address) error {
	// load genesis
	doc, err := tm.GenesisDocFromFile(filename)
	if err != nil {
		return err
	}

	// set app state
	token, _ := json.Marshal(initBalance)
	appState := fmt.Sprintf(`{
        "wallets": [{
            "name": "faucet",
            "address": "%s",
            "coins": [%s]
        }],
        "tokens": [{
            "ticker": "%s",
            "name": "Default token",
            "sig_figs": 9
        }]
    }`, addr, string(token), initBalance.Ticker)
	doc.AppStateJSON = []byte(appState)

	// save file
	return doc.SaveAs(filename)
}
