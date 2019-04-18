/*
Package sigs provides basic authentication
middleware to verify the signatures on the transaction,
and maintain nonces for replay protection.
*/
package sigs

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	signatureVerifyCost = 500
)

// RegisterQuery will register this bucket as "/auth"
func RegisterQuery(qr weave.QueryRouter) {
	NewBucket().Register("auth", qr)
}

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

// Check verifies signatures before calling down the stack.
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	stx, ok := tx.(SignedTx)
	if !ok {
		return next.Check(ctx, store, tx)
	}

	chainID := weave.GetChainID(ctx)
	signers, err := VerifyTxSignatures(store, stx, chainID)
	if err != nil {
		return nil, errors.Wrap(err, "cannot verify signatures")
	}
	if len(signers) == 0 && !d.allowMissingSigs {
		return nil, errors.Wrap(errors.ErrUnauthorized, "missing signature")
	}

	ctx = withSigners(ctx, signers)

	res, err := next.Check(ctx, store, tx)
	if err != nil {
		return nil, err
	}
	// The most expensive operation is the signature validation. We must
	// charge gas proportionally to the effort. We only charge for the
	// valid signatures. Invalid signatures are ignored.
	res.GasPayment += int64(len(signers) * signatureVerifyCost)
	return res, nil
}

// Deliver verifies signatures before calling down the stack.
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	stx, ok := tx.(SignedTx)
	if !ok {
		return next.Deliver(ctx, store, tx)
	}

	chainID := weave.GetChainID(ctx)
	signers, err := VerifyTxSignatures(store, stx, chainID)
	if err != nil {
		return nil, errors.Wrap(err, "cannot verify signatures")
	}
	if len(signers) == 0 && !d.allowMissingSigs {
		return nil, errors.Wrap(errors.ErrUnauthorized, "missing signature")
	}

	ctx = withSigners(ctx, signers)
	return next.Deliver(ctx, store, tx)
}
