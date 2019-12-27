package txfee

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &Configuration{}, migration.NoModification)
}

func (c *Configuration) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Owner", c.Owner.Validate())
	if err := c.BaseFee.Validate(); err != nil {
		errs = errors.AppendField(errs, "BaseFee", err)
	} else if !c.BaseFee.IsPositive() {
		errs = errors.AppendField(errs, "BaseFee",
			errors.Wrap(errors.ErrAmount, "must be a positive value"))
	}
	return errs
}
