package hashlock

import (
	"github.com/confio/weave"
)

// Decorator adds permissions to context based on preimages
type Decorator struct{}

var _ weave.Decorator = Decorator{}

// NewDecorator returns a default hashlock decorator
func NewDecorator() Decorator {
	return Decorator{}
}

// Check verifies signatures before calling down the stack
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Checker) (weave.CheckResult, error) {

	// If the Tx supports this functionality, and there is a preimage
	// present, then add this permission to the context
	if hk, ok := tx.(HashKeyTx); ok {
		preimage := hk.GetPreimage()
		if preimage != nil {
			ctx = withPreimage(ctx, preimage)
		}
	}

	return next.Check(ctx, store, tx)
}

// Deliver verifies signatures before calling down the stack
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (weave.DeliverResult, error) {

	// If the Tx supports this functionality, and there is a preimage
	// present, then add this permission to the context
	if hk, ok := tx.(HashKeyTx); ok {
		preimage := hk.GetPreimage()
		if preimage != nil {
			ctx = withPreimage(ctx, preimage)
		}
	}

	return next.Deliver(ctx, store, tx)
}
