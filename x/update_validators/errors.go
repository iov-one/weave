package update_validators

import (
	"fmt"
	"reflect"

	"github.com/confio/weave/errors"
)

// ABCI Response Codes
// x/update_validators reserves 40 ~ 49.
const (
	CodeEmptyDiff    uint32 = 40
	CodeWrongType           = 41
	CodeNoPermission        = 42
)

var (
	errEmptyDiff    = fmt.Errorf("Empty validator diff")
	errWrongType    = fmt.Errorf("Wrong type for accounts storage")
	errUnauthorized = fmt.Errorf("Not authorized to perform this transation")
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

func ErrUnauthorized(t string) error {
	return errors.WithLog(t, errUnauthorized, CodeNoPermission)
}
