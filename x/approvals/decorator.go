package approvals

import (
	"github.com/confio/weave/errors"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

// Decorator checks multisig contract if available
type TimeoutDecorator struct {
	auth   x.Authenticator
	bucket ApprovalBucket
}

var _ weave.Decorator = TimeoutDecorator{}

// NewDecorator returns a default multisig decorator
func NewDecorator(auth x.Authenticator) TimeoutDecorator {
	return TimeoutDecorator{auth, NewApprovalBucket()}
}

// Check enforce multisig contract before calling down the stack
func (d TimeoutDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (weave.CheckResult, error) {
	var res weave.CheckResult
	newCtx, err := d.withApproval(ctx, store, tx)
	if err != nil {
		return res, err
	}

	return next.Check(newCtx, store, tx)
}

// Deliver enforces multisig contract before calling down the stack
func (d TimeoutDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	newCtx, err := d.withApproval(ctx, store, tx)
	if err != nil {
		return res, err
	}

	return next.Deliver(newCtx, store, tx)
}

func (d TimeoutDecorator) withApproval(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.Context, error) {
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

			ctx = withApproval(ctx, approvalID)
		}
	}

	return ctx, nil
}

func (d TimeoutDecorator) getApproval(store weave.KVStore, id []byte) (*Approval, error) {
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

// Decorator checks multisig contract if available
type SigDecorator struct {
	auth x.Authenticator
}

var _ weave.Decorator = SigDecorator{}

// NewDecorator returns a default multisig decorator
func NewSigDecorator(auth x.Authenticator) SigDecorator {
	return SigDecorator{auth}
}

// Check enforce multisig contract before calling down the stack
func (d SigDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (weave.CheckResult, error) {
	var res weave.CheckResult
	newCtx, err := d.withApproval(ctx, store, tx)
	if err != nil {
		return res, err
	}

	return next.Check(newCtx, store, tx)
}

// Deliver enforces multisig contract before calling down the stack
func (d SigDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (weave.DeliverResult, error) {
	var res weave.DeliverResult
	newCtx, err := d.withApproval(ctx, store, tx)
	if err != nil {
		return res, err
	}

	return next.Deliver(newCtx, store, tx)
}

func (d SigDecorator) withApproval(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.Context, error) {
	addresses := x.GetAddresses(ctx, d.auth)
	for _, addr := range addresses {
		ctx = withApproval(ctx, addr)
	}
	return ctx, nil
}
