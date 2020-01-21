package qualityscore

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &Configuration{}, migration.NoModification)
}

func (c *Configuration) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", c.Metadata.Validate())
	errs = errors.AppendField(errs, "Owner", c.Owner.Validate())
	return errs
}

func loadConf(db gconf.Store) (*Configuration, error) {
	var conf Configuration
	if err := gconf.Load(db, "account", &conf); err != nil {
		return nil, errors.Wrap(err, "load")
	}
	return &conf, nil
}
