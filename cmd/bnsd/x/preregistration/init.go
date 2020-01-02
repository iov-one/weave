package preregistration

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
)

// Initializer fulfils the Initializer interface to load data from the genesis
// file
type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial account info from genesis and save it to the
// database
func (*Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, kv weave.KVStore) error {
	// Preregistration configuration is optional.
	if err := gconf.InitConfig(kv, opts, "preregistration", &Configuration{}); err != nil && !errors.ErrNotFound.Is(err) {
		return errors.Wrap(err, "init config")
	}
	return nil
}
