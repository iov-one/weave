package namecoin

import (
	"fmt"

	"github.com/confio/weave/errors"
)

// ABCI Response Codes
// bov takes 1000-1100
// namecoin takes 1000-1010
const (
	CodeInvalidToken = 1000
	CodeInvalidIndex = 1001
)

var (
	errInvalidTokenName = fmt.Errorf("Invalid token name")
	errInvalidSigFigs   = fmt.Errorf("Invalid significant figures")
	errInvalidIndex     = fmt.Errorf("Cannot calculate index")
)

func ErrInvalidTokenName(name string) error {
	return errors.WithLog(name, errInvalidTokenName, CodeInvalidToken)
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
