package app

import (
	"encoding/json"
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
