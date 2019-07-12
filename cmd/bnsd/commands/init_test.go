package commands

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/tmtest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/tendermint/tendermint/libs/log"
)

func TestInit(t *testing.T) {
	home, cleanup := tmtest.SetupConfig(t, "testdata")
	defer cleanup()

	logger := log.NewNopLogger()
	args := []string{"ETH", "a5dd251d3cd29dae900b089218ae9740165139fa"}
	err := server.InitCmd(bnsd.GenInitOptions, logger, home, args)
	assert.Nil(t, err)

	// make sure we set proper data
	genFile := filepath.Join(home, "config", "genesis.json")

	bz, err := ioutil.ReadFile(genFile)
	assert.Nil(t, err)

	var genesis struct {
		State struct {
			Cash []struct {
				Address weave.Address
				Coins   coin.Coins
			}
		} `json:"app_state"`
	}
	err = json.Unmarshal(bz, &genesis)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(genesis.State.Cash))
	wallet := genesis.State.Cash[0]
	want, err := hex.DecodeString(args[1])
	assert.Nil(t, err)
	assert.Equal(t, weave.Address(want), wallet.Address)
	assert.Equal(t, 1, len(wallet.Coins))
	assert.Equal(t, &coin.Coin{Ticker: args[0], Whole: 123456789}, wallet.Coins[0])
}
