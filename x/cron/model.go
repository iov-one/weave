package cron

import (
	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &TaskResult{}, migration.NoModification)
}

var _ orm.CloneableData = (*TaskResult)(nil)

func (t *TaskResult) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", t.Metadata.Validate())
	errs = errors.AppendField(errs, "ExecTime", t.ExecTime.Validate())
	if len(t.Info) > maxInfoSize {
		errs = errors.Append(errs, errors.Field("Info", errors.ErrInput, "maximum allowed length is %d", maxInfoSize))
	}
	return errs
}

const maxInfoSize = 10240

func (t *TaskResult) Copy() orm.CloneableData {
	return &TaskResult{
		Metadata:   t.Metadata.Copy(),
		Successful: t.Successful,
		Info:       t.Info,
		ExecTime:   t.ExecTime,
		ExecHeight: t.ExecHeight,
	}
}

// NewTaskResultBucket returns a bucket for storing Task results.
func NewTaskResultBucket() orm.ModelBucket {
	b := orm.NewModelBucket("trs", &TaskResult{})
	return migration.NewModelBucket("cron", b)
}

func RegisterQuery(qr weave.QueryRouter) {
	NewTaskResultBucket().Register("crontaskresults", qr)
}
