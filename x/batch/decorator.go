/*
Package batch provides batch transaction support
middleware to support multiple operations in one
transaction
*/
package batch

import (
	"github.com/iov-one/weave"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/common"
)

const maxBatcMessages = 10

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
	next weave.Checker) (weave.CheckResult, error) {
	var res weave.CheckResult
	var err error
	msg, err := tx.GetMsg()
	if err != nil {
		return res, err
	}

	if batchMsg, ok := msg.(Msg); ok {
		if err = batchMsg.Validate(); err != nil {
			return res, err
		}

		msgList := batchMsg.MsgList()

		checks := make([]weave.CheckResult, len(msgList))
		for i, msg := range msgList {
			checks[i], err = next.Check(ctx, store, &BatchTx{Tx: tx, Msg: msg})
			if err != nil {
				return res, err
			}
		}
		res = d.combineChecks(checks)
		return res, err
	}

	return next.Check(ctx, store, tx)
}

// combines all data bytes as a go-amino array.
// joins all log messages with \n
func (*Decorator) combineChecks(checks []weave.CheckResult) weave.CheckResult {
	datas := make([][]byte, len(checks))
	logs := make([][]byte, len(checks))
	var allocated, payments int64
	for i, r := range checks {
		datas[i] = r.Data
		//TODO: Is this good enough conversion? Should be, in theory
		logs[i] = []byte(r.Log)
		allocated += r.GasAllocated
		payments += r.GasPayment
	}

	data, _ := (&ByteArrayList{Elements: datas}).Marshal()
	log, _ := (&ByteArrayList{Elements: logs}).Marshal()

	return weave.CheckResult{
		Data: data,
		//TODO: Is this good enough conversion? Should be, in theory
		Log:          string(log),
		GasAllocated: allocated,
		GasPayment:   payments,
	}
}

// Deliver iterates through messages in a batch transaction and passes them
// down the stack
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	var err error

	msg, err := tx.GetMsg()
	if err != nil {
		return res, err
	}

	if batchMsg, ok := msg.(Msg); ok {
		if err = batchMsg.Validate(); err != nil {
			return res, err
		}

		msgList := batchMsg.MsgList()

		delivers := make([]weave.DeliverResult, len(msgList))
		for i, msg := range msgList {
			delivers[i], err = next.Deliver(ctx, store, &BatchTx{Tx: tx, Msg: msg})
			if err != nil {
				return res, err
			}
		}
		res = d.combineDelivers(delivers)
		return res, err
	}

	return next.Deliver(ctx, store, tx)
}

// combines all data bytes as a go-amino array.
// joins all log messages with \n
func (*Decorator) combineDelivers(delivers []weave.DeliverResult) weave.DeliverResult {
	datas := make([][]byte, len(delivers))
	logs := make([][]byte, len(delivers))
	var payments int64
	var diffs []types.ValidatorUpdate
	var tags []common.KVPair
	for i, r := range delivers {
		datas[i] = r.Data
		//TODO: Is this good enough conversion? Should be, in theory
		logs[i] = []byte(r.Log)
		payments += r.GasUsed
		if len(r.Diff) > 0 {
			diffs = append(diffs, r.Diff...)
		}
		if len(r.Tags) > 0 {
			tags = append(tags, r.Tags...)
		}
	}

	data, _ := (&ByteArrayList{Elements: datas}).Marshal()
	log, _ := (&ByteArrayList{Elements: logs}).Marshal()

	return weave.DeliverResult{
		Data: data,
		//TODO: Is this good enough?
		Log:     string(log),
		GasUsed: payments,
		Diff:    diffs,
		//TODO: Per Ethan's comment this is sorted
		// https://github.com/iov-one/weave/pull/188#discussion_r234531097
		// but I couldn't find a place where, so need to figure it out
		Tags: tags,
	}
}
