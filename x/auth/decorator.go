/*
Package auth provides basic authentication
middleware to verify the signatures on the transaction,
and maintain nonces for replay protection.
*/
package auth

import (
	"github.com/confio/weave"
	"github.com/confio/weave/errors"
)

//----------------- Decorator ----------------
//
// This is just a binding from the functionality into the
// Application stack, not much business logic here.

// Decorator verifies the signatures and adds them to the context
type Decorator struct {
	allowMissingSigs bool
}

var _ weave.Decorator = Decorator{}

// NewDecorator returns a default authentication decorator,
// which appends the chainID before checking the signature,
// and requires at least one signature to be present
func NewDecorator() Decorator {
	return Decorator{
		allowMissingSigs: false,
	}
}

// SigsNotRequired allows us to pass along items with no signatures
func (d Decorator) SigsNotRequired() Decorator {
	d.allowMissingSigs = true
	return d
}

// Check verifies signatures before calling down the stack
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Checker) (res weave.CheckResult, err error) {

	stx, ok := tx.(SignedTx)
	if !ok {
		if d.allowMissingSigs {
			return next.Check(ctx, store, tx)
		}
		return res, errors.ErrMissingSignature()
	}

	ctx, err = VerifySignatures(ctx, store, stx)
	if err != nil && !(d.allowMissingSigs && errors.IsMissingSignatureErr(err)) {
		return
	}
	return next.Check(ctx, store, tx)
}

// Deliver verifies signatures before calling down the stack
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (res weave.DeliverResult, err error) {

	stx, ok := tx.(SignedTx)
	if !ok {
		if d.allowMissingSigs {
			return next.Deliver(ctx, store, tx)
		}
		return res, errors.ErrMissingSignature()
	}

	ctx, err = VerifySignatures(ctx, store, stx)
	if err != nil && !(d.allowMissingSigs && errors.IsMissingSignatureErr(err)) {
		return
	}
	return next.Deliver(ctx, store, tx)
}
