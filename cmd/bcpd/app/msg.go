package app

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x/batch"
)

var _ batch.Msg = (*BatchMsg)(nil)

func (*BatchMsg) Path() string {
	return batch.PathExecuteBatchMsg
}

func (msg *BatchMsg) MsgList() ([]weave.Msg, error) {
	messages := make([]weave.Msg, len(msg.Messages))
	// make sure to cover all messages defined in protobuf
	//TODO: Might be easier with reflection?
	for i, m := range msg.Messages {
		res, err := func() (weave.Msg, error) {
			switch t := m.GetSum().(type) {
			case *BatchMsg_Union_SendMsg:
				return t.SendMsg, nil
			case *BatchMsg_Union_NewTokenMsg:
				return t.NewTokenMsg, nil
			case *BatchMsg_Union_SetNameMsg:
				return t.SetNameMsg, nil
			case *BatchMsg_Union_CreateEscrowMsg:
				return t.CreateEscrowMsg, nil
			case *BatchMsg_Union_ReleaseEscrowMsg:
				return t.ReleaseEscrowMsg, nil
			case *BatchMsg_Union_ReturnEscrowMsg:
				return t.ReturnEscrowMsg, nil
			case *BatchMsg_Union_UpdateEscrowMsg:
				return t.UpdateEscrowMsg, nil
			case *BatchMsg_Union_CreateContractMsg:
				return t.CreateContractMsg, nil
			case *BatchMsg_Union_UpdateContractMsg:
				return t.UpdateContractMsg, nil
			case *BatchMsg_Union_SetValidatorsMsg:
				return t.SetValidatorsMsg, nil
			default:
				return nil, errors.ErrUnknownTxType(t)
			}
		}()
		if err != nil {
			return messages, err
		}
		messages[i] = res.(weave.Msg)
	}
	return messages, nil
}

func (msg *BatchMsg) Validate() error {
	return batch.Validate(msg)
}
