package txfee

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &UpdateConfigurationMsg{}, migration.NoModification)
}

var _ weave.Msg = (*UpdateConfigurationMsg)(nil)

func (m *UpdateConfigurationMsg) Validate() error {
	var errs error
	c := m.Patch
	if len(c.Owner) != 0 {
		errs = errors.AppendField(errs, "Patch.Owner", c.Owner.Validate())
	}
	if err := c.BaseFee.Validate(); err != nil {
		errs = errors.AppendField(errs, "Patch.BaseFee", err)
	} else if !c.BaseFee.IsPositive() {
		errs = errors.AppendField(errs, "Patch.BaseFee",
			errors.Wrap(errors.ErrAmount, "must be a positive value"))
	}
	return errs
}

func (*UpdateConfigurationMsg) Path() string {
	return "txfee/update_configuration"
}
