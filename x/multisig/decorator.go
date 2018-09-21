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
func NewDecorator(auth x.Authenticator) Decorator {
	return Decorator{auth, NewContractBucket()}
}

// Check enforce multisig contract before calling down the stack
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (res weave.CheckResult, err error) {
	if multisigContract, ok := tx.(MultiSigTx); ok {
		id := multisigContract.GetMultisig()
		ctx, err = d.withMultisig(ctx, store, id)
		if err != nil {
			return res, err
		}
	}

	return next.Check(ctx, store, tx)
}

// Deliver enforces multisig contract before calling down the stack
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (res weave.DeliverResult, err error) {
	if multisigContract, ok := tx.(MultiSigTx); ok {
		id := multisigContract.GetMultisig()
		ctx, err = d.withMultisig(ctx, store, id)
		if err != nil {
			return res, err
		}
	}

	return next.Deliver(ctx, store, tx)
}

func (d Decorator) withMultisig(ctx weave.Context, store weave.KVStore, id []byte) (weave.Context, error) {
	if id == nil {
		return ctx, nil
	}

	// check if we already have it
	if d.auth.HasAddress(ctx, MultiSigCondition(id).Address()) {
		return ctx, nil
	}

	// load contract
	contract, err := d.getContract(store, id)
	if err != nil {
		return ctx, err
	}

	// calls withMultisig recursively for each subcontract encountered
	sigs := make([]weave.Address, len(contract.Sigs))
	for i, sig := range contract.Sigs {
		if weave.Address(sig).Validate() == nil {
			// thats just a signture
			sigs[i] = sig
		} else {
			// that could be a multisig id
			sigs[i] = MultiSigCondition(sig).Address()
		}
	}

	// check sigs (can be sig or multisig)
	authenticated := x.HasNAddresses(ctx, d.auth, sigs, int(contract.ActivationThreshold))
	if !authenticated {
		return ctx, ErrUnauthorizedMultiSig(id)
	}

	ctx = withMultisig(ctx, id)
	return ctx, nil
}

func (d Decorator) getContract(store weave.KVStore, id []byte) (*Contract, error) {
	obj, err := d.bucket.Get(store, id)
	if err != nil {
		return nil, err
	}

	if obj == nil || (obj != nil && obj.Value() == nil) {
		return nil, ErrContractNotFound(id)
	}

	contract := obj.Value().(*Contract)
	return contract, err
}
