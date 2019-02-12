package app

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x/batch"
)

var _ batch.Msg = (*BatchMsg)(nil)

func (*BatchMsg) Path() string {
	return batch.PathExecuteBatchMsg
}

func (msg *BatchMsg) MsgList() ([]weave.Msg, error) {
	messages := make([]weave.Msg, len(msg.Messages))
	// make sure to cover all messages defined in protobuf
	for i, m := range msg.Messages {
		res, err := weave.ExtractMsgFromSum(m.GetSum())
		if err != nil {
			return messages, err
		}
		messages[i] = res
	}
	return messages, nil
}

func (msg *BatchMsg) Validate() error {
	return batch.Validate(msg)
}
