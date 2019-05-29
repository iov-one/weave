package cash

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
)

func (c *Configuration) Validate() error {
	// owner field is optional... possible to make it immutable
	if len(c.Owner) != 0 {
		if err := c.Owner.Validate(); err != nil {
			return errors.Wrap(err, "owner address")
		}
	}
	if len(c.CollectorAddress) == 0 {
		return errors.Wrap(errors.ErrState, "collector address missing")
	}
	if err := c.CollectorAddress.Validate(); err != nil {
		return errors.Wrap(err, "collector address")
	}

	if !c.MinimalFee.IsZero() {
		if err := c.MinimalFee.Validate(); err != nil {
			return errors.Wrap(err, "minimal fee")
		}
		if !c.MinimalFee.IsNonNegative() {
			return errors.Wrap(errors.ErrState, "minimal fee cannot be negative")
		}
	}
	return nil
}

func mustLoadConf(db gconf.Store) Configuration {
	var conf Configuration
	if err := gconf.Load(db, "cash", &conf); err != nil {
		err = errors.Wrap(err, "load configuration")
		panic(err)
	}
	return conf
}
