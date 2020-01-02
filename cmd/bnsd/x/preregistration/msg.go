package preregistration

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

var _ weave.Msg = (*RegisterMsg)(nil)

func (RegisterMsg) Path() string {
	return "preregistration/register"
}

func (msg *RegisterMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	if !isValidDomain(msg.Domain) {
		errs = errors.AppendField(errs, "Domain", errors.Wrapf(errors.ErrInput, "must match %q", validDomainRule))
	}
	errs = errors.AppendField(errs, "Owner", msg.Owner.Validate())
	return errs
}
