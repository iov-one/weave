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
	MsgList() []weave.Msg
}

func Validate(msg Msg) error {
	if len(msg.MsgList()) > MaxBatchMessages {
		//TODO: Figure out if this is the correct error
		return errors.ErrTooLarge()
	}
	return nil
}
