package cash

import (
	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/confio/weave/x"
)

var (
	defaultCollector = weave.NewAddress([]byte("no-fees-here"))
)

//----------------- FeeDecorator ----------------
//
// This is just a binding from the functionality into the
// Application stack, not much business logic here.

// FeeDecorator ensures that the fee can be deducted from
// the account. All deducted fees are send to the collector,
// which can be set to an address controlled by another
// extension ("smart contract").
//
// If minFee is zero, no fees required, but will
// speed processing. If a currency is set on minFee,
// then all fees must be paid in that currency
//
// It uses auth to verify the sender
type FeeDecorator struct {
	minFee    x.Coin
	auth      x.Authenticator
	collector weave.Address
}

var _ weave.Decorator = FeeDecorator{}

// NewFeeDecorator returns a FeeDecorator with the given
// minimum fee, and all collected fees going to a
// default address.
func NewFeeDecorator(auth x.Authenticator, min x.Coin) FeeDecorator {
	return FeeDecorator{
		auth:      auth,
		minFee:    min,
		collector: defaultCollector,
	}
}

// WithCollector allows you to set the collector in app setup
func (d FeeDecorator) WithCollector(addr weave.Address) FeeDecorator {
	d.collector = addr
	return d
}

// Check verifies and deducts fees before calling down the stack
func (d FeeDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Checker) (weave.CheckResult, error) {

	var res weave.CheckResult
	finfo, err := d.extractFee(ctx, tx)
	if err != nil {
		return res, err
	}

	// if nothing returned, but no error, just move along
	fee := finfo.GetFees()
	if x.IsEmpty(fee) {
		return next.Check(ctx, store, tx)
	}

	// verify we have access to the money
	if !d.auth.HasPermission(ctx, finfo.Payer) {
		return res, errors.ErrUnauthorized()
	}
	// and have enough
	err = MoveCoins(store, finfo.Payer, d.collector, *fee)
	if err != nil {
		return res, err
	}

	// now update the importance...
	paid := toPayment(*fee)
	res, err = next.Check(ctx, store, tx)
	res.GasPayment += paid
	return res, err
}

// Deliver verifies and deducts fees before calling down the stack
func (d FeeDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (weave.DeliverResult, error) {

	var res weave.DeliverResult
	finfo, err := d.extractFee(ctx, tx)
	if err != nil {
		return res, err
	}

	// if nothing returned, but no error, just move along
	fee := finfo.GetFees()
	if x.IsEmpty(fee) {
		return next.Deliver(ctx, store, tx)
	}

	// verify we have access to the money
	if !d.auth.HasPermission(ctx, finfo.Payer) {
		return res, errors.ErrUnauthorized()
	}
	// and subtract it from the account
	err = MoveCoins(store, finfo.Payer, d.collector, *fee)
	if err != nil {
		return res, err
	}

	return next.Deliver(ctx, store, tx)
}

func (d FeeDecorator) extractFee(ctx weave.Context, tx weave.Tx) (*FeeInfo, error) {
	var finfo *FeeInfo
	ftx, ok := tx.(FeeTx)
	if ok {
		payer := x.MainSigner(ctx, d.auth)
		finfo = ftx.GetFees().DefaultPayer(payer)
	}

	fee := finfo.GetFees()
	if x.IsEmpty(fee) {
		if d.minFee.IsZero() {
			return finfo, nil
		}
		return nil, ErrInsufficientFees(x.Coin{})
	}

	// make sure it is a valid fee (non-negative, going somewhere)
	err := finfo.Validate()
	if err != nil {
		return nil, err
	}

	cmp := d.minFee
	// minimum has no currency -> accept everything
	if cmp.Ticker == "" {
		cmp.Ticker = fee.Ticker
	}
	if !fee.SameType(cmp) {
		return nil, x.ErrInvalidCurrency("fee", fee.Ticker)
	}
	if !fee.IsGTE(cmp) {
		return nil, ErrInsufficientFees(*fee)
	}
	return finfo, nil
}

// toPayment calculates how much we prioritize the tx
// one point per fractional unit
func toPayment(fee x.Coin) int64 {
	base := int64(fee.Fractional)
	base += int64(fee.Whole) * int64(x.FracUnit)
	return base
}
