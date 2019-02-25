package cash

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/x"
)

/*
DynamicFeeDecorator is an enhanced version the basic FeeDecorator with
better handling of tx errors, and ability to deduct/enforce
app-specific fees.

The business logic flow is:

* If tx fee < min fee, reject with error.

* Deduct proposed fee or error.

* Run the tx.

* If tx errors, revert all tx changes and refund all but the min fee

* If tx succeeded, but requested a RequiredFee higher than paid fee,
revert all tx changes and refund all but the min fee, returning an error.

* If tx succeeded, and had paid at least RequiredFee, everything is committed
and we return success

It also embeds a checkpoint inside, so in the typical application stack:
	cash.NewFeeDecorator(authFn, ctrl),
	utils.NewSavepoint().OnDeliver(),
can be replaced by
	cash.NewDynamicFeeDecorator(authFn, ctrl),

As with FeeDecorator, all deducted fees are send to the collector,
whose address is configured via gconf package.
*/
type DynamicFeeDecorator struct {
	auth    x.Authenticator
	control FeeController
}

// FeeController is a minimal subset of the full cash.Controller
type FeeController interface {
	// MoveCoins removes funds from the source account and adds them to the
	// destination account. This operation is atomic.
	MoveCoins(store weave.KVStore, src weave.Address, dest weave.Address, amount x.Coin) error
}

// const (
// 	GconfCollectorAddress = "cash:collector_address"
// 	GconfMinimalFee       = "cash:minimal_fee"
// )

var _ weave.Decorator = DynamicFeeDecorator{}

// NewDynamicFeeDecorator returns a DynamicFeeDecorator with the given
// minimum fee, and all collected fees going to a default address.
func NewDynamicFeeDecorator(auth x.Authenticator, control FeeController) DynamicFeeDecorator {
	return DynamicFeeDecorator{
		auth:    auth,
		control: control,
	}
}

// Check verifies and deducts fees before calling down the stack
func (d DynamicFeeDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
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
		return res, errors.ErrUnauthorized.New("Fee payer signature missing")
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
func (d DynamicFeeDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
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
		return res, errors.ErrUnauthorized.New("Fee payer signature missing")
	}
	// and subtract it from the account
	collector := gconf.Address(store, GconfCollectorAddress)
	err = d.control.MoveCoins(store, finfo.Payer, collector, *fee)
	if err != nil {
		return res, err
	}

	return next.Deliver(ctx, store, tx)
}

func (d DynamicFeeDecorator) extractFee(ctx weave.Context, tx weave.Tx, store weave.KVStore) (*FeeInfo, error) {
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
