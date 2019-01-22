package validators

import (
	"fmt"
	"reflect"

	"github.com/iov-one/weave/errors"
)

// ABCI Response Codes
// x/update_validators reserves 40 ~ 49.
const (
	CodeEmptyDiff         uint32 = 40
	CodeWrongType                = 41
	CodeInvalidPubKey            = 42
	CodeEmptyValidatorSet        = 43
)

var (
	errEmptyDiff         = fmt.Errorf("Empty validator diff")
	errWrongType         = fmt.Errorf("Wrong type for accounts storage")
	errInvalidPubKey     = fmt.Errorf("Invalid public key")
	errEmptyValidatorSet = fmt.Errorf("Empty validator set")
)

func ErrEmptyDiff() error {
	return errors.WithCode(errEmptyDiff, CodeEmptyDiff)
}

func ErrWrongType(t interface{}) error {
	typeName := ""
	if t != nil {
		typeName = reflect.TypeOf(t).Name()
	}
	return errors.WithLog(typeName, errWrongType, CodeWrongType)
}
