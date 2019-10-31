package migration

import (
	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
)

func (c *Configuration) Validate() error {
	if err := c.Admin.Validate(); err != nil {
		return errors.Wrap(err, "admin")
	}
	return nil
}

func loadConf(db gconf.ReadStore) (*Configuration, error) {
	var conf Configuration
	if err := gconf.Load(db, "migration", &conf); err != nil {
		return nil, errors.Wrap(err, "gconf")
	}
	return &conf, nil
}

// CurrentAdmin returns migration extension admin address as currently
// configured.
//
// This function is useful for the `gconf` package users to provide a one time
// authentication address during configuration initialization. See
// `gconf.NewUpdateConfigurationHandler` for more details.
func CurrentAdmin(db weave.ReadOnlyKVStore) (weave.Address, error) {
	conf, err := loadConf(db)
	if err != nil {
		return nil, errors.Wrap(err, "load configuration")
	}
	return conf.Admin, nil
}
