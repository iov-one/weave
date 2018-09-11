package multisig

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

// Decorator checks multisig contract if available
type Decorator struct {
	auth   x.Authenticator
	bucket ContractBucket
}

var _ weave.Decorator = Decorator{}

// NewDecorator returns a default multisig decorator
func NewDecorator() Decorator {
	return Decorator{}
}

// Check enforce multisig contract before calling down the stack
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Checker) (weave.CheckResult, error) {
	var res weave.CheckResult
	err := d.withMultisig(ctx, store, tx)
	if err != nil {
		return res, err
	}

	return next.Check(ctx, store, tx)
}

// Deliver enforces multisig contract before calling down the stack
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	err := d.withMultisig(ctx, store, tx)
	if err != nil {
		return res, err
	}

	return next.Deliver(ctx, store, tx)
}

func (d Decorator) withMultisig(ctx weave.Context, store weave.KVStore, tx weave.Tx) error {
	if multisigContract, ok := tx.(MultiSigTx); ok {
		// does tx have multisig ?
		addr := multisigContract.GetMultiSig()
		if addr == nil {
			return nil
		}

		// load contract
		obj, err := d.bucket.Get(store, multisigContract.GetMultiSig())
		if err != nil {
			return err
		}

		// check sigs
		contract := obj.Value().(*Contract)
		conditions := MultiSigConditions(*contract)
		authenticated := x.HasNConditions(ctx, d.auth, conditions, int(contract.ActivationThreshold))
		if !authenticated {
			return ErrUnauthorizedMultiSig(addr)
		}
	}
	return nil
}
