package msgfee

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
)

type FeeDecorator struct {
	bucket *MsgFeeBucket
}

var _ weave.Decorator = (*FeeDecorator)(nil)

// NewFeeDecorator returns a decorator that is upading the cost of processing
// each message according to the fee configured per each message type.
func NewFeeDecorator() *FeeDecorator {
	return &FeeDecorator{
		bucket: NewMsgFeeBucket(),
	}
}

func (d *FeeDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (weave.CheckResult, error) {
	return next.Check(ctx, store, tx)
}

func (d *FeeDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (weave.DeliverResult, error) {
	res, err := next.Deliver(ctx, store, tx)
	if err != nil {
		return res, err
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return res, errors.Wrap(err, "cannot get message")
	}
	fee, err := d.bucket.MessageFee(store, msg.Path())
	if err != nil {
		return res, errors.Wrap(err, "cannot get fee")
	}

	if !coin.IsEmpty(fee) {
		total, err := res.RequiredFee.Add(*fee)
		if err != nil {
			return res, errors.Wrap(err, "cannot apply message type fee")
		}
		res.RequiredFee = total
	}
	return res, nil
}
