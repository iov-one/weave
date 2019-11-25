package datamigration

import (
	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &ExecuteMigrationMsg{}, migration.NoModification)
}

var _ weave.Msg = (*ExecuteMigrationMsg)(nil)

func (msg *ExecuteMigrationMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	if len(msg.MigrationID) == 0 {
		errs = errors.AppendField(errs, "MigrationID", errors.ErrEmpty)
	}
	return errs
}

func (ExecuteMigrationMsg) Path() string {
	return "datamigration/execute_migration_msg"
}
