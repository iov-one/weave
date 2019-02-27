/*

DynamicFeeDecorator is an enhanced version the basic FeeDecorator with better
handling of transaction errors and ability to deduct/enforce app-specific fees.

The business logic is:
* If a transaction fee < min fee, or a transaction fee cannot be paid, reject with
  an error.
* Run the transaction.
* If a transaction processing results in an error, revert all transaction
  changes and charge only the min fee.

TODO: * If a transaction succeeded, but requested a RequiredFee higher than
paid fee, revert all transaction changes and refund all but the min fee,
returning an error.

* If a transaction succeeded, and at least RequiredFee was paid, everything is
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
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/x"
)

type DynamicFeeDecorator struct {
	auth x.Authenticator
	ctrl CoinMover
	// these are cached values to not hit gconf on each read
	collector weave.Address
	minFee    *coin.Coin
}

var _ weave.Decorator = DynamicFeeDecorator{}

// NewDynamicFeeDecorator returns a DynamicFeeDecorator with the given
// minimum fee, and all collected fees going to a default address.
func NewDynamicFeeDecorator(auth x.Authenticator, ctrl CoinMover) DynamicFeeDecorator {
	return DynamicFeeDecorator{
		auth: auth,
		ctrl: ctrl,
	}
}

// Check verifies and deducts fees before calling down the stack
func (d DynamicFeeDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Checker) (weave.CheckResult, error) {

	// prepare fees for subtransaction -- shared in Check and Deliver
	var res weave.CheckResult
	fee, payer, cache, err := d.prepare(ctx, store, tx)
	if err != nil {
		return res, err
	}

	// read config
	minFee := d.getMinFee(store)

	// do subtransaction in a function for easier error handling
	res, err = func() (weave.CheckResult, error) {
		// shadow with local variables...
		err := d.maybeTakeFee(cache, payer, fee)
		if err != nil {
			return weave.CheckResult{}, err
		}
		res, err := next.Check(ctx, cache, tx)
		// TODO: check RequiredFee here and return an error if insufficient
		return res, err
	}()

	// on error, rollback, then take the minfee from the store
	if err != nil {
		cache.Discard()
		// if this fails, we aborted early above, we can just ignore return value
		// this is 2 ops, not 1, for errors, but done to optimize the success case to use 1 not 2
		d.maybeTakeFee(store, payer, minFee)
		// return error from the transaction, not the possible error from minFee deduction
		return res, err
	}

	// if success, we commit and update the importance
	cache.Write()
	res.GasPayment += toPayment(fee)
	return res, err
}

// Deliver verifies and deducts fees before calling down the stack
func (d DynamicFeeDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (weave.DeliverResult, error) {

	var res weave.DeliverResult
	fee, payer, cache, err := d.prepare(ctx, store, tx)
	if err != nil {
		return res, err
	}

	// read config
	minFee := d.getMinFee(store)

	// do subtransaction in a function for easier error handling
	res, err = func() (weave.DeliverResult, error) {
		// shadow with local variables...
		err := d.maybeTakeFee(cache, payer, fee)
		if err != nil {
			return weave.DeliverResult{}, err
		}
		res, err := next.Deliver(ctx, cache, tx)
		// TODO: check RequiredFee here and return an error if insufficient
		return res, err
	}()

	// on error, rollback, then take the minfee from the store
	if err != nil {
		cache.Discard()
		// if this fails, we aborted early above, we can just ignore return value
		// this is 2 ops, not 1, for errors, but done to optimize the success case to use 1 not 2
		d.maybeTakeFee(store, payer, minFee)
		// return error from the transaction, not the possible error from minFee deduction
		return res, err
	}

	// if success, we commit and update the importance
	cache.Write()
	return res, err
}

// maybeTakeFee will send fees only if they are positive, so we don't have to check IsZero() everywhere else
func (d DynamicFeeDecorator) maybeTakeFee(store weave.KVStore, src weave.Address, amount coin.Coin) error {
	if amount.IsZero() {
		return nil
	}
	dest := d.getCollector(store)
	return d.ctrl.MoveCoins(store, src, dest, amount)
}

// prepare is all shared setup between Check and Deliver, one more level above extractFee
func (d DynamicFeeDecorator) prepare(ctx weave.Context, store weave.KVStore, tx weave.Tx) (fee coin.Coin, payer weave.Address, cache weave.KVCacheWrap, err error) {
	var finfo *FeeInfo
	fee = coin.Coin{}

	// extract expected fee
	finfo, err = d.extractFee(ctx, tx, store)
	if err != nil {
		return
	}
	// safely dererefence the fees (handling nil)
	pfee := finfo.GetFees()
	if pfee != nil {
		fee = *pfee
	}
	payer = finfo.GetPayer()

	// verify we have access to the money
	if !d.auth.HasAddress(ctx, payer) {
		err = errors.ErrUnauthorized.New("Fee payer signature missing")
		return
	}

	// ensure we can execute subtransactions (see check on utils.Savepoint)
	cstore, ok := store.(weave.CacheableKVStore)
	if !ok {
		err = errors.ErrInternal.New("Need cachable kvstore")
		return
	}

	// prepare a cached version to work on
	cache = cstore.CacheWrap()
	return
}

// this returns the fee info to deduct and the error if incorrectly set
func (d DynamicFeeDecorator) extractFee(ctx weave.Context, tx weave.Tx, store weave.KVStore) (*FeeInfo, error) {
	var finfo *FeeInfo
	ftx, ok := tx.(FeeTx)
	if ok {
		payer := x.MainSigner(ctx, d.auth).Address()
		finfo = ftx.GetFees().DefaultPayer(payer)
	}

	fee := finfo.GetFees()
	if coin.IsEmpty(fee) {
		minFee := d.getMinFee(store)
		if minFee.IsZero() {
			return finfo, nil
		}
		return nil, errors.ErrInsufficientAmount.Newf("fees %#v", &coin.Coin{})
	}

	// make sure it is a valid fee (non-negative, going somewhere)
	err := finfo.Validate()
	if err != nil {
		return nil, err
	}

	cmp := d.getMinFee(store)
	// minimum has no currency -> accept everything
	if cmp.Ticker == "" {
		cmp.Ticker = fee.Ticker
	}
	if !fee.SameType(cmp) {
		return nil, coin.ErrInvalidCurrency.Newf("%s vs fee %s", cmp.Ticker, fee.Ticker)

	}
	if !fee.IsGTE(cmp) {
		return nil, errors.ErrInsufficientAmount.Newf("fees %#v", fee)
	}
	return finfo, nil
}

func (d DynamicFeeDecorator) getCollector(store weave.KVStore) weave.Address {
	if d.collector == nil {
		d.collector = gconf.Address(store, GconfCollectorAddress)
	}
	return d.collector
}

func (d DynamicFeeDecorator) getMinFee(store weave.KVStore) coin.Coin {
	if d.minFee == nil {
		fee := gconf.Coin(store, GconfMinimalFee)
		d.minFee = &fee
	}
	return *d.minFee
}
