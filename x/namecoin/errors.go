package namecoin

import (
	"fmt"

	"github.com/confio/weave/errors"
)

// ABCI Response Codes
// bov takes 1000-1100
// namecoin takes 1000-1010
const (
	CodeInvalidToken  = 1000
	CodeInvalidIndex  = 1001
	CodeInvalidWallet = 1002
)

var (
	errInvalidTokenName  = fmt.Errorf("Invalid token name")
	errDuplicateToken    = fmt.Errorf("Token with that ticker already exists")
	errInvalidSigFigs    = fmt.Errorf("Invalid significant figures")
	errInvalidIndex      = fmt.Errorf("Cannot calculate index")
	errInvalidWalletName = fmt.Errorf("Invalid name for a wallet")
	errChangeWalletName  = fmt.Errorf("Wallet already has a name")
)

func ErrInvalidTokenName(name string) error {
	return errors.WithLog(name, errInvalidTokenName, CodeInvalidToken)
}
func ErrDuplicateToken(name string) error {
	return errors.WithLog(name, errDuplicateToken, CodeInvalidToken)
}
func ErrInvalidSigFigs(figs int32) error {
	msg := fmt.Sprintf("%d", figs)
	return errors.WithLog(msg, errInvalidSigFigs, CodeInvalidToken)
}
func IsInvalidToken(err error) bool {
	return errors.HasErrorCode(err, CodeInvalidToken)
}

func ErrInvalidIndex(reason string) error {
	return errors.WithLog(reason, errInvalidIndex, CodeInvalidIndex)
}
func IsInvalidIndex(err error) bool {
	return errors.HasErrorCode(err, CodeInvalidIndex)
}

func ErrChangeWalletName() error {
	return errors.WithCode(errChangeWalletName, CodeInvalidWallet)
}
func ErrInvalidWalletName(name string) error {
	return errors.WithLog(name, errInvalidWalletName, CodeInvalidWallet)
}
func IsInvalidWallet(err error) bool {
	return errors.HasErrorCode(err, CodeInvalidWallet)
}
