package account

import (
	"regexp"

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
	if err := validateRegexp(c.ValidDomain); err != nil {
		errs = errors.AppendField(errs, "ValidDomain", err)
	}
	if err := validateRegexp(c.ValidName); err != nil {
		errs = errors.AppendField(errs, "ValidName", err)
	}
	if err := validateRegexp(c.ValidBlockchainID); err != nil {
		errs = errors.AppendField(errs, "ValidBlockchainID", err)
	}
	if err := validateRegexp(c.ValidBlockchainAddress); err != nil {
		errs = errors.AppendField(errs, "ValidBlockchainAddress", err)
	}
	if c.DomainRenew <= 0 {
		errs = errors.AppendField(errs, "DomainRenew", errors.Wrap(errors.ErrInput, "must be greater than zero"))
	}
	return errs
}

// validateRegexp returns an error if provided string is not a valid regular
// expression.
// This function ensures that the regular expression is a complete match test
// by ensuring ^ and $ presence.
func validateRegexp(rx string) error {
	if rx == "" {
		return errors.Wrap(errors.ErrEmpty, "cannot be empty")
	}
	if len(rx) > 1024 {
		return errors.Wrap(errors.ErrInput, "too long")
	}
	_, err := regexp.Compile(rx)
	if err != nil {
		return errors.Wrap(errors.ErrInput, err.Error())
	}

	if rx[0] != '^' || rx[len(rx)-1] != '$' {
		return errors.Wrap(errors.ErrInput, "regular expression must match the whole input, start with ^ and end with $ characters to enforce full match")
	}

	return nil
}

func loadConf(db gconf.Store) (*Configuration, error) {
	var conf Configuration
	if err := gconf.Load(db, "account", &conf); err != nil {
		return nil, errors.Wrap(err, "load")
	}
	return &conf, nil
}
