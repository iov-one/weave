package msgfee

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
)

// AntispamFeeDecorator implements a decorator that for each processed transaction
// asks for a minimal fee. The fee is defined globally in the app.
// If fee is not set (zero value) or is less than the fee already asked for the transaction
// then this decorator is a noop.
type AntispamFeeDecorator struct {
	fee coin.Coin
}

var _ weave.Decorator = (*AntispamFeeDecorator)(nil)

// NewAntispamFeeDecorator returns an AntispamFeeDecorator
func NewAntispamFeeDecorator(fee coin.Coin) *AntispamFeeDecorator {
	if fee.IsZero() {
		// Returning a nil is a way to inform weave to ignore this
		// decorator. Instead of checking during the runtime if the fee
		// is zero, we can create a no operation (ignored) decorator
		// instance instead.
		return nil
	}
	return &AntispamFeeDecorator{fee: fee}
}

func (d *AntispamFeeDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	res, err := next.Check(ctx, store, tx)
	if d == nil { // Since NewAntispamFeeDecorator can return nil, let's be graceful here
		return res, err
	}
	if err != nil {
		return nil, err
	}
	if res.RequiredFee.IsZero() {
		return nil, errors.Wrap(errors.ErrEmpty, "required must not be zero")
	}
	if !res.RequiredFee.SameType(d.fee) {
		return nil, errors.Wrapf(errors.ErrCurrency,
			"antispam fee has the wrong type: expected %q, got %q", d.fee.Ticker, res.RequiredFee.Ticker)
	}
	if !res.RequiredFee.IsGTE(d.fee) {
		res.RequiredFee = d.fee
	}
	return res, nil
}

func (d *AntispamFeeDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	return next.Deliver(ctx, store, tx)
}
