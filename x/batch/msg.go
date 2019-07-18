package batch

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	PathExecuteBatchMsg = "batch/execute_batch"
)

type Msg interface {
	weave.Msg
	MsgList() ([]weave.Msg, error)
}

func Validate(msg Msg) error {
	msgs, err := msg.MsgList()
	if err != nil {
		return errors.Wrap(err, "cannot retrieve batch message")
	}
	if len(msgs) > MaxBatchMessages {
		return errors.Wrapf(errors.ErrInput, "transaction is too large, max is %d", MaxBatchMessages)
	}
	return nil
}
