/*

FeeDecorator ensures that the fee can be deducted from the account. All
deducted fees are send to the collector, which can be set to an address
controlled by another extension ("smart contract").
Collector address is configured via gconf package.

Minimal fee is configured via gconf package. If minimal is zero, no fees
required, but will speed processing. If a currency is set on minimal fee, then
all fees must be paid in that currency

It uses auth to verify the source.

*/

package cash

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

type FeeDecorator struct {
	auth x.Authenticator
	ctrl CoinMover
}

var _ weave.Decorator = FeeDecorator{}

// NewFeeDecorator returns a FeeDecorator with the given
// minimum fee, and all collected fees going to a
// default address.
func NewFeeDecorator(auth x.Authenticator, ctrl CoinMover) FeeDecorator {
	return FeeDecorator{
		auth: auth,
		ctrl: ctrl,
	}
}

// Check verifies and deducts fees before calling down the stack
func (d FeeDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	finfo, err := d.extractFee(ctx, tx, store)
	if err != nil {
		return nil, err
	}

	// if nothing returned, but no error, just move along
	fee := finfo.GetFees()
	if coin.IsEmpty(fee) {
		return next.Check(ctx, store, tx)
	}

	// verify we have access to the money
	if !d.auth.HasAddress(ctx, finfo.Payer) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "Fee payer signature missing")
	}
	// and have enough
	collector := mustLoadConf(store).CollectorAddress
	err = d.ctrl.MoveCoins(store, finfo.Payer, collector, *fee)
	if err != nil {
		return nil, err
	}

	// now update the importance...
	paid := toPayment(*fee)
	res, err := next.Check(ctx, store, tx)
	if err != nil {
		return nil, err
	}
	res.GasPayment += paid
	return res, nil
}

// Deliver verifies and deducts fees before calling down the stack
func (d FeeDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	finfo, err := d.extractFee(ctx, tx, store)
	if err != nil {
		return nil, err
	}

	// if nothing returned, but no error, just move along
	fee := finfo.GetFees()
	if coin.IsEmpty(fee) {
		return next.Deliver(ctx, store, tx)
	}

	// verify we have access to the money
	if !d.auth.HasAddress(ctx, finfo.Payer) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "Fee payer signature missing")
	}
	// and subtract it from the account
	collector := mustLoadConf(store).CollectorAddress
	err = d.ctrl.MoveCoins(store, finfo.Payer, collector, *fee)
	if err != nil {
		return nil, err
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
	if coin.IsEmpty(fee) {
		minFee := mustLoadConf(store).MinimalFee
		if minFee.IsZero() {
			return finfo, nil
		}
		return nil, errors.Wrapf(errors.ErrAmount, "fees %#v", fee)
	}

	// make sure it is a valid fee (non-negative, going somewhere)
	err := finfo.Validate()
	if err != nil {
		return nil, err
	}

	cmp := mustLoadConf(store).MinimalFee
	if cmp.IsZero() {
		return finfo, nil
	}
	if cmp.Ticker == "" {
		return nil, errors.Wrap(errors.ErrCurrency, "no ticker")
	}

	if !fee.SameType(cmp) {
		err := errors.Wrapf(errors.ErrCurrency,
			"%s vs fee %s", cmp.Ticker, fee.Ticker)
		return nil, err

	}
	if !fee.IsGTE(cmp) {
		return nil, errors.Wrapf(errors.ErrAmount, "fees %#v", fee)
	}
	return finfo, nil
}

// toPayment calculates how much we prioritize the tx
// one point per fractional unit
func toPayment(fee coin.Coin) int64 {
	base := int64(fee.Fractional)
	base += int64(fee.Whole) * int64(coin.FracUnit)
	return base
}
