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
	if len(m.Targets) == 0 {
		return errors.Wrap(errors.ErrEmpty, "targets")
	}
	for i, t := range m.Targets {
		if err := t.Validate(); err != nil {
			return errors.Wrapf(err, "target #%d", i)
		}
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
	return "username/changeTokenOwner"
}

func (m *ChangeTokenTargetsMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := m.Username.Validate(); err != nil {
		return errors.Wrap(err, "username")
	}
	if len(m.NewTargets) == 0 {
		return errors.Wrap(errors.ErrEmpty, "targets")
	}
	for i, t := range m.NewTargets {
		if err := t.Validate(); err != nil {
			return errors.Wrapf(err, "new target #%d", i)
		}
	}
	return nil
}

func (ChangeTokenTargetsMsg) Path() string {
	return "username/changeTokenTargets"
}
