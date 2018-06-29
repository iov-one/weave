package update_validators

import (
	"fmt"

	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/confio/weave/x"
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
	return errors.WithLog("", errEmptyDiff, CodeEmptyDiff)
}

func ErrWrongType(t string) error {
	return errors.WithLog(t, errWrongType, CodeWrongType)
}

func ErrUnauthorized(t string) error {
	return errors.WithLog(t, errUnauthorized, CodeNoPermission)
}
