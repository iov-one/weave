package statefix

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &ExecuteFixMsg{}, migration.NoModification)
}

var _ weave.Msg = (*ExecuteFixMsg)(nil)

func (msg *ExecuteFixMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	if len(msg.FixID) == 0 {
		errs = errors.AppendField(errs, "FixID", errors.ErrEmpty)
	}
	return errs
}

func (ExecuteFixMsg) Path() string {
	return "statefix/executefix"
}
