package client

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/iov-one/weave"
	weaveClient "github.com/iov-one/weave/client"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/migration"
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

	code := weaveClient.TestWithTendermint(app, func(n *nm.Node) {
		node = n
	}, m)
	os.Exit(code)

}

func initApp(config *cfg.Config, addr weave.Address) (abci.Application, error) {
	opts := &server.Options{
		MinFee: coin.Coin{},
		Home:   config.RootDir,
		Logger: logger,
		Debug:  false,
	}
	bcp, err := bnsd.GenerateApp(opts)
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
			dict{
				"address": addr,
				"coins":   coin.Coins{&initBalance},
			},
		},
		"conf": dict{
			"cash": cash.Configuration{
				CollectorAddress: weave.NewAddress([]byte("fake-collector-address")),
				MinimalFee:       coin.Coin{}, // no fee
			},
			"migration": migration.Configuration{
				Admin: weave.Condition("multisig/usage/0000000000000001").Address(),
			},
		},
		"initialize_schema": []dict{
			{"ver": 1, "pkg": "batch"},
			{"ver": 1, "pkg": "cash"},
			{"ver": 1, "pkg": "currency"},
			{"ver": 1, "pkg": "distribution"},
			{"ver": 1, "pkg": "escrow"},
			{"ver": 1, "pkg": "gov"},
			{"ver": 1, "pkg": "msgfee"},
			{"ver": 1, "pkg": "multisig"},
			{"ver": 1, "pkg": "paychan"},
			{"ver": 1, "pkg": "sigs"},
			{"ver": 1, "pkg": "utils"},
			{"ver": 1, "pkg": "validators"},
		},
	})
	if err != nil {
		return fmt.Errorf("serialize state: %s", err)
	}
	doc.AppState = appState
	return doc.SaveAs(filename)
}

type dict map[string]interface{}
