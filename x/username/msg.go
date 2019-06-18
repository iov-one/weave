package username

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &RegisterUsernameTokenMsg{}, migration.NoModification)
	migration.MustRegister(1, &TransferUsernameTokenMsg{}, migration.NoModification)
	migration.MustRegister(1, &ChangeUsernameTokenTargetsMsg{}, migration.NoModification)
}

func (m *RegisterUsernameTokenMsg) Validate() error {
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

func (RegisterUsernameTokenMsg) Path() string {
	return "username/registerUsernameToken"
}

func (m *TransferUsernameTokenMsg) Validate() error {
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

func (TransferUsernameTokenMsg) Path() string {
	return "username/transferUsernameToken"
}

func (m *ChangeUsernameTokenTargetsMsg) Validate() error {
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

func (ChangeUsernameTokenTargetsMsg) Path() string {
	return "username/changeUsernameTokenTargets"
}
