package cash

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/x"
)

//----------------- FeeDecorator ----------------
//
// This is just a binding from the functionality into the
// Application stack, not much business logic here.

// FeeDecorator ensures that the fee can be deducted from
// the account. All deducted fees are send to the collector,
// which can be set to an address controlled by another
// extension ("smart contract").
// Collector address is configured via gconf package.
//
// Minimal fee is configured via gconf package. If minimal is zero, no fees
// required, but will speed processing. If a currency is set on minimal fee,
// then all fees must be paid in that currency
//
// It uses auth to verify the sender
type FeeDecorator struct {
	auth    x.Authenticator
	control Controller
}

const (
	GconfCollectorAddress = "cash:collector_address"
	GconfMinimalFee       = "cash:minimal_fee"
)

var _ weave.Decorator = FeeDecorator{}

// NewFeeDecorator returns a FeeDecorator with the given
// minimum fee, and all collected fees going to a
// default address.
func NewFeeDecorator(auth x.Authenticator, control Controller) FeeDecorator {
	return FeeDecorator{
		auth:    auth,
		control: control,
	}
}

// Check verifies and deducts fees before calling down the stack
func (d FeeDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Checker) (weave.CheckResult, error) {

	var res weave.CheckResult
	finfo, err := d.extractFee(ctx, tx, store)
	if err != nil {
		return res, err
	}

	// if nothing returned, but no error, just move along
	fee := finfo.GetFees()
	if x.IsEmpty(fee) {
		return next.Check(ctx, store, tx)
	}

	// verify we have access to the money
	if !d.auth.HasAddress(ctx, finfo.Payer) {
		return res, errors.ErrUnauthorized
	}
	// and have enough
	collector := gconf.Address(store, GconfCollectorAddress)
	err = d.control.MoveCoins(store, finfo.Payer, collector, *fee)
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
	finfo, err := d.extractFee(ctx, tx, store)
	if err != nil {
		return res, err
	}

	// if nothing returned, but no error, just move along
	fee := finfo.GetFees()
	if x.IsEmpty(fee) {
		return next.Deliver(ctx, store, tx)
	}

	// verify we have access to the money
	if !d.auth.HasAddress(ctx, finfo.Payer) {
		return res, errors.ErrUnauthorized
	}
	// and subtract it from the account
	collector := gconf.Address(store, GconfCollectorAddress)
	err = d.control.MoveCoins(store, finfo.Payer, collector, *fee)
	if err != nil {
		return res, err
	}

	return next.Deliver(ctx, store, tx)
}

func (d FeeDecorator) extractFee(ctx weave.Context, tx weave.Tx, store weave.KVStore) (*FeeInfo, error) {
	var finfo *FeeInfo
	ftx, ok := tx.(FeeTx)
	if ok {
		payer := x.MainSigner(ctx, d.auth).Address()
		finfo = ftx.GetFees().DefaultPayer(payer)
	}

	fee := finfo.GetFees()
	if x.IsEmpty(fee) {
		minFee := gconf.Coin(store, GconfMinimalFee)
		if minFee.IsZero() {
			return finfo, nil
		}
		return nil, errors.ErrInsufficientAmount.Newf("fees %#v", &x.Coin{})
	}

	// make sure it is a valid fee (non-negative, going somewhere)
	err := finfo.Validate()
	if err != nil {
		return nil, err
	}

	cmp := gconf.Coin(store, GconfMinimalFee)
	// minimum has no currency -> accept everything
	if cmp.Ticker == "" {
		cmp.Ticker = fee.Ticker
	}
	if !fee.SameType(cmp) {
		return nil, x.ErrInvalidCurrency.Newf("%s vs fee %s", cmp.Ticker, fee.Ticker)

	}
	if !fee.IsGTE(cmp) {
		return nil, errors.ErrInsufficientAmount.Newf("fees %#v", fee)
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
