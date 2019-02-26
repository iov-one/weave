package commands

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
)

func TestInit(t *testing.T) {
	home := setupConfig(t)
	defer os.RemoveAll(home)

	logger := log.NewNopLogger()
	args := []string{"ETH", "ABCD123456789000DEADBEEF00ABCD123456789000"}
	err := server.InitCmd(app.GenInitOptions, logger, home, args)
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

// setupConfig creates a homedir to run inside,
// and copies demo tendermint files there.
//
// these files reside in testdata and can be created
// via `tendermint init`. Current version v0.16.0
func setupConfig(t *testing.T) string {
	rootDir, err := ioutil.TempDir("", "mock-sdk-cmd")
	require.NoError(t, err)
	err = copyConfigFiles(t, rootDir)
	require.NoError(t, err)
	return rootDir
}

func copyConfigFiles(t *testing.T, rootDir string) error {
	// make the output dir
	outDir := filepath.Join(rootDir, "config")
	err := os.Mkdir(outDir, 0755)
	if err != nil {
		return err
	}

	// copy everything over from testdata
	inDir := "testdata"
	files, err := ioutil.ReadDir(inDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		input := filepath.Join(inDir, f.Name())
		output := filepath.Join(outDir, f.Name())
		t.Logf("Copying %s to %s", input, output)
		err = fileCopy(input, output, f.Mode())
		if err != nil {
			return err
		}
	}

	return nil
}

func fileCopy(input, output string, mode os.FileMode) error {
	from, err := os.Open(input)
	if err != nil {
		return err
	}
	defer from.Close()

	to, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE, mode)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	return err
}
