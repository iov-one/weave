package app

import (
	"github.com/iov-one/weave"
)

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
func (c chainInitializer) FromGenesis(opts weave.Options, params weave.GenesisParams, kv weave.KVStore) error {
	for _, i := range c.inits {
		err := i.FromGenesis(opts, params, kv)
		if err != nil {
			return err
		}
	}
	return nil
}
