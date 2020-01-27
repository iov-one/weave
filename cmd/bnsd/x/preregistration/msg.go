package preregistration

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &RegisterMsg{}, migration.NoModification)
	migration.MustRegister(1, &UpdateConfigurationMsg{}, migration.NoModification)
}

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

var _ weave.Msg = (*UpdateConfigurationMsg)(nil)

func (UpdateConfigurationMsg) Path() string {
	return "preregistration/update_configuration"
}

func (m *UpdateConfigurationMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "Patch", m.Patch.Validate())
	return errs
}
