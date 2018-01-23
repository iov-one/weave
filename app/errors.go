package app

import (
	"fmt"

	"github.com/confio/weave/errors"
)

// ABCI Response Codes
// Base SDK reserves 0 ~ 99. App uses 10 ~ 19
const (
	CodeNoSuchPath uint32 = 10
)

var (
	errNoSuchPath = fmt.Errorf("Path not registered")
)

func ErrNoSuchPath(path string) error {
	return errors.WithLog(path, errNoSuchPath, CodeNoSuchPath)
}
func IsNoSuchPathErr(err error) bool {
	return errors.IsSameError(errNoSuchPath, err)
}
