package server

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tmlibs/log"

	"github.com/confio/weave/std"
)

// setupViper creates a homedir to run inside,
// and copies demo tendermint files there.
//
// these files reside in testdata and can be created
// via `tendermint init`. Current version v0.16.0
func setupViper(t *testing.T) func() {
	rootDir, err := ioutil.TempDir("", "mock-sdk-cmd")
	require.NoError(t, err)
	err = copyConfigFiles(rootDir)
	require.NoError(t, err)

	viper.Set("home", rootDir)
	return func() {
		os.RemoveAll(rootDir)
	}
}

func copyConfigFiles(rootDir string) error {
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
		fmt.Printf("Copying %s to %s\n", input, output)
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

func TestInit(t *testing.T) {
	defer setupViper(t)()

	logger := log.NewNopLogger()
	cmd := InitCmd(std.GenInitOptions, logger)
	err := cmd.RunE(nil, nil)
	require.NoError(t, err)

	// make sure we set proper data
	home := viper.GetString("home")
	genFile := filepath.Join(home, "config", "genesis.json")

	var doc genesisDoc
	bz, err := ioutil.ReadFile(genFile)
	require.NoError(t, err)
	err = json.Unmarshal(bz, &doc)
	require.NoError(t, err)
	// keep old values, and add our values
	assert.EqualValues(t, []byte(`"test-chain-LgVOZ0"`),
		doc["chain_id"])
	assert.NotEmpty(t, doc["validators"])
	assert.NotEmpty(t, doc[appStateKey])
}
