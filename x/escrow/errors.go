package escrow

import (
	"fmt"

	"github.com/confio/weave/errors"
)

// ABCI Response Codes
// bov takes 1000-1100
// escrow takes 1010-1020
const (
	CodeMissingPermission = 1010
	CodeInvalidPermission = 1011
	CodeInvalidMetadata   = 1012
	CodeNoEscrow          = 1013

	// CodeInvalidIndex  = 1001
	// CodeInvalidWallet = 1002
)

var (
	errMissingArbiter        = fmt.Errorf("Missing Arbiter")
	errMissingSender         = fmt.Errorf("Missing Sender")
	errMissingRecipient      = fmt.Errorf("Missing Recipient")
	errMissingAllPermissions = fmt.Errorf("Missing All Permissions")

	errInvalidMemo     = fmt.Errorf("Memo field too long")
	errInvalidTimeout  = fmt.Errorf("Invalid Timeout")
	errInvalidEscrowID = fmt.Errorf("Invalid Escrow ID")

	errNoSuchEscrow = fmt.Errorf("No Escrow with this ID")

	// errInvalidIndex      = fmt.Errorf("Cannot calculate index")
	// errInvalidWalletName = fmt.Errorf("Invalid name for a wallet")
	// errChangeWalletName  = fmt.Errorf("Wallet already has a name")
	// errNoSuchWallet      = fmt.Errorf("No wallet exists with this address")
)

func ErrMissingArbiter() error {
	return errors.WithCode(errMissingArbiter, CodeMissingPermission)
}
func ErrMissingSender() error {
	return errors.WithCode(errMissingSender, CodeMissingPermission)
}
func ErrMissingRecipient() error {
	return errors.WithCode(errMissingRecipient, CodeMissingPermission)
}
func ErrMissingAllPermissions() error {
	return errors.WithCode(errMissingAllPermissions, CodeMissingPermission)
}
func IsMissingPermissionErr(err error) bool {
	return errors.HasErrorCode(err, CodeMissingPermission)
}

func ErrInvalidPermission(perm []byte) error {
	return errors.ErrUnrecognizedPermission(perm)
}
func IsInvalidPermissionErr(err error) bool {
	return errors.IsUnrecognizedPermissionErr(err)
}

func ErrInvalidMemo(memo string) error {
	return errors.WithLog(memo, errInvalidMemo, CodeInvalidMetadata)
}
func ErrInvalidTimeout(timeout int64) error {
	msg := fmt.Sprintf("%d", timeout)
	return errors.WithLog(msg, errInvalidTimeout, CodeInvalidMetadata)
}
func ErrInvalidEscrowID(id []byte) error {
	msg := "(nil)"
	if len(id) > 0 {
		msg = fmt.Sprintf("%X", id)
	}
	return errors.WithLog(msg, errInvalidEscrowID, CodeInvalidMetadata)
}
func IsInvalidMetadataErr(err error) bool {
	return errors.HasErrorCode(err, CodeInvalidMetadata)
}

func ErrNoSuchEscrow(id []byte) error {
	msg := fmt.Sprintf("%X", id)
	return errors.WithLog(msg, errNoSuchEscrow, CodeNoEscrow)
}
func IsNoSuchEscrowErr(err error) bool {
	return errors.HasErrorCode(err, CodeNoEscrow)
}
