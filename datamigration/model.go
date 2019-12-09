package datamigration

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &ExecutedMigration{}, migration.NoModification)
}

func (em *ExecutedMigration) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", em.Metadata.Validate())
	return errs
}

func NewExecutedMigrationBucket() orm.ModelBucket {
	b := orm.NewModelBucket("execmig", &ExecutedMigration{})
	return migration.NewModelBucket("datamigration", b)
}
