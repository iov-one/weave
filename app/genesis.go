package app

import (
	"github.com/confio/weave"
	"github.com/confio/weave/errors"
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

//------- storing chainID ---------

// _wv: is a prefix for weave internal data
const chainIDKey = "_wv:chainID"

// loadChainID returns the chain id stored if any
func loadChainID(kv weave.KVStore) string {
	v := kv.Get([]byte(chainIDKey))
	return string(v)
}

// saveChainID stores a chain id in the kv store.
// Returns error if already set, or invalid name
func saveChainID(kv weave.KVStore, chainID string) error {
	if !weave.IsValidChainID(chainID) {
		return errors.ErrInvalidChainID(chainID)
	}
	k := []byte(chainIDKey)
	if kv.Has(k) {
		return errors.ErrModifyChainID()
	}
	kv.Set(k, []byte(chainID))
	return nil
}
