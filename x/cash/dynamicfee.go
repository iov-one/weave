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

The business logic is:

* If tx fee < min fee, or tx fee cannot be paid, reject with error.

* Run the tx.

* If tx errors, revert all tx changes and charge only the min fee

TODO: * If tx succeeded, but requested a RequiredFee higher than paid fee,
revert all tx changes and refund all but the min fee, returning an error.

* If tx succeeded, and at least RequiredFee was paid, everything is committed
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
	// these are cached values to not hit gconf on each read
	collector weave.Address
	minFee    *x.Coin
}

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
	paid := toPayment(*fee)

	// verify we have access to the money
	if !d.auth.HasAddress(ctx, finfo.Payer) {
		return res, errors.ErrUnauthorized.New("Fee payer signature missing")
	}
	collector := d.getCollector(store)

	// ensure we can execute subtransactions (see check on utils.Savepoint)
	cstore, ok := store.(weave.CacheableKVStore)
	if !ok {
		return res, errors.ErrInternal.New("Need cachable kvstore")
	}

	//---- START: subtransaction - this can be rolled back
	cache := cstore.CacheWrap()

	// do subtransaction in a function for easier error handling
	res, err = func() (weave.CheckResult, error) {
		// shadow with local variables...
		err := d.control.MoveCoins(cache, finfo.Payer, collector, *fee)
		if err != nil {
			return weave.CheckResult{}, err
		}
		return next.Check(ctx, cache, tx)
	}()

	// on error, rollback, then take the minfee from the store
	if err != nil {
		cache.Discard()
		minFee := d.getMinFee(store)
		// if this fails, we aborted early above, we can just ignore return value
		// this is 2 ops, not 1, for errors, but done to optimize the success case to use 1 not 2
		d.control.MoveCoins(store, finfo.Payer, collector, minFee)
		// return error from the transaction, not the possible error from minFee deduction
		return res, err
	}

	// if success, we commit and update the importance
	cache.Write()
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
	collector := d.getCollector(store)
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
		minFee := d.getMinFee(store)
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

	cmp := d.getMinFee(store)
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

func (d DynamicFeeDecorator) getCollector(store weave.KVStore) weave.Address {
	if d.collector == nil {
		d.collector = gconf.Address(store, GconfCollectorAddress)
	}
	return d.collector
}

func (d DynamicFeeDecorator) getMinFee(store weave.KVStore) x.Coin {
	if d.minFee == nil {
		fee := gconf.Coin(store, GconfMinimalFee)
		d.minFee = &fee
	}
	return *d.minFee
}
