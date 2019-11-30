package blueaccount

import (
	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &UpdateConfigurationMsg{}, migration.NoModification)

	migration.MustRegister(1, &RegisterDomainMsg{}, migration.NoModification)
	migration.MustRegister(1, &TransferDomainMsg{}, migration.NoModification)
	migration.MustRegister(1, &RenewDomainMsg{}, migration.NoModification)
	migration.MustRegister(1, &DeleteDomainMsg{}, migration.NoModification)

	migration.MustRegister(1, &RegisterAccountMsg{}, migration.NoModification)
	migration.MustRegister(1, &TransferAccountMsg{}, migration.NoModification)
	migration.MustRegister(1, &ReplaceAccountTargetsMsg{}, migration.NoModification)
	migration.MustRegister(1, &DeleteAccountMsg{}, migration.NoModification)
	migration.MustRegister(1, &FlushDomainMsg{}, migration.NoModification)
}

var _ weave.Msg = (*UpdateConfigurationMsg)(nil)

func (UpdateConfigurationMsg) Path() string {
	return "blueaccount/update_configuration_msg"
}

func (msg *UpdateConfigurationMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Patch", msg.Patch.Validate())
	return errs
}

var _ weave.Msg = (*RegisterDomainMsg)(nil)

func (RegisterDomainMsg) Path() string {
	return "blueaccount/register_domain_msg"
}

func (msg *RegisterDomainMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	if len(msg.Owner) > 0 {
		errs = errors.AppendField(errs, "Owner", msg.Owner.Validate())
	}
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	errs = errors.AppendField(errs, "ThirdPartyToken", validateThirdPartyToken(msg.ThirdPartyToken))
	return errs
}

var _ weave.Msg = (*TransferDomainMsg)(nil)

func (TransferDomainMsg) Path() string {
	return "blueaccount/transfer_domain_msg"
}

func (msg *TransferDomainMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	errs = errors.AppendField(errs, "NewOwner", msg.NewOwner.Validate())
	return errs
}

var _ weave.Msg = (*RenewDomainMsg)(nil)

func (RenewDomainMsg) Path() string {
	return "blueaccount/renew_domain_msg"
}

func (msg *RenewDomainMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	errs = errors.AppendField(errs, "ThirdPartyToken", validateThirdPartyToken(msg.ThirdPartyToken))
	return errs
}

var _ weave.Msg = (*DeleteDomainMsg)(nil)

func (DeleteDomainMsg) Path() string {
	return "blueaccount/delete_domain_msg"
}

func (msg *DeleteDomainMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	return errs
}

var _ weave.Msg = (*RegisterAccountMsg)(nil)

func (RegisterAccountMsg) Path() string {
	return "blueaccount/register_account_msg"
}

func (msg *RegisterAccountMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	if len(msg.Owner) != 0 {
		errs = errors.AppendField(errs, "Owner", msg.Owner.Validate())
	}
	errs = errors.AppendField(errs, "Targets", validateTargets(msg.Targets))
	errs = errors.AppendField(errs, "ThirdPartyToken", validateThirdPartyToken(msg.ThirdPartyToken))
	return errs
}

var _ weave.Msg = (*TransferAccountMsg)(nil)

func (TransferAccountMsg) Path() string {
	return "blueaccount/transfer_account_msg"
}

var _ weave.Msg = (*RenewAccountMsg)(nil)

func (RenewAccountMsg) Path() string {
	return "blueaccount/renew_account_msg"
}

func (msg *RenewAccountMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	return errs
}

func (msg *TransferAccountMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	errs = errors.AppendField(errs, "NewOwner", msg.NewOwner.Validate())
	return errs
}

var _ weave.Msg = (*ReplaceAccountTargetsMsg)(nil)

func (ReplaceAccountTargetsMsg) Path() string {
	return "blueaccount/replace_account_targets_msg"
}

func (msg *ReplaceAccountTargetsMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	errs = errors.AppendField(errs, "NewTargets", validateTargets(msg.NewTargets))
	return errs
}

var _ weave.Msg = (*DeleteAccountMsg)(nil)

func (DeleteAccountMsg) Path() string {
	return "blueaccount/delete_account_msg"
}

func (msg *DeleteAccountMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	// Domain validation rules are dynamically set via configuration and
	// cannot be enforced here.
	if len(msg.Domain) == 0 {
		errs = errors.AppendField(errs, "Domain", errors.ErrEmpty)
	}
	return errs
}

var _ weave.Msg = (*FlushDomainMsg)(nil)

func (FlushDomainMsg) Path() string {
	return "blueaccount/delete_all_accounts_msg"
}

func (msg *FlushDomainMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	// Domain validation rules are dynamically set via configuration and
	// cannot be enforced here.
	if len(msg.Domain) == 0 {
		errs = errors.AppendField(errs, "Domain", errors.ErrEmpty)
	}
	return errs
}

// validateDomain returns an error if provided domain string is not acceptable.
// Domain validation rules are dynamically set via configuration and cannot be
// fully enforced by a function that does not have an access to the database.
func validateDomain(domain string) error {
	if len(domain) == 0 {
		return errors.ErrEmpty
	}
	// iov is not an acceptable domain because it is reserved by the Red
	// Account functionality.
	if domain == "iov" {
		return errors.Wrap(errors.ErrInput, `"iov" is not an acceptable domain`)
	}
	return nil
}

// validateThirdPartyToken returns an error if provided token is not valid.
func validateThirdPartyToken(token []byte) error {
	const maxLen = 64
	if len(token) > maxLen {
		return errors.Wrapf(errors.ErrInput, "must not be longer than %d characters", maxLen)
	}
	return nil
}
