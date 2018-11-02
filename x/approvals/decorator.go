package approvals

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

// Decorator checks multisig contract if available
type Decorator struct {
	auth x.Authenticator
}

var _ weave.Decorator = Decorator{}

// NewDecorator returns a default multisig decorator
func NewDecorator(auth x.Authenticator) Decorator {
	return Decorator{auth}
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
	addresses := x.GetAddresses(ctx, d.auth)
	for _, addr := range addresses {
		ctx = withApproval(ctx, addr)
	}
	return ctx, nil
}
