package cron

import (
	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Task{}, migration.NoModification)
}

var _ orm.CloneableData = (*Task)(nil)

func (t *Task) Validate() error {
	return nil
}

func (t *Task) Copy() orm.CloneableData {
	auth := make([]weave.Address, len(t.Auth))
	copy(auth, t.Auth)

	serialized := make([]byte, len(t.Serialized))
	copy(serialized, t.Serialized)

	return &Task{
		Metadata:   t.Metadata.Copy(),
		Serialized: serialized,
		Auth:       auth,
	}
}

// NewTaskBucket returns a bucket for storing Task state.
func NewTaskBucket() orm.ModelBucket {
	b := orm.NewModelBucket("tasks", &Task{})
	return migration.NewModelBucket("cron", b)
}
