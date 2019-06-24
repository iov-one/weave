/*

DynamicFeeDecorator is an enhanced version the basic FeeDecorator with better
handling of transaction errors and ability to deduct/enforce app-specific fees.

The business logic is:
1. If a transaction fee < min fee, or a transaction fee cannot be paid, reject
   it with an error.
2. Run the transaction.
3. If a transaction processing results in an error, revert all transaction
   changes and charge only the min fee.

TODO: If a transaction succeeded, but requested a RequiredFee higher than paid
fee, revert all transaction changes and refund all but the min fee, returning
an error.

If a transaction succeeded, and at least RequiredFee was paid, everything is
committed and we return success

It also embeds a checkpoint inside, so in the typical application stack:

	cash.NewFeeDecorator(authFn, ctrl),
	utils.NewSavepoint().OnDeliver(),

can be replaced by

	cash.NewDynamicFeeDecorator(authFn, ctrl),

As with FeeDecorator, all deducted fees are send to the collector, whose
address is configured via gconf package.

*/

package cash

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

type DynamicFeeDecorator struct {
	auth x.Authenticator
	ctrl CoinMover
}

var _ weave.Decorator = DynamicFeeDecorator{}

// NewDynamicFeeDecorator returns a DynamicFeeDecorator with the given
// minimum fee, and all collected fees going to a default address.
func NewDynamicFeeDecorator(auth x.Authenticator, ctrl Controller) DynamicFeeDecorator {
	return DynamicFeeDecorator{
		auth: auth,
		ctrl: ctrl,
	}
}

// Check verifies and deducts fees before calling down the stack
func (d DynamicFeeDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (cres *weave.CheckResult, cerr error) {
	fee, payer, cache, err := d.prepare(ctx, store, tx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot prepare")
	}

	defer func() {
		if cerr == nil {
			// If we cannot write the cache, then we return error
			// here means that nothing got committed. This means
			// that no change is persisted and the whole check
			// failed.
			if err := cache.Write(); err != nil {
				cache.Discard()
				cres = nil
				cerr = err
			} else {
				cres.GasPayment += toPayment(fee)
			}
		} else {
			cache.Discard()
			_ = d.chargeMinimalFee(store, payer)
		}
	}()

	if err := d.chargeFee(cache, payer, fee); err != nil {
		return nil, errors.Wrap(err, "cannot charge fee")
	}
	cres, err = next.Check(ctx, cache, tx)
	if err != nil {
		return nil, err
	}
	// if we have success, ensure that we paid at least the RequiredFee (IsGTE enforces the same token)
	if !cres.RequiredFee.IsZero() && !fee.IsGTE(cres.RequiredFee) {
		return nil, errors.Wrapf(errors.ErrAmount, "fee less than required fee of %#v", cres.RequiredFee)
	}
	return cres, nil
}

// Deliver verifies and deducts fees before calling down the stack
func (d DynamicFeeDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (dres *weave.DeliverResult, derr error) {
	fee, payer, cache, err := d.prepare(ctx, store, tx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot prepare")
	}

	defer func() {
		if derr == nil {
			// If we cannot write the cache, then we return error
			// here means that nothing got committed. This means
			// that no change is persisted and the whole delivery
			// failed.
			if err := cache.Write(); err != nil {
				cache.Discard()
				dres = nil
				derr = err
			}
		} else {
			cache.Discard()
			_ = d.chargeMinimalFee(store, payer)
		}
	}()

	if err := d.chargeFee(cache, payer, fee); err != nil {
		return nil, errors.Wrap(err, "cannot charge fee")
	}
	res, err := next.Deliver(ctx, cache, tx)
	if err != nil {
		return res, err
	}
	// if we have success, ensure that we paid at least the RequiredFee (IsGTE enforces the same token)
	if !res.RequiredFee.IsZero() && !fee.IsGTE(res.RequiredFee) {
		return nil, errors.Wrapf(errors.ErrAmount, "Fee less than required fee of %#v", res.RequiredFee)
	}
	return res, nil
}

func (d DynamicFeeDecorator) chargeFee(store weave.KVStore, src weave.Address, amount coin.Coin) error {
	if amount.IsZero() {
		return nil
	}
	dest := mustLoadConf(store).CollectorAddress
	return d.ctrl.MoveCoins(store, src, dest, amount)
}

// chargeMinimalFee deduct an anty span fee from a given account.
func (d DynamicFeeDecorator) chargeMinimalFee(store weave.KVStore, src weave.Address) error {
	fee := mustLoadConf(store).MinimalFee
	if fee.IsZero() {
		return nil
	}
	if fee.Ticker == "" {
		return errors.Wrap(errors.ErrHuman, "minimal fee without a ticker")
	}
	return d.chargeFee(store, src, fee)
}

// prepare is all shared setup between Check and Deliver. It computes the fee
// for the transaction, ensures that the payer is authenticated and prepares
// the database transaction.
func (d DynamicFeeDecorator) prepare(ctx weave.Context, store weave.KVStore, tx weave.Tx) (fee coin.Coin, payer weave.Address, cache weave.KVCacheWrap, err error) {
	finfo, err := d.extractFee(ctx, tx, store)
	if err != nil {
		return fee, payer, cache, errors.Wrap(err, "cannot extract fee")
	}
	// Dererefence the fees (handling nil).
	if pfee := finfo.GetFees(); pfee != nil {
		fee = *pfee
	}
	payer = finfo.GetPayer()

	// Verify we have access to the money.
	if !d.auth.HasAddress(ctx, payer) {
		err := errors.Wrap(errors.ErrUnauthorized, "fee payer signature missing")
		return fee, payer, cache, err
	}

	// Ensure we can execute subtransactions (see check on utils.Savepoint).
	cstore, ok := store.(weave.CacheableKVStore)
	if !ok {
		err = errors.Wrap(errors.ErrHuman, "need cachable kvstore")
		return fee, payer, cache, err
	}
	cache = cstore.CacheWrap()
	return fee, payer, cache, nil
}

// this returns the fee info to deduct and the error if incorrectly set
func (d DynamicFeeDecorator) extractFee(ctx weave.Context, tx weave.Tx, store weave.KVStore) (*FeeInfo, error) {
	var finfo *FeeInfo
	ftx, ok := tx.(FeeTx)
	if ok {
		payer := x.MainSigner(ctx, d.auth).Address()
		finfo = ftx.GetFees().DefaultPayer(payer)
	}

	txFee := finfo.GetFees()
	if coin.IsEmpty(txFee) {
		minFee := mustLoadConf(store).MinimalFee
		if minFee.IsZero() {
			return finfo, nil
		}
		return nil, errors.Wrap(errors.ErrAmount, "zero transaction fee is not allowed")
	}

	if err := finfo.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid fee")
	}

	minFee := mustLoadConf(store).MinimalFee
	if minFee.IsZero() {
		return finfo, nil
	}
	if minFee.Ticker == "" {
		return nil, errors.Wrap(errors.ErrHuman, "minumal fee curency not set")
	}
	if !txFee.SameType(minFee) {
		err := errors.Wrapf(errors.ErrCurrency,
			"min fee is %s and tx fee is %s", minFee.Ticker, txFee.Ticker)
		return nil, err

	}
	if !txFee.IsGTE(minFee) {
		return nil, errors.Wrapf(errors.ErrAmount, "transaction fee less than minimum: %v", txFee)
	}
	return finfo, nil
}
