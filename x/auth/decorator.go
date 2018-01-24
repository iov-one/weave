/*
Package auth provides basic authentication
middleware to verify the signatures on the transaction,
and maintain nonces for replay protection.
*/
package auth

import "github.com/confio/weave"

//----------------- Decorator ----------------
//
// This is just a binding from the functionality into the
// Application stack, not much business logic here.

type Decorator struct {
}

var _ weave.Decorator = Decorator{}

func NewDecorator() Decorator {
	return Decorator{}
}

// Check verifies signatures before calling down the stack
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Checker) (res weave.CheckResult, err error) {

	ctx, err = VerifySignatures(ctx, store, tx)
	if err != nil {
		return
	}
	return next.Check(ctx, store, tx)
}

// Deliver verifies signatures before calling down the stack
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (res weave.DeliverResult, err error) {

	ctx, err = VerifySignatures(ctx, store, tx)
	if err != nil {
		return
	}
	return next.Deliver(ctx, store, tx)
}
