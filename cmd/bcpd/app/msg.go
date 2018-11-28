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
	// make sure to cover all messages defined in protobuf
	//TODO: Might be easier with reflection?
	for i, m := range msg.Messages {
		res := func() weave.Msg {
			switch t := m.GetSum().(type) {
			case *BatchMsg_Union_SendMsg:
				return t.SendMsg
			case *BatchMsg_Union_NewTokenMsg:
				return t.NewTokenMsg
			case *BatchMsg_Union_SetNameMsg:
				return t.SetNameMsg
			case *BatchMsg_Union_CreateEscrowMsg:
				return t.CreateEscrowMsg
			case *BatchMsg_Union_ReleaseEscrowMsg:
				return t.ReleaseEscrowMsg
			case *BatchMsg_Union_ReturnEscrowMsg:
				return t.ReturnEscrowMsg
			case *BatchMsg_Union_UpdateEscrowMsg:
				return t.UpdateEscrowMsg
			case *BatchMsg_Union_CreateContractMsg:
				return t.CreateContractMsg
			case *BatchMsg_Union_UpdateContractMsg:
				return t.UpdateContractMsg
			case *BatchMsg_Union_SetValidatorsMsg:
				return t.SetValidatorsMsg
			default:
				return nil
			}
		}()
		messages[i] = res.(weave.Msg)
	}
	return messages
}

func (msg *BatchMsg) Validate() error {
	return batch.Validate(msg)
}
