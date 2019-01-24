package validators

import (
	stderr "errors"
	"reflect"

	"github.com/iov-one/weave/errors"
)

// ABCI Response Codes
// x/update_validators reserves 40 ~ 49.
const (
	CodeEmptyDiff         = 40
	CodeWrongType         = 41
	CodeInvalidPubKey     = 42
	CodeEmptyValidatorSet = 43
	CodeInvalidPower      = 44
	CodeNotFound          = 45
)

var (
	errEmptyDiff         = stderr.New("Empty validator diff")
	errWrongType         = stderr.New("Wrong type for accounts storage")
	errInvalidPubKey     = stderr.New("Invalid public key")
	errEmptyValidatorSet = stderr.New("Empty validator set")
	errInvalidPower      = stderr.New("Power value is invalid")
	errNotFound          = stderr.New("Not found")
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

func ErrNotFound(entityName string) error {
	return errors.WithLog(entityName, errNotFound, CodeNotFound)
}
