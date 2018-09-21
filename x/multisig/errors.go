package multisig

import (
	"fmt"

	"github.com/iov-one/weave/errors"
)

// ABCI Response Codes
// multisig takes 1030-1040
const (
	CodeInvalidMsg             = 1030
	CodeMultisigAuthentication = 1031
)

var (
	errMissingSigs      = fmt.Errorf("Missing sigs")
	errInvalidThreshold = fmt.Errorf("Activation threshold must be lower than or equal to the number of sigs")

	errUnauthorizedMultiSig = fmt.Errorf("Multisig authentication failed")
	errContractNotFound     = fmt.Errorf("Multisig contract not found")
)

func ErrMissingSigs() error {
	return errors.WithCode(errMissingSigs, CodeInvalidMsg)
}
func ErrInvalidActivationThreshold() error {
	return errors.WithCode(errInvalidThreshold, CodeInvalidMsg)
}
func ErrInvalidChangeThreshold() error {
	return errors.WithCode(errInvalidThreshold, CodeInvalidMsg)
}
func IsInvalidMsgErr(err error) bool {
	return errors.HasErrorCode(err, CodeInvalidMsg)
}

func ErrUnauthorizedMultiSig(contract []byte) error {
	msg := fmt.Sprintf("contract=%X", contract)
	return errors.WithLog(msg, errUnauthorizedMultiSig, CodeMultisigAuthentication)
}
func ErrContractNotFound(contract []byte) error {
	msg := fmt.Sprintf("contract=%X", contract)
	return errors.WithLog(msg, errContractNotFound, CodeMultisigAuthentication)
}
func IsMultiSigAuthenticationErr(err error) bool {
	return errors.HasErrorCode(err, CodeMultisigAuthentication)
}
