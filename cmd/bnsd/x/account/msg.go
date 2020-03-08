package account

import (
	"crypto/sha256"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &UpdateConfigurationMsg{}, migration.NoModification)

	migration.MustRegister(1, &RegisterDomainMsg{}, migration.NoModification)
	migration.MustRegister(1, &TransferDomainMsg{}, migration.NoModification)
	migration.MustRegister(1, &RenewDomainMsg{}, migration.NoModification)
	migration.MustRegister(1, &DeleteDomainMsg{}, migration.NoModification)
	migration.MustRegister(1, &ReplaceAccountMsgFeesMsg{}, migration.NoModification)
	migration.MustRegister(1, &FlushDomainMsg{}, migration.NoModification)

	migration.MustRegister(1, &RegisterAccountMsg{}, migration.NoModification)
	migration.MustRegister(1, &TransferAccountMsg{}, migration.NoModification)
	migration.MustRegister(1, &ReplaceAccountTargetsMsg{}, migration.NoModification)
	migration.MustRegister(1, &DeleteAccountMsg{}, migration.NoModification)
	migration.MustRegister(1, &RenewAccountMsg{}, migration.NoModification)
	migration.MustRegister(1, &AddAccountCertificateMsg{}, migration.NoModification)
	migration.MustRegister(1, &DeleteAccountCertificateMsg{}, migration.NoModification)
}

var _ weave.Msg = (*UpdateConfigurationMsg)(nil)

func (UpdateConfigurationMsg) Path() string {
	return "account/update_configuration"
}

func (msg *UpdateConfigurationMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Patch", msg.Patch.Validate())
	return errs
}

var _ weave.Msg = (*RegisterDomainMsg)(nil)

func (RegisterDomainMsg) Path() string {
	return "account/register_domain"
}

func (msg *RegisterDomainMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Admin", msg.Admin.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	errs = errors.AppendField(errs, "MsgFees", validateMsgFees(msg.MsgFees))
	if msg.Broker != nil {
		errs = errors.AppendField(errs, "Broker", msg.Broker.Validate())
	}
	return errs
}

var _ weave.Msg = (*TransferDomainMsg)(nil)

func (TransferDomainMsg) Path() string {
	return "account/transfer_domain"
}

func (msg *TransferDomainMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	errs = errors.AppendField(errs, "NewOwner", msg.NewAdmin.Validate())
	return errs
}

var _ weave.Msg = (*RenewDomainMsg)(nil)

func (RenewDomainMsg) Path() string {
	return "account/renew_domain"
}

func (msg *RenewDomainMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	return errs
}

var _ weave.Msg = (*DeleteDomainMsg)(nil)

func (DeleteDomainMsg) Path() string {
	return "account/delete_domain"
}

func (msg *DeleteDomainMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	return errs
}

var _ weave.Msg = (*RegisterAccountMsg)(nil)

func (RegisterAccountMsg) Path() string {
	return "account/register_account"
}

func (msg *RegisterAccountMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	if len(msg.Owner) != 0 {
		errs = errors.AppendField(errs, "Owner", msg.Owner.Validate())
	}
	// NewTargets cannot be validated here because it requires Configuration instance
	if msg.Broker != nil {
		errs = errors.AppendField(errs, "Broker", msg.Broker.Validate())
	}
	return errs
}

var _ weave.Msg = (*TransferAccountMsg)(nil)

func (TransferAccountMsg) Path() string {
	return "account/transfer_account"
}

func (msg *TransferAccountMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	if msg.NewOwner != nil {
		errs = errors.AppendField(errs, "NewOwner", msg.NewOwner.Validate())
	}
	return errs
}

var _ weave.Msg = (*ReplaceAccountTargetsMsg)(nil)

func (ReplaceAccountTargetsMsg) Path() string {
	return "account/replace_account_targets"
}

func (msg *ReplaceAccountTargetsMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	// NewTargets cannot be validated here because it requires Configuration instance
	return errs
}

var _ weave.Msg = (*DeleteAccountMsg)(nil)

func (DeleteAccountMsg) Path() string {
	return "account/delete_account"
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
	return "account/delete_all_accounts"
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

func (RenewAccountMsg) Path() string {
	return "account/renew_account"
}

func (msg *RenewAccountMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	return errs
}

func (AddAccountCertificateMsg) Path() string {
	return "account/add_account_certificate"
}

func (msg *AddAccountCertificateMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	switch n := len(msg.Certificate); {
	case n == 0:
		errs = errors.AppendField(errs, "Certificate", errors.ErrEmpty)
	case n > 10240:
		errs = errors.AppendField(errs, "Certificate", errors.Wrap(errors.ErrInput, "too big"))
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
	return nil
}

var _ weave.Msg = (*ReplaceAccountMsgFeesMsg)(nil)

func (ReplaceAccountMsgFeesMsg) Path() string {
	return "account/replace_account_msg_fees"
}

func (msg *ReplaceAccountMsgFeesMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	errs = errors.AppendField(errs, "NewMsgFees", validateMsgFees(msg.NewMsgFees))
	return errs
}

var _ weave.Msg = (*DeleteAccountCertificateMsg)(nil)

func (DeleteAccountCertificateMsg) Path() string {
	return "account/delete_account_certificate"
}

func (msg *DeleteAccountCertificateMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	errs = errors.AppendField(errs, "Domain", validateDomain(msg.Domain))
	if len(msg.CertificateHash) != sha256.Size {
		errs = errors.AppendField(errs, "CertificateHash", errors.Wrapf(errors.ErrInput, "invalid length %d", len(msg.CertificateHash)))
	}
	return errs
}
