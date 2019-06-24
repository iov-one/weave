package bnsd

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/batch"
	"github.com/iov-one/weave/x/cash"
)

// Fee sets the FeeInfo for this tx
func (tx *Tx) Fee(payer weave.Address, fee coin.Coin) {
	tx.Fees = &cash.FeeInfo{
		Payer: payer,
		Fees:  &fee}
}

// Boiler-plate needed to bridge the ExecuteBatchMsg protobuf type into something usable by the batch extension

var _ batch.Msg = (*ExecuteBatchMsg)(nil)

func (*ExecuteBatchMsg) Path() string {
	return batch.PathExecuteBatchMsg
}

func (msg *ExecuteBatchMsg) Validate() error {
	return batch.Validate(msg)
}

func (msg *ExecuteBatchMsg) MsgList() ([]weave.Msg, error) {
	var err error
	messages := make([]weave.Msg, len(msg.Messages))
	for i, m := range msg.Messages {
		messages[i], err = weave.ExtractMsgFromSum(m.GetSum())
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
}

// Boiler-plate needed to bridge the ExecuteProposalBatchMsg protobuf type into something usable by the batch extension

var _ batch.Msg = (*ExecuteProposalBatchMsg)(nil)

func (*ExecuteProposalBatchMsg) Path() string {
	return batch.PathExecuteBatchMsg
}

func (msg *ExecuteProposalBatchMsg) Validate() error {
	return batch.Validate(msg)
}

func (msg *ExecuteProposalBatchMsg) MsgList() ([]weave.Msg, error) {
	var err error
	messages := make([]weave.Msg, len(msg.Messages))
	for i, m := range msg.Messages {
		messages[i], err = weave.ExtractMsgFromSum(m.GetSum())
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
}
