package app

import (
	"github.com/iov-one/weave/x/batch"
	"github.com/iov-one/weave"
)

var _ batch.Msg = (*BatchMsg)(nil)


func(*BatchMsg) Path() string {
	return batch.PathExecuteBatchMsg
}

func(msg *BatchMsg) MsgList() []weave.Msg {
	messages := make([]weave.Msg, len(msg.Messages))
	for _, m :=  range msg.Messages {
		messages = append(messages, m.GetSum().(weave.Msg))
	}
	return messages
}