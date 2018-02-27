/*
Package auth provides basic authentication
middleware to verify the signatures on the transaction,
and maintain nonces for replay protection.
*/
package sigs

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

// AllowMissingSigs allows us to pass along items with no signatures
func (d Decorator) AllowMissingSigs() Decorator {
	d.allowMissingSigs = true
	return d
}

// Check verifies signatures before calling down the stack
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Checker) (weave.CheckResult, error) {

	var res weave.CheckResult
	var err error
	var signers []weave.Address

	if stx, ok := tx.(SignedTx); ok {
		chainID := weave.GetChainID(ctx)
		signers, err = VerifyTxSignatures(store, stx, chainID)
		if err != nil {
			return res, err
		}
	}
	if len(signers) == 0 && !d.allowMissingSigs {
		return res, errors.ErrMissingSignature()
	}

	ctx = withSigners(ctx, signers)
	return next.Check(ctx, store, tx)
}

// Deliver verifies signatures before calling down the stack
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (weave.DeliverResult, error) {

	var res weave.DeliverResult
	var err error
	var signers []weave.Address
	if stx, ok := tx.(SignedTx); ok {
		chainID := weave.GetChainID(ctx)
		signers, err = VerifyTxSignatures(store, stx, chainID)
		if err != nil {
			return res, err
		}
	}
	if len(signers) == 0 && !d.allowMissingSigs {
		return res, errors.ErrMissingSignature()
	}

	ctx = withSigners(ctx, signers)
	return next.Deliver(ctx, store, tx)
}
