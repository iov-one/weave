package msgfee

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
)

// FeeDecorator implements a decorator that for each processed transaction
// attach an additional fee to the result. Each fee is declared per
// transaction type. If fee is not set (zero value) then this decorator does
// not increase the required fee value.
// Additional fee is attached to only those transaction results that represent
// a success.
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

func (d *FeeDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	res, err := next.Check(ctx, store, tx)
	if err != nil {
		return nil, err
	}

	fee, err := txFee(d.bucket, store, tx)
	if err != nil {
		return nil, err
	}
	if !coin.IsEmpty(fee) {
		total, err := res.RequiredFee.Add(*fee)
		if err != nil {
			return nil, errors.Wrap(err, "cannot apply message type fee")
		}
		res.RequiredFee = total
	}
	return res, nil
}

func (d *FeeDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	res, err := next.Deliver(ctx, store, tx)
	if err != nil {
		return nil, err
	}

	fee, err := txFee(d.bucket, store, tx)
	if err != nil {
		return nil, err
	}
	if !coin.IsEmpty(fee) {
		total, err := res.RequiredFee.Add(*fee)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot apply message type fee to %v", res.RequiredFee)
		}
		res.RequiredFee = total
	}
	return res, nil
}

// txFee returns the fee value for a given transaction as configured in the store.
func txFee(bucket *MsgFeeBucket, store weave.KVStore, tx weave.Tx) (*coin.Coin, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, errors.Wrap(err, "cannot get message")
	}
	fee, err := bucket.MessageFee(store, msg.Path())
	if err != nil {
		return nil, errors.Wrap(err, "cannot get fee")
	}
	return fee, nil
}
