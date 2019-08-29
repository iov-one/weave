package msgfee

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &SetMsgFeeMsg{}, migration.NoModification)
	migration.MustRegister(1, &UpdateConfigurationMsg{}, migration.NoModification)
}

var _ weave.Msg = (*SetMsgFeeMsg)(nil)

func (m *SetMsgFeeMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.MsgPath) == 0 {
		errs = errors.AppendField(errs, "MsgPath", errors.ErrEmpty)
	}
	if !m.Fee.IsNonNegative() {
		errs = errors.AppendField(errs, "Fee",
			errors.Wrap(errors.ErrAmount, "must be non negative"))
	}
	return errs
}

func (SetMsgFeeMsg) Path() string {
	return "msgfee/set_msg_fee"
}

var _ weave.Msg = (*UpdateConfigurationMsg)(nil)

// Validate will skip any zero fields and validate the set ones.
func (m *UpdateConfigurationMsg) Validate() error {
	var errs error
	c := m.Patch
	if len(c.Owner) != 0 {
		errs = errors.AppendField(errs, "Owner", c.Owner.Validate())
	}
	if len(c.FeeAdmin) != 0 {
		errs = errors.AppendField(errs, "FeeAdmin", c.FeeAdmin.Validate())
	}
	return errs
}

func (*UpdateConfigurationMsg) Path() string {
	return "msgfee/update_configuration"
}
