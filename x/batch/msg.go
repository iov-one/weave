package batch

import (
	"github.com/iov-one/weave"
	)

const (
	PathExecuteBatchMsg = "batch/execute"
)

type Msg interface {
	weave.Msg
	MsgList() []weave.Msg
}