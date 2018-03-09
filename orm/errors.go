package orm

import (
	"fmt"

	"github.com/confio/weave/errors"
)

// ABCI Response Codes
// orm reserves 10~19
const (
	CodeInvalidIndex        = 10
	CodeDuplicate           = 11
	CodeMissing             = 12
	CodeInvalidModification = 13
	CodeProgrammer          = 15
)

var (
	// errInsufficientFees  = fmt.Errorf("Insufficient fees")
	// errInsufficientFunds = fmt.Errorf("Insufficient funds")
	errInvalidIndex = fmt.Errorf("No such index")

	errUniqueConstraint = fmt.Errorf("Duplicate violates unique constraint on index")
	errRefInSet         = fmt.Errorf("Ref already in set")

	errMissingKey         = fmt.Errorf("Missing key")
	errMissingValue       = fmt.Errorf("Missing value")
	errNoRefs             = fmt.Errorf("No references")
	errRemoveUnregistered = fmt.Errorf("Cannot remove index to something that was not added")

	errModifiedPK = fmt.Errorf("Cannot modify the primary key of an object")

	errUpdateNil = fmt.Errorf("update requires at least one non-nil object")
	errBoolean   = fmt.Errorf("You have violated the rules of boolean logic")
)

func ErrInvalidIndex(reason string) error {
	return errors.WithLog(reason, errInvalidIndex, CodeInvalidIndex)
}
func IsInvalidIndexErr(err error) bool {
	return errors.IsSameError(errInvalidIndex, err)
}

func ErrUniqueConstraint(reason string) error {
	return errors.WithLog(reason, errUniqueConstraint, CodeDuplicate)
}
func IsUniqueConstraintErr(err error) bool {
	return errors.IsSameError(errUniqueConstraint, err)
}
func ErrRefInSet() error {
	return errors.WithCode(errRefInSet, CodeDuplicate)
}
func IsRefInSetErr(err error) bool {
	return errors.IsSameError(errRefInSet, err)
}

func IsMissingErr(err error) bool {
	return errors.HasErrorCode(err, CodeMissing)
}
func ErrMissingKey() error {
	return errors.WithCode(errMissingKey, CodeMissing)
}
func ErrMissingValue() error {
	return errors.WithCode(errMissingValue, CodeMissing)
}
func ErrNoRefs() error {
	return errors.WithCode(errNoRefs, CodeMissing)
}
func ErrRemoveUnregistered() error {
	return errors.WithCode(errRemoveUnregistered, CodeMissing)
}

func IsInvalidModificationErr(err error) bool {
	return errors.HasErrorCode(err, CodeInvalidModification)
}
func ErrModifiedPK() error {
	return errors.WithCode(errModifiedPK, CodeInvalidModification)
}

func IsProgammerErr(err error) bool {
	return errors.HasErrorCode(err, CodeProgrammer)
}
func ErrUpdateNil() error {
	return errors.WithCode(errUpdateNil, CodeProgrammer)
}
func ErrBoolean() error {
	return errors.WithCode(errBoolean, CodeProgrammer)
}

// func ErrEmptyAccount(addr []byte) error {
//     msg := fmt.Sprintf("%X", addr)
//     return errors.WithLog(msg, errEmptyAccount, CodeEmptyAccount)
// }
// func IsEmptyAccountErr(err error) bool {
//     return errors.IsSameError(errEmptyAccount, err)
// }
