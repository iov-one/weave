package multisig

import (
	"fmt"

	"github.com/iov-one/weave/errors"
)

// ABCI Response Codes
// multisig takes 1030-1031
const (
	CodeMissingSigs       = 1030
	CodeInvalidThreshold  = 1031
	CodeContractDuplicate = 1032
)

var (
	errMissingSigs = fmt.Errorf("Missing sigs")

	errInvalidThreshold = fmt.Errorf("Activation threshold must be lower than or equal to the number of sigs")

	errContractDuplicate = fmt.Errorf("Contract already exists")
)

func ErrMissingSigs() error {
	return errors.WithCode(errMissingSigs, CodeMissingSigs)
}
func IsMissingSigsErr(err error) bool {
	return errors.HasErrorCode(err, CodeMissingSigs)
}

func ErrInvalidActivationThreshold() error {
	return errors.WithCode(errInvalidThreshold, CodeInvalidThreshold)
}
func ErrInvalidChangeThreshold() error {
	return errors.WithCode(errInvalidThreshold, CodeInvalidThreshold)
}
func IsInvalidThresholdErr(err error) bool {
	return errors.HasErrorCode(err, CodeInvalidThreshold)
}

func ErrContractDuplicate(contract []byte) error {
	msg := fmt.Sprintf("author=%X", contract)
	return errors.WithLog(msg, errContractDuplicate, CodeContractDuplicate)
}
func IsContractDuplicatedErr(err error) bool {
	return errors.HasErrorCode(err, CodeContractDuplicate)
}
