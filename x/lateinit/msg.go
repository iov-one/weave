package lateinit

import (
	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &ExecuteInitMsg{}, migration.NoModification)
}

var _ weave.Msg = (*ExecuteInitMsg)(nil)

func (msg *ExecuteInitMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	if len(msg.InitID) == 0 {
		errs = errors.AppendField(errs, "InitID", errors.ErrEmpty)
	}
	return errs
}

func (ExecuteInitMsg) Path() string {
	return "lateinit/execute_init_msg"
}
