package server

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/tendermint/tmlibs/log"

	"github.com/confio/weave"
	"github.com/confio/weave/crypto"
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

// GenerateCoinKey returns the address of a public key,
// along with the secret phrase to recover the private key.
// You can give coins to this address and return the recovery
// phrase to the user to access them.
func GenerateCoinKey() (weave.Address, string, error) {
	// TODO: we need to generate BIP39 recovery phrases in crypto
	privKey := crypto.GenPrivKeyEd25519()
	addr := privKey.PublicKey().Address()
	return addr, "TODO: add a recovery phrase", nil
}

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
