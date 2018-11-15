package batch

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

const (
	PathExecuteBatchMsg = "batch/execute"
)

var _ weave.Msg = (*ExecuteBatchMsg)(nil)

func (*ExecuteBatchMsg) Path() string {
	return PathExecuteBatchMsg
}

func (m *ExecuteBatchMsg) Validate() error {
	for _, v := range m.Messages {
		if err := v.Msg.(x.Validater).Validate(); err != nil {
			return err
		}
	}
	return nil
}
