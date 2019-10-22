package statefix

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &ExecutedFix{}, migration.NoModification)
}

func (ef *ExecutedFix) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", ef.Metadata.Validate())
	if len(ef.FixID) == 0 {
		errs = errors.AppendField(errs, "FixID", errors.ErrEmpty)
	}
	return errs
}

func NewExecutedFixBucket() orm.ModelBucket {
	b := orm.NewModelBucket("efix", &ExecutedFix{})
	return migration.NewModelBucket("statefix", b)
}
