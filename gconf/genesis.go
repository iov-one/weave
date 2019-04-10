package gconf

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// Initializer fulfils the InitStater interface to load data from
// the genesis file
type Initializer struct {
	// Conf must be set to a pointer to a configuration structure that
	// represents the configuration needed by the application.
	Conf interface{}
}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial account info from genesis
// and save it to the database
func (ini *Initializer) FromGenesis(opts weave.Options, db weave.KVStore) error {
	if err := Load(db, ini.Conf); err != nil {
		return errors.Wrap(err, "cannot load configuration from the store")
	}
	err := opts.ReadOptions("gconf", &ini.Conf)
	if err != nil {
		return err
	}

	if err := Save(db, ini.Conf); err != nil {
		return errors.Wrap(err, "cannot save configuration in the store")
	}
	return nil
}
