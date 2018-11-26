package app

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x/batch"
)

var _ batch.Msg = (*BatchMsg)(nil)

func (*BatchMsg) Path() string {
	return batch.PathExecuteBatchMsg
}

func (msg *BatchMsg) MsgList() []weave.Msg {
	messages := make([]weave.Msg, len(msg.Messages))
	for _, m := range msg.Messages {
		messages = append(messages, m.GetSum().(weave.Msg))
	}
	return messages
}

func (msg *BatchMsg) Validate() error {
	return batch.Validate(msg)
}
