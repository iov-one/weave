package escrow

import (
	"fmt"

	"github.com/iov-one/weave/errors"
)

// ABCI Response Codes
// bov takes 1000-1100
// escrow takes 1010-1020
const (
	CodeNoEscrow         = 1010
	CodeMissingCondition = 1011
	CodeInvalidCondition = 1012
	CodeInvalidMetadata  = 1013
	CodeInvalidHeight    = 1014

	// CodeInvalidIndex  = 1001
	// CodeInvalidWallet = 1002
)

var (
	errMissingArbiter       = fmt.Errorf("Missing Arbiter")
	errMissingSender        = fmt.Errorf("Missing Sender")
	errMissingRecipient     = fmt.Errorf("Missing Recipient")
	errMissingAllConditions = fmt.Errorf("Missing All Conditions")

	errInvalidMemo     = fmt.Errorf("Memo field too long")
	errInvalidTimeout  = fmt.Errorf("Invalid Timeout")
	errInvalidEscrowID = fmt.Errorf("Invalid Escrow ID")

	errNoSuchEscrow = fmt.Errorf("No Escrow with this ID")

	errEscrowExpired    = fmt.Errorf("Escrow already expired")
	errEscrowNotExpired = fmt.Errorf("Escrow not yet expired")

	// errInvalidIndex      = fmt.Errorf("Cannot calculate index")
	// errInvalidWalletName = fmt.Errorf("Invalid name for a wallet")
	// errChangeWalletName  = fmt.Errorf("Wallet already has a name")
	// errNoSuchWallet      = fmt.Errorf("No wallet exists with this address")
)

func ErrMissingArbiter() error {
	return errors.WithCode(errMissingArbiter, CodeMissingCondition)
}
func ErrMissingSender() error {
	return errors.WithCode(errMissingSender, CodeMissingCondition)
}
func ErrMissingRecipient() error {
	return errors.WithCode(errMissingRecipient, CodeMissingCondition)
}
func ErrMissingAllConditions() error {
	return errors.WithCode(errMissingAllConditions, CodeMissingCondition)
}
func IsMissingConditionErr(err error) bool {
	return errors.HasErrorCode(err, CodeMissingCondition)
}

func ErrInvalidCondition(perm []byte) error {
	return errors.ErrUnrecognizedCondition(perm)
}
func IsInvalidConditionErr(err error) bool {
	return errors.IsUnrecognizedConditionErr(err)
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

func ErrEscrowExpired(timeout int64) error {
	msg := fmt.Sprintf("%d", timeout)
	return errors.WithLog(msg, errEscrowExpired, CodeInvalidHeight)
}
func ErrEscrowNotExpired(timeout int64) error {
	msg := fmt.Sprintf("%d", timeout)
	return errors.WithLog(msg, errEscrowNotExpired, CodeInvalidHeight)
}
