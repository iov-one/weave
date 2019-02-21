package batch

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

const (
	PathExecuteBatchMsg = "batch/execute"
)

type Msg interface {
	weave.Msg
	x.Validater
	MsgList() ([]weave.Msg, error)
}

func Validate(msg Msg) error {
	l, err := msg.MsgList()
	if err != nil {
		return errors.Wrap(err, "cannot retrieve batch message")
	}

	if len(l) > MaxBatchMessages {
		return errors.ErrInvalidInput.New("transaction is too large")
	}
	return nil
}
