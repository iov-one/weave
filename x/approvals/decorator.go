package approvals

import (
	"github.com/confio/weave/errors"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

// Decorator checks multisig contract if available
type Decorator struct {
	auth   x.Authenticator
	bucket ApprovalBucket
}

var _ weave.Decorator = Decorator{}

// NewDecorator returns a default multisig decorator
func NewDecorator(auth x.Authenticator) Decorator {
	return Decorator{auth, NewApprovalBucket()}
}

// Check enforce multisig contract before calling down the stack
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (weave.CheckResult, error) {
	var res weave.CheckResult
	newCtx, err := d.withApproval(ctx, store, tx)
	if err != nil {
		return res, err
	}

	return next.Check(newCtx, store, tx)
}

// Deliver enforces multisig contract before calling down the stack
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	newCtx, err := d.withApproval(ctx, store, tx)
	if err != nil {
		return res, err
	}

	return next.Deliver(newCtx, store, tx)
}

func (d Decorator) withApproval(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.Context, error) {
	if approvalTx, ok := tx.(ApprovalTx); ok {
		ids := approvalTx.GetApprovals()
		for _, approvalID := range ids {
			if approvalID == nil {
				return ctx, nil
			}

			// load contract
			approval, err := d.getApproval(store, approvalID)
			if err != nil {
				return ctx, err
			}

			// check if we already have it
			if d.auth.HasAddress(ctx, ApprovalCondition(approvalID, approval.Action).Address()) {
				return ctx, nil
			}

			// check if we already have it
			if d.auth.HasAddress(ctx, approval.Address) {
				return ctx, errors.ErrUnauthorized()
			}

			ctx = withApproval(ctx, approvalID, approval.Action)
		}
	}

	return ctx, nil
}

func (d Decorator) getApproval(store weave.KVStore, id []byte) (*Approval, error) {
	obj, err := d.bucket.Get(store, id)
	if err != nil {
		return nil, err
	}

	if obj == nil || (obj != nil && obj.Value() == nil) {
		return nil, ErrContractNotFound(id)
	}

	contract := obj.Value().(*Approval)
	return contract, err
}
