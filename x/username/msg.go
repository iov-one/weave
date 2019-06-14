package username

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &RegisterTokenMsg{}, migration.NoModification)
	migration.MustRegister(1, &ChangeTokenOwnerMsg{}, migration.NoModification)
	migration.MustRegister(1, &ChangeTokenTargetMsg{}, migration.NoModification)
}

func (m *RegisterTokenMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := m.Username.Validate(); err != nil {
		return errors.Wrap(err, "username")
	}
	if err := m.Target.Validate(); err != nil {
		return errors.Wrap(err, "target")
	}
	return nil
}

func (RegisterTokenMsg) Path() string {
	return "username/register-token"
}

func (m *ChangeTokenOwnerMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := m.Username.Validate(); err != nil {
		return errors.Wrap(err, "username")
	}
	if err := m.NewOwner.Validate(); err != nil {
		return errors.Wrap(err, "new owner")
	}
	return nil
}

func (ChangeTokenOwnerMsg) Path() string {
	return "username/change-token-owner"
}

func (m *ChangeTokenTargetMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := m.Username.Validate(); err != nil {
		return errors.Wrap(err, "username")
	}
	if err := m.NewTarget.Validate(); err != nil {
		return errors.Wrap(err, "new target")
	}
	return nil
}

func (ChangeTokenTargetMsg) Path() string {
	return "username/change-token-target"
}
