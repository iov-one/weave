package auth

import (
	"github.com/confio/weave"
	"github.com/confio/weave/errors"
)

//----------------- Controller ------------------
//
// Place actual business logic here.
// Anything that may be called from another extension can be public
// to encourage composition. Anything unsafe to be called from
// arbitrary extensions should be private.
// This is the main entry point to a package.
//
// Controller should contain package-level functions, not
// objects with state, to make it easy to call from other extensions.

// VerifySignatures checks all the signatures on the tx, which must have
// at least one.
//
// returns error on bad signature and
// returns a modified context with auth info on success
func VerifySignatures(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (weave.Context, error) {

	sigs := tx.GetSignatures()
	if len(sigs) == 0 {
		return nil, errors.ErrMissingSignature()
	}

	bz := tx.GetSignBytes()
	for _, sig := range sigs {
		// load account to get pubkey and nonce

		// verify signature matches (and set pubkey if needed)
		if !sig.PubKey.VerifyBytes(bz, sig.Signature) {
			return ctx, errors.ErrInvalidSignature()
		}

		// verify nonce is proper (and increment)

		// save account changes

		// add to the context we will return
	}

	return ctx, nil

}
