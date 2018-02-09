package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/confio/weave"
	"github.com/pkg/errors"
)

// Genesis file format, designed to be overlayed with tendermint genesis
type Genesis struct {
	ChainID    string  `json:"chain_id"`
	AppOptions Options `json:"app_options"`
}

// loadGenesis tries to load a given file into a Genesis struct
func loadGenesis(filePath string) (Genesis, error) {
	var gen Genesis

	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return gen, errors.Wrap(err, "loading genesis file")
	}

	// the basecoin genesis go-wire/data :)
	err = json.Unmarshal(bytes, &gen)
	if err != nil {
		return gen, errors.Wrap(err, "unmarshaling genesis file")
	}
	return gen, nil
}

// Options are the app options
// Each extension can look up it's key and parse the json as desired
type Options map[string]json.RawMessage

var _ weave.Options = Options{}

// ReadOptions reads the values stored under a given key,
// and parses the json into the given obj.
// Returns an error if it cannot parse.
// Noop and no error if key is missing
func (o Options) ReadOptions(key string, obj interface{}) error {
	msg := o[key]
	if len(msg) == 0 {
		return nil
	}
	return json.Unmarshal([]byte(msg), obj)
}

//------ init state -----

// ChainInitState lets you initialize many extensions with one function
func ChainInitState(inits ...weave.InitStater) weave.InitStater {
	return chainInitState{inits}
}

type chainInitState struct {
	inits []weave.InitStater
}

// InitState will pass opts to all InitStaters in the list,
// aborting at the first error.
func (c chainInitState) InitState(opts weave.Options, kv weave.KVStore) error {
	for _, i := range c.inits {
		err := i.InitState(opts, kv)
		if err != nil {
			return err
		}
	}
	return nil
}

//------- storing chainID ---------

const chainIDKey = "internal/chainID"

// loadChainID returns the chain id stored if any
func loadChainID(kv weave.KVStore) string {
	v := kv.Get([]byte(chainIDKey))
	return string(v)
}

// saveChainID stores a chain id in the kv store.
// Returns error if already set, or invalid name
func saveChainID(kv weave.KVStore, chainID string) error {
	if !weave.IsValidChainID(chainID) {
		return fmt.Errorf("Invalid chainID: %s", chainID)
	}
	k := []byte(chainIDKey)
	if kv.Has(k) {
		return fmt.Errorf("ChainID already set")
	}
	kv.Set(k, []byte(chainID))
	return nil
}
