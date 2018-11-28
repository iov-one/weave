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
		return errors.Wrap(err)
	}

	if len(l) > MaxBatchMessages {
		//TODO: Figure out if this is the correct error
		return errors.ErrTooLarge()
	}
	return nil
}
