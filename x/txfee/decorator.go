package txfee

import (
	"math"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
)

type Decorator struct {
}

// NewDecorator returns a transaction fee decorator instance that is adding an
// additional fee to each processed transaction, depending on that transaction
// binary size.
//
// This decorator does not directly deduct fees from transaction fee payers
// account. This decorator depends on presence of cash.DynamicFeeDecorator to
// withdraw funds equal to the final transaction fee.
func NewDecorator() *Decorator {
	return &Decorator{}
}

var _ weave.Decorator = (*Decorator)(nil)

func (d *Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	res, err := next.Check(ctx, store, tx)
	if err != nil {
		return nil, err
	}

	fee, err := d.fee(store, tx)
	if err != nil {
		if errors.ErrNotFound.Is(err) {
			// If configuration does not exist, this decorator is no-op.
			return res, nil
		}
		return nil, errors.Wrap(err, "cannot compute transaction size fee")
	}
	if !coin.IsEmpty(fee) {
		sum, err := res.RequiredFee.Add(*fee)
		if err != nil {
			return nil, errors.Wrap(err, "cannot apply transaction size fee")
		}
		res.RequiredFee = sum
	}
	return res, nil
}

func (d *Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	res, err := next.Deliver(ctx, store, tx)
	if err != nil {
		return nil, err
	}

	fee, err := d.fee(store, tx)
	if err != nil {
		if errors.ErrNotFound.Is(err) {
			// If configuration does not exist, this decorator is no-op.
			return res, nil
		}
		return nil, errors.Wrap(err, "cannot compute transaction size fee")
	}
	if !coin.IsEmpty(fee) {
		sum, err := res.RequiredFee.Add(*fee)
		if err != nil {
			return nil, errors.Wrap(err, "cannot apply transaction size fee")
		}
		res.RequiredFee = sum
	}
	return res, nil
}

// fee returns a transaction fee value, computed for given transaction and
// according to the current extension configuration.
func (Decorator) fee(db weave.ReadOnlyKVStore, tx weave.Tx) (*coin.Coin, error) {
	conf, err := loadConf(db)
	if err != nil {
		return nil, errors.Wrap(err, "load conf")
	}

	raw, err := tx.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "tx marshal")
	}
	txSize := len(raw) // Together with signatures.

	return TransactionFee(txSize, conf.BaseFee, conf.FreeBytes)
}

func TransactionFee(txSize int, baseFee coin.Coin, freeBytes int32) (*coin.Coin, error) {
	paidSize := int32(txSize) - freeBytes
	if paidSize <= 0 {
		return coin.NewCoinp(0, 0, "IOV"), nil
	}

	// ((max(0, bytes_size(tx) - free_bytes) ** 2) * base_fee
	mul := math.Pow(float64(paidSize), 2)
	if math.IsInf(mul, 1) {
		return nil, errors.ErrOverflow
	}
	fee, err := baseFee.Multiply(int64(mul))
	if err != nil {
		return nil, errors.Wrap(err, "coin multiply")
	}
	return &fee, nil
}

func loadConf(db gconf.ReadStore) (*Configuration, error) {
	var conf Configuration
	if err := gconf.Load(db, "txfee", &conf); err != nil {
		return nil, errors.Wrap(err, "gconf")
	}
	return &conf, nil
}
