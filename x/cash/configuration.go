package cash

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/x"
)

func (c *Configuration) Validate() error {
	// owner field is optional... possible to make it immutible
	if len(c.Owner) != 0 {
		if err := c.Owner.Validate(); err != nil {
			return errors.Wrap(err, "owner address")
		}
	}
	if len(c.CollectorAddress) == 0 {
		return errors.Wrap(errors.ErrInvalidState, "collector address missing")
	}
	if err := c.CollectorAddress.Validate(); err != nil {
		return errors.Wrap(err, "collector address")
	}

	if !c.MinimalFee.IsZero() {
		if err := c.MinimalFee.Validate(); err != nil {
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
	if err := gconf.Load(db, "cash", &conf); err != nil {
		err = errors.Wrap(err, "load configuration")
		panic(err)
	}
	return conf
}

var _ weave.Msg = (*ConfigurationMsg)(nil)

// Validate will skip any zero fields and validate the set ones
// TODO: we should make it easier to reuse code with Configuration
func (m *ConfigurationMsg) Validate() error {
	c := m.Patch
	if len(c.Owner) != 0 {
		if err := c.Owner.Validate(); err != nil {
			return errors.Wrap(err, "owner address")
		}
	}
	if len(c.CollectorAddress) != 0 {
		if err := c.CollectorAddress.Validate(); err != nil {
			return errors.Wrap(err, "collector address")
		}
	}
	if !c.MinimalFee.IsZero() {
		if err := c.MinimalFee.Validate(); err != nil {
			return errors.Wrap(err, "minimal fee")
		}
		if !c.MinimalFee.IsNonNegative() {
			return errors.Wrap(errors.ErrInvalidState, "minimal fee cannot be negative")
		}
	}
	return nil
}

func (*ConfigurationMsg) Path() string {
	return "cash/update_config"
}

func NewConfigHandler(auth x.Authenticator) weave.Handler {
	var conf Configuration
	return gconf.NewUpdateConfigurationHandler("cash", &conf, auth)
}
