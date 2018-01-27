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
	tx SignedTx) (weave.Context, error) {

	sigs := tx.GetSignatures()
	if len(sigs) == 0 {
		return nil, errors.ErrMissingSignature()
	}

	bz := tx.GetSignBytes()
	chainID := weave.GetChainID(ctx)

	signers := make([]weave.Address, 0, len(sigs))
	for _, sig := range sigs {
		// TODO: separate into own function (verify one sig)

		// load account
		key := sig.Address
		if key == nil {
			key = sig.PubKey.Address()
		}
		user := GetOrCreateUser(store, NewUserKey(key))

		// set the pubkey if not yet set
		if user.HasPubKey() {
			user.SetPubKey(sig.PubKey)
		}

		// verify signature matches (and set pubkey if needed)
		// appending the nonce for this signature
		toSign := BuildSignBytes(bz, chainID, sig.Sequence)
		if !user.PubKey().VerifyBytes(toSign, sig.Signature) {
			return ctx, errors.ErrInvalidSignature()
		}

		// verify nonce is proper (and increment)
		err := user.CheckAndIncrementSequence(sig.Sequence)
		if err != nil {
			return ctx, err
		}

		// save account changes
		user.Save()

		signers = append(signers, key)
	}

	return withSigners(ctx, signers), nil
}

// BuildSignBytes combines all info on the actual tx before signing
func BuildSignBytes(signBytes []byte, chainID string, nonce int64) []byte {
	// TODO: nonce
	return append(signBytes, []byte(chainID)...)
}

// TODO: helpers to create a signature
