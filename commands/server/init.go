package server

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/tendermint/tmlibs/log"
)

const (
	appStateKey = "app_state"
)

// InitCmd will initialize all files for tendermint,
// along with proper app_options.
// The application can pass in a function to generate
// proper options. And may want to use GenerateCoinKey
// to create default account(s).
func InitCmd(gen GenOptions, logger log.Logger, home string, args []string) error {
	// no app_options, leave like tendermint
	if gen == nil {
		return nil
	}

	// Now, we want to add the custom app_options
	options, err := gen(args)
	if err != nil {
		return err
	}

	// And add them to the genesis file
	genFile := filepath.Join(home, "config", "genesis.json")
	return addGenesisOptions(genFile, options)
}

// GenOptions can parse command-line and flag to
// generate default app_options for the genesis file.
// This is application-specific
type GenOptions func(args []string) (json.RawMessage, error)

// genesisDoc involves some tendermint-specific structures we don't
// want to parse, so we just grab it into a raw object format,
// so we can add one line.
type genesisDoc map[string]json.RawMessage

func addGenesisOptions(filename string, options json.RawMessage) error {
	bz, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var doc genesisDoc
	err = json.Unmarshal(bz, &doc)
	if err != nil {
		return err
	}

	doc[appStateKey] = options
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, out, 0600)
}
