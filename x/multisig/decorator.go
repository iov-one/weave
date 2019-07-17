package multisig

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	multisigParticipantGasCost = 10
)

// Decorator checks multisig contract if available
type Decorator struct {
	auth   x.Authenticator
	bucket orm.ModelBucket
}

var _ weave.Decorator = Decorator{}

// NewDecorator returns a default multisig decorator
func NewDecorator(auth x.Authenticator) Decorator {
	return Decorator{auth, NewContractBucket()}
}

// Check enforce multisig contract before calling down the stack
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	newCtx, cost, err := d.authMultisig(ctx, store, tx)
	if err != nil {
		return nil, err
	}

	res, err := next.Check(newCtx, store, tx)
	if err != nil {
		return nil, err
	}
	res.GasPayment += cost
	return res, nil
}

// Deliver enforces multisig contract before calling down the stack
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	newCtx, _, err := d.authMultisig(ctx, store, tx)
	if err != nil {
		return nil, err
	}

	return next.Deliver(newCtx, store, tx)
}

func (d Decorator) authMultisig(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.Context, int64, error) {
	multisigContract, ok := tx.(MultiSigTx)
	if !ok {
		return ctx, 0, nil
	}

	var gasCost int64
	ids := multisigContract.GetMultisig()
	for _, contractID := range ids {
		if contractID == nil {
			continue
		}

		// A contract can be activated by another contract being fulfilled.
		if d.auth.HasAddress(ctx, MultiSigCondition(contractID).Address()) {
			continue
		}

		var contract Contract
		if err := d.bucket.One(store, contractID, &contract); err != nil {
			return ctx, 0, errors.Wrap(err, "cannot load contract from the store")
		}

		var weight Weight
		for _, p := range contract.Participants {
			if d.auth.HasAddress(ctx, p.Signature) {
				weight += p.Weight
				gasCost += multisigParticipantGasCost
			}
		}
		if weight < contract.ActivationThreshold {
			err := errors.Wrapf(errors.ErrUnauthorized,
				"%d weight is not enough to activate %q", weight, contractID)
			return ctx, 0, err
		}

		ctx = withMultisig(ctx, contractID)
	}

	return ctx, gasCost, nil
}
