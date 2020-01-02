package termdeposit

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateDepositContractMsg{}, migration.NoModification)
	migration.MustRegister(1, &DepositMsg{}, migration.NoModification)
	migration.MustRegister(1, &ReleaseDepositMsg{}, migration.NoModification)
	migration.MustRegister(1, &UpdateConfigurationMsg{}, migration.NoModification)
}

var _ weave.Msg = (*CreateDepositContractMsg)(nil)

func (CreateDepositContractMsg) Path() string {
	return "termdeposit/create_deposit_contract"
}

func (m *CreateDepositContractMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	// Only a base ValidUntil validation can be done without knowing the
	// current time. A full validation must be done in a handler.
	errs = errors.AppendField(errs, "ValidSince", m.ValidSince.Validate())
	errs = errors.AppendField(errs, "ValidUntil", m.ValidUntil.Validate())
	if !m.ValidSince.Time().Before(m.ValidUntil.Time()) {
		errs = errors.AppendField(errs, "ValidSince",
			errors.Wrap(errors.ErrInput, "ValidSince must be before ValidUntil"))
	}
	return errs
}

var _ weave.Msg = (*DepositMsg)(nil)

func (DepositMsg) Path() string {
	return "termdeposit/deposit"
}

func (m *DepositMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.DepositContractID) == 0 {
		errs = errors.AppendField(errs, "DepositContractID", errors.ErrEmpty)
	}
	if err := m.Amount.Validate(); err != nil {
		errs = errors.AppendField(errs, "Amount", err)
	} else if !m.Amount.IsPositive() {
		errs = errors.AppendField(errs, "Amount", errors.Wrap(errors.ErrAmount, "must be greater than zero"))
	}
	errs = errors.AppendField(errs, "Depositor", m.Depositor.Validate())
	return errs
}

var _ weave.Msg = (*ReleaseDepositMsg)(nil)

func (ReleaseDepositMsg) Path() string {
	return "termdeposit/release_deposit"
}

func (m *ReleaseDepositMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	if len(m.DepositID) == 0 {
		errs = errors.AppendField(errs, "DepositID", errors.ErrEmpty)
	}
	return errs
}

var _ weave.Msg = (*UpdateConfigurationMsg)(nil)

func (UpdateConfigurationMsg) Path() string {
	return "termdeposit/update_configuration"
}

func (m *UpdateConfigurationMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "Patch", m.Patch.Validate())
	return errs
}
