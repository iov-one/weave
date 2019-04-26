package migration

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
)

func (c *Configuration) Validate() error {
	if err := c.Admin.Validate(); err != nil {
		return errors.Wrap(err, "admin")
	}
	return nil
}

func mustLoadConf(db gconf.Store) Configuration {
	var conf Configuration
	if err := gconf.Load(db, "migration", &conf); err != nil {
		err = errors.Wrap(err, "load configuration")
		panic(err)
	}
	return conf
}
