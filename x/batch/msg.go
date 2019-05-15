package batch

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	PathExecuteBatchMsg = "batch/execute"
)

type Msg interface {
	weave.Msg
	MsgList() ([]weave.Msg, error)
}

func Validate(msg Msg) error {
	l, err := msg.MsgList()
	multiErr := errors.MultiAddNamed("Message", errors.Wrap(err, "cannot retrieve batch message"))

	msgNum := len(l)
	if msgNum > MaxBatchMessages {
		multiErr.AddNamed("Size", errors.Wrapf(errors.ErrInput,
			"transaction is too large, max: %d vs current: %d", MaxBatchMessages, msgNum))
	}

	if multiErr.IsEmpty() {
		return nil
	}

	return multiErr
}
