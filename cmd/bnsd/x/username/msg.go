package username

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &RegisterTokenMsg{}, migration.NoModification)
	migration.MustRegister(1, &TransferTokenMsg{}, migration.NoModification)
	migration.MustRegister(1, &ChangeTokenTargetsMsg{}, migration.NoModification)
}

func (m *RegisterTokenMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := m.Username.Validate(); err != nil {
		return errors.Wrap(err, "username")
	}
	if err := validateTargets(m.Targets); err != nil {
		return errors.Wrap(err, "targets")
	}
	return nil
}

func (RegisterTokenMsg) Path() string {
	return "username/registerToken"
}

func (m *TransferTokenMsg) Validate() error {
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

func (TransferTokenMsg) Path() string {
	return "username/transferToken"
}

func (m *ChangeTokenTargetsMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := m.Username.Validate(); err != nil {
		return errors.Wrap(err, "username")
	}
	if err := validateTargets(m.NewTargets); err != nil {
		return errors.Wrap(err, "new targets")
	}
	return nil
}

func (ChangeTokenTargetsMsg) Path() string {
	return "username/changeTokenTargets"
}
