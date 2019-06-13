/*
Package batch provides batch transaction support
middleware to support multiple operations in one
transaction
*/
package batch

import (
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/tendermint/tendermint/libs/common"
)

const MaxBatchMessages = 10

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
	msg weave.Msg
}

func (tx *BatchTx) GetMsg() (weave.Msg, error) {
	return tx.msg, nil
}

// Check iterates through messages in a batch transaction and passes them
// down the stack
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}

	batchMsg, ok := msg.(Msg)
	if !ok {
		return next.Check(ctx, store, tx)
	}

	if err = batchMsg.Validate(); err != nil {
		return nil, err
	}

	msgList, _ := batchMsg.MsgList()

	checks := make([]*weave.CheckResult, len(msgList))
	for i, msg := range msgList {
		checks[i], err = next.Check(ctx, store, &BatchTx{Tx: tx, msg: msg})
		if err != nil {
			return nil, err
		}
	}
	return d.combineChecks(checks)
}

// combines all data bytes as protobuf.
// joins all log messages with \n
func (*Decorator) combineChecks(checks []*weave.CheckResult) (*weave.CheckResult, error) {
	datas := make([][]byte, len(checks))
	logs := make([]string, len(checks))
	var allocated, payments int64
	var required coin.Coin
	var err error
	for i, r := range checks {
		datas[i] = r.Data
		logs[i] = r.Log
		allocated += r.GasAllocated
		payments += r.GasPayment
		if required.IsZero() {
			required = r.RequiredFee
		} else if !r.RequiredFee.IsZero() {
			required, err = required.Add(r.RequiredFee)
			if err != nil {
				return nil, err
			}
		}
	}

	data, _ := (&ByteArrayList{Elements: datas}).Marshal()

	return &weave.CheckResult{
		Data:         data,
		Log:          strings.Join(logs, "\n"),
		GasAllocated: allocated,
		GasPayment:   payments,
		RequiredFee:  required,
	}, nil
}

// Deliver iterates through messages in a batch transaction and passes them
// down the stack
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}

	batchMsg, ok := msg.(Msg)
	if !ok {
		return next.Deliver(ctx, store, tx)
	}

	if err = batchMsg.Validate(); err != nil {
		return nil, err
	}

	msgList, _ := batchMsg.MsgList()

	delivers := make([]*weave.DeliverResult, len(msgList))
	for i, msg := range msgList {
		delivers[i], err = next.Deliver(ctx, store, &BatchTx{Tx: tx, msg: msg})
		if err != nil {
			return nil, err
		}
	}
	return d.combineDelivers(delivers)
}

// combines all data bytes as protobuf.
// joins all log messages with \n
func (*Decorator) combineDelivers(delivers []*weave.DeliverResult) (*weave.DeliverResult, error) {
	datas := make([][]byte, len(delivers))
	logs := make([]string, len(delivers))
	var payments int64
	var diffs []weave.ValidatorUpdate
	var tags []common.KVPair
	var required coin.Coin
	var err error
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
		if required.IsZero() {
			required = r.RequiredFee
		} else if !r.RequiredFee.IsZero() {
			required, err = required.Add(r.RequiredFee)
			if err != nil {
				return nil, err
			}
		}
	}

	data, _ := (&ByteArrayList{Elements: datas}).Marshal()
	log := strings.Join(logs, "\n")

	return &weave.DeliverResult{
		Data:    data,
		Log:     log,
		GasUsed: payments,
		Diff:    diffs,
		// https://github.com/iov-one/weave/pull/188#discussion_r234531097
		// but I couldn't find a place where, so need to figure it out
		Tags:        tags,
		RequiredFee: required,
	}, nil
}
