package currency

import (
	stderrors "errors"
	fmt "fmt"

	"github.com/iov-one/weave/errors"
)

const (
	CodeInvalidToken = 2000

	CodeInvalidObject = 1100 // TODO: this is the same as in x/namecoin
)

var (
	errInvalidObject    = stderrors.New("Wrong object type for this bucket")
	errInvalidTokenName = stderrors.New("Invalid token name")
	errInvalidSigFigs   = stderrors.New("Invalid significant figures")
	errDuplicateToken   = stderrors.New("Token with that ticker already exists")
)

func ErrInvalidObject(obj interface{}) error {
	msg := fmt.Sprintf("%T", obj)
	return errors.WithLog(msg, errInvalidObject, CodeInvalidObject)
}

func ErrInvalidSigFigs(figs int32) error {
	msg := fmt.Sprintf("%d", figs)
	return errors.WithLog(msg, errInvalidSigFigs, CodeInvalidToken)
}

func ErrInvalidTokenName(name string) error {
	return errors.WithLog(name, errInvalidTokenName, CodeInvalidToken)
}

func ErrDuplicateToken(name string) error {
	return errors.WithLog(name, errDuplicateToken, CodeInvalidToken)
}
