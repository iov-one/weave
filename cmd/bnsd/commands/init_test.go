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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
)

func TestInit(t *testing.T) {
	home, cleanup := tmtest.SetupConfig(t, "testdata")
	defer cleanup()

	logger := log.NewNopLogger()
	args := []string{"ETH", "a5dd251d3cd29dae900b089218ae9740165139fa"}
	err := server.InitCmd(bnsd.GenInitOptions, logger, home, args)
	require.NoError(t, err)

	// make sure we set proper data
	genFile := filepath.Join(home, "config", "genesis.json")

	bz, err := ioutil.ReadFile(genFile)
	require.NoError(t, err)

	var genesis struct {
		State struct {
			Cash []struct {
				Address weave.Address
				Coins   coin.Coins
			}
		} `json:"app_state"`
	}
	err = json.Unmarshal(bz, &genesis)
	assert.NoErrorf(t, err, "cannot unmarshal genesis: %s", err)

	if assert.Equal(t, 1, len(genesis.State.Cash), string(bz)) {
		wallet := genesis.State.Cash[0]
		want, err := hex.DecodeString(args[1])
		assert.NoError(t, err)
		assert.Equal(t, weave.Address(want), wallet.Address)
		if assert.Equal(t, 1, len(wallet.Coins), "Genesis: %s", bz) {
			assert.Equal(t, &coin.Coin{Ticker: args[0], Whole: 123456789}, wallet.Coins[0])
		}
	}
}
