package app

import (
	"reflect"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

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
	for _, ini := range c.inits {
		if err := ini.FromGenesis(opts, params, kv); err != nil {
			// Attach package name to the error produced by the
			// initializer. This is extremely helpful in narrowing
			// down where the genesis declaration is invalid.
			tp := reflect.TypeOf(ini)
			for tp.Kind() == reflect.Ptr {
				tp = tp.Elem()
			}
			pkg := tp.PkgPath()
			return errors.Wrapf(err, "initializer %q", pkg)
		}
	}
	return nil
}
