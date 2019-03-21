package multisig

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
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
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (weave.CheckResult, error) {
	var res weave.CheckResult
	newCtx, err := d.authMultisig(ctx, store, tx)
	if err != nil {
		return res, err
	}

	return next.Check(newCtx, store, tx)
}

// Deliver enforces multisig contract before calling down the stack
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	newCtx, err := d.authMultisig(ctx, store, tx)
	if err != nil {
		return res, err
	}

	return next.Deliver(newCtx, store, tx)
}

func (d Decorator) authMultisig(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.Context, error) {
	multisigContract, ok := tx.(MultiSigTx)
	if !ok {
		return ctx, nil
	}

	ids := multisigContract.GetMultisig()
	for _, contractID := range ids {
		if contractID == nil {
			return ctx, nil
		}

		// If already authenticated it does not matter if multisig can
		// authenticate as well. Any authentication method is enough.
		if d.auth.HasAddress(ctx, MultiSigCondition(contractID).Address()) {
			return ctx, nil
		}

		contract, err := d.bucket.GetContract(store, contractID)
		if err != nil {
			return ctx, err
		}

		var power Weight
		for _, p := range contract.Participants {
			if d.auth.HasAddress(ctx, p.Signature) {
				power += p.Power
			}
		}
		if power < contract.ActivationThreshold {
			return ctx, errors.Wrapf(errors.ErrUnauthorized, "%d power is not enough", power)
		}

		ctx = withMultisig(ctx, contractID)
	}

	return ctx, nil
}
