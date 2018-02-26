package app

import (
	"github.com/confio/weave"
)

// Genesis file format, designed to be overlayed with tendermint genesis
type Genesis struct {
	ChainID  string        `json:"chain_id"`
	AppState weave.Options `json:"app_state"`
}

//------ init state -----

// ChainInitializers lets you initialize many extensions with one function
func ChainInitializers(inits ...weave.Initializer) weave.Initializer {
	return chainInitializer{inits}
}

type chainInitializer struct {
	inits []weave.Initializer
}

// FromGenesis will pass opts to all Initializers in the list,
// aborting at the first error.
func (c chainInitializer) FromGenesis(opts weave.Options, kv weave.KVStore) error {
	for _, i := range c.inits {
		err := i.FromGenesis(opts, kv)
		if err != nil {
			return err
		}
	}
	return nil
}
