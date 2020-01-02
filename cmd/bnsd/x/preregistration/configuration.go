package preregistration

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
)

func (c *Configuration) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Owner", c.Owner.Validate())
	return errs
}

func loadConf(db gconf.Store) (Configuration, error) {
	var conf Configuration
	if err := gconf.Load(db, "preregistration", &conf); err != nil {
		return conf, errors.Wrap(err, "load configuration")
	}
	return conf, nil
}
