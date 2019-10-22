package lateinit

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &ExecutedInit{}, migration.NoModification)
}

func (ef *ExecutedInit) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", ef.Metadata.Validate())
	if len(ef.InitID) == 0 {
		errs = errors.AppendField(errs, "InitID", errors.ErrEmpty)
	}
	return errs
}

func NewExecutedInitBucket() orm.ModelBucket {
	b := orm.NewModelBucket("execinit", &ExecutedInit{})
	return migration.NewModelBucket("lateinit", b)
}
