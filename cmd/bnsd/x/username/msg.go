package username

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &RegisterTokenMsg{}, migration.NoModification)
	migration.MustRegister(1, &TransferTokenMsg{}, migration.NoModification)
	migration.MustRegister(1, &ChangeTokenTargetsMsg{}, migration.NoModification)
	migration.MustRegister(1, &RegisterNamespaceMsg{}, migration.NoModification)
}

var _ weave.Msg = (*RegisterNamespaceMsg)(nil)

func (m *RegisterNamespaceMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.Label) == 0 {
		errs = errors.AppendField(errs, "Label", errors.ErrEmpty)
	}
	return errs
}

func (RegisterNamespaceMsg) Path() string {
	return "username/register_namespace"
}

var _ weave.Msg = (*RegisterTokenMsg)(nil)

func (m *RegisterTokenMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}

	// Username should but cannot be validated here.

	if err := validateTargets(m.Targets); err != nil {
		return errors.Wrap(err, "targets")
	}
	return nil
}

func (RegisterTokenMsg) Path() string {
	return "username/register_token"
}

var _ weave.Msg = (*TransferTokenMsg)(nil)

func (m *TransferTokenMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}

	// Username should but cannot be validated here.

	if err := m.NewOwner.Validate(); err != nil {
		return errors.Wrap(err, "new owner")
	}
	return nil
}

func (TransferTokenMsg) Path() string {
	return "username/transfer_token"
}

var _ weave.Msg = (*ChangeTokenTargetsMsg)(nil)

func (m *ChangeTokenTargetsMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}

	// Username should but cannot be validated here.

	if err := validateTargets(m.NewTargets); err != nil {
		return errors.Wrap(err, "new targets")
	}
	return nil
}

func (ChangeTokenTargetsMsg) Path() string {
	return "username/change_token_targets"
}
