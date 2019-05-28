package commands

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/iov-one/weave/examples/mycoind/app"
)

func TestInit(t *testing.T) {
	home := setupConfig(t)
	defer os.RemoveAll(home)

	logger := log.NewNopLogger()
	args := []string{"ETH", "ABCD123456789000DEADBEEF00ABCD123456789000"}
	err := server.InitCmd(app.GenInitOptions, logger, home, args)
	assert.Nil(t, err)

	// make sure we set proper data
	genFile := filepath.Join(home, "config", "genesis.json")

	var doc server.GenesisDoc
	bz, err := ioutil.ReadFile(genFile)
	assert.Nil(t, err)
	err = json.Unmarshal(bz, &doc)
	assert.Nil(t, err)
	// keep old values, and add our values
	assert.Equal(t, json.RawMessage(`"test-chain-tspYJj"`),
		doc["chain_id"])
	if doc["validators"] == nil || len(doc["validators"]) == 0 {
		t.Fatalf("genesis validators not defined")
	}
	if doc[server.AppStateKey] == nil || len(doc[server.AppStateKey]) == 0 {
		t.Fatalf("genesis app state not defined")
	}

	gTime := time.Time{}
	err = gTime.UnmarshalJSON(doc[server.GenesisTimeKey])
	assert.Nil(t, err)
	assert.Equal(t, true, gTime.After(time.Now().Add(-10*time.Minute)))

	if !strings.Contains(string(doc[server.AppStateKey]), `"ticker": "ETH"`) {
		t.Fatalf("Missing ETH ticker in genesis app state")
	}

	err = server.InitCmd(app.GenInitOptions, logger, home, args)
	assert.Equal(t, errors.ErrState.Is(err), true)
}

// setupConfig creates a homedir to run inside,
// and copies demo tendermint files there.
//
// these files reside in testdata and can be created
// via `tendermint init`. Current version v0.16.0
func setupConfig(t *testing.T) string {
	rootDir, err := ioutil.TempDir("", "mock-sdk-cmd")
	assert.Nil(t, err)
	err = copyConfigFiles(rootDir)
	assert.Nil(t, err)
	return rootDir
}

func copyConfigFiles(rootDir string) error {
	// make the output dir
	outDir := filepath.Join(rootDir, server.DirConfig)
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
