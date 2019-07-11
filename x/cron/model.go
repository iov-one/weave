package cron

import (
	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &TaskResult{}, migration.NoModification)
}

var _ orm.CloneableData = (*TaskResult)(nil)

func (t *TaskResult) Validate() error {
	return nil
}

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
