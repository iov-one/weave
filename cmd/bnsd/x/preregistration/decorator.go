package preregistration

import (
	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
)

// NewZeroFeeDecorator returns a decorator that lowers the fee for a record
// preregistration message.
func NewZeroFeeDecorator() *ZeroFeeDecorator {
	return &ZeroFeeDecorator{}
}

// ZeroFeeDecorator zero fee for a transaction that contains a single message
// of registering a preregistration record. Preregistration records can be
// inserted only by an admin, so even the antispam fee is not necessary.
type ZeroFeeDecorator struct{}

var _ weave.Decorator = (*ZeroFeeDecorator)(nil)

func (*ZeroFeeDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	res, err := next.Check(ctx, store, tx)
	if err != nil {
		return res, err
	}
	if res.RequiredFee.IsZero() {
		return res, err
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return nil, errors.Wrap(err, "cannot unpack transaction message")
	}
	if _, ok := msg.(*RegisterMsg); ok {
		// Zero the fee.
		res.RequiredFee = coin.Coin{}
	}
	return res, nil
}

func (*ZeroFeeDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	res, err := next.Deliver(ctx, store, tx)
	if err != nil {
		return res, err
	}
	if res.RequiredFee.IsZero() {
		return res, err
	}

	msg, err := tx.GetMsg()
	if err != nil {
		return nil, errors.Wrap(err, "cannot unpack transaction message")
	}
	if _, ok := msg.(*RegisterMsg); ok {
		// Zero the fee.
		res.RequiredFee = coin.Coin{}
	}
	return res, nil
}
