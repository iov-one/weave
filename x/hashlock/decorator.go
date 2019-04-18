package hashlock

import (
	"github.com/iov-one/weave"
)

// Decorator adds permissions to context based on preimages
type Decorator struct{}

var _ weave.Decorator = Decorator{}

// NewDecorator returns a default hashlock decorator
func NewDecorator() Decorator {
	return Decorator{}
}

// Check verifies signatures before calling down the stack.
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	ctx = d.withPreimage(ctx, tx)
	return next.Check(ctx, store, tx)
}

// Deliver verifies signatures before calling down the stack.
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	ctx = d.withPreimage(ctx, tx)
	return next.Deliver(ctx, store, tx)
}

// withPreimage adds the hash preimage condition to the context if the Tx
// supports this functionality, and there is a preimage present.
func (d Decorator) withPreimage(ctx weave.Context, tx weave.Tx) weave.Context {
	if hk, ok := tx.(HashKeyTx); ok {
		preimage := hk.GetPreimage()
		if preimage != nil {
			ctx = withPreimage(ctx, preimage)
		}
	}
	return ctx
}
