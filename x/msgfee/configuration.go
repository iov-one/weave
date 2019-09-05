package msgfee

import (
	"github.com/iov-one/weave/errors"
)

func (c *Configuration) Validate() error {
	var errs error
	// Owner field is optional.
	if len(c.Owner) != 0 {
		errs = errors.AppendField(errs, "Owner", c.Owner.Validate())
	}
	// FeeAdmin field is optional.
	if len(c.FeeAdmin) != 0 {
		errs = errors.AppendField(errs, "FeeAdmin", c.FeeAdmin.Validate())
	}
	return errs
}
