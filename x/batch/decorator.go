/*
Package batch provides batch transaction support
middleware to support multiple operations in one
transaction
*/
package batch

import (
	"strings"

	"github.com/iov-one/weave"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/common"
)

//----------------- Decorator ----------------
//
// This is just a binding from the functionality into the
// Application stack, not much business logic here.

// Decorator iterates through batch transaction messages and passes them down the stack
type Decorator struct {
}

var _ weave.Decorator = Decorator{}

// NewDecorator returns a batch transaction decorator
func NewDecorator() Decorator {
	return Decorator{}
}

type BatchTx struct {
	weave.Tx
	Msg weave.Msg
}

func (tx *BatchTx) GetMsg() (weave.Msg, error) {
	return tx.Msg, nil
}

// Check iterates through messages in a batch transaction and passes them
// down the stack
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Checker) (res weave.CheckResult, err error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return
	}

	if batchMsg, ok := msg.(*ExecuteBatchMsg); ok {
		checks := make([]weave.CheckResult, len(batchMsg.Messages))
		for i, msg := range batchMsg.Messages {
			checks[i], err = next.Check(ctx, store, &BatchTx{Tx: tx, Msg: msg.GetMsg().(weave.Msg)})
			if err != nil {
				return
			}
		}
		res = d.combineChecks(checks)
		return
	}

	return next.Check(ctx, store, tx)
}

// combines all data bytes as a go-amino array.
// joins all log messages with \n
func (*Decorator) combineChecks(checks []weave.CheckResult) weave.CheckResult {
	datas := make([][]byte, len(checks))
	logs := make([]string, len(checks))
	var allocated, payments int64
	for i, r := range checks {
		datas[i] = r.Data
		logs[i] = r.Log
		allocated += r.GasAllocated
		payments += r.GasPayment
	}
	return weave.CheckResult{
		Data:         amino.MustMarshalBinary(datas),
		Log:          strings.Join(logs, "\n"),
		GasAllocated: allocated,
		GasPayment:   payments,
	}
}

// Deliver iterates through messages in a batch transaction and passes them
// down the stack
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (res weave.DeliverResult, err error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return
	}

	if batchMsg, ok := msg.(*ExecuteBatchMsg); ok {
		delivers := make([]weave.DeliverResult, len(batchMsg.Messages))
		for i, msg := range batchMsg.Messages {
			delivers[i], err = next.Deliver(ctx, store, &BatchTx{Tx: tx, Msg: msg.GetMsg().(weave.Msg)})
			if err != nil {
				return
			}
		}
		res = d.combineDelivers(delivers)
		return
	}

	return next.Deliver(ctx, store, tx)
}

// combines all data bytes as a go-amino array.
// joins all log messages with \n
func (*Decorator) combineDelivers(delivers []weave.DeliverResult) weave.DeliverResult {
	datas := make([][]byte, len(delivers))
	logs := make([]string, len(delivers))
	var payments int64
	var diffs []types.ValidatorUpdate
	var tags []common.KVPair
	for i, r := range delivers {
		datas[i] = r.Data
		logs[i] = r.Log
		payments += r.GasUsed
		if len(r.Diff) > 0 {
			diffs = append(diffs, r.Diff...)
		}
		if len(r.Tags) > 0 {
			tags = append(tags, r.Tags...)
		}
	}
	return weave.DeliverResult{
		Data:    amino.MustMarshalBinary(datas),
		Log:     strings.Join(logs, "\n"),
		GasUsed: payments,
		Diff:    diffs,
		Tags:    tags,
	}
}
