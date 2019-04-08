package cash

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
)

func (c *Configuration) Validate() error {
	if len(c.CollectorAddress) == 0 {
		return errors.Wrap(errors.ErrInvalidState, "collector address missing")
	}
	if err := c.CollectorAddress.Validate(); err != nil {
		return errors.Wrap(err, "collector address")
	}

	if !c.MinimalFee.IsZero() {
		if err := c.Validate(); err != nil {
			return errors.Wrap(err, "minimal fee")
		}
		if !c.MinimalFee.IsNonNegative() {
			return errors.Wrap(errors.ErrInvalidState, "minimal fee cannot be negative")
		}
	}
	return nil
}

func mustLoadConf(db gconf.Store) Configuration {
	var conf Configuration
	if err := gconf.Load(db, &conf); err != nil {
		err = errors.Wrap(err, "load configuration")
		panic(err)
	}
	return conf
}
