package auth

import (
	"encoding/binary"

	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	crypto "github.com/tendermint/go-crypto"
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

	bz := tx.GetSignBytes()
	chainID := weave.GetChainID(ctx)

	sigs := tx.GetSignatures()
	if len(sigs) == 0 {
		return nil, errors.ErrMissingSignature()
	}

	signers := make([]weave.Address, 0, len(sigs))
	for _, sig := range sigs {
		// TODO: separate into own function (verify one sig)
		signer, err := VerifySignature(store, sig, bz, chainID)
		if err != nil {
			return ctx, err
		}
		signers = append(signers, signer)
	}

	return withSigners(ctx, signers), nil
}

func VerifySignature(store weave.KVStore, sig StdSignature,
	signBytes []byte, chainID string) (weave.Address, error) {

	// load account
	key := sig.Address
	if key == nil {
		key = sig.PubKey.Address()
	}
	user := GetOrCreateUser(store, NewUserKey(key))

	if user.HasPubKey() {
		if sig.PubKey.Empty() {
			// TODO: better error
			return nil, errors.ErrInternal("Must set pubkey on first sign")
		}
		user.SetPubKey(sig.PubKey)
	}

	toSign := BuildSignBytes(signBytes, chainID, sig.Sequence)
	if !user.PubKey().VerifyBytes(toSign, sig.Signature) {
		return nil, errors.ErrInvalidSignature()
	}

	err := user.CheckAndIncrementSequence(sig.Sequence)
	if err != nil {
		return nil, err
	}

	user.Save()
	return key, nil
}

// BuildSignBytes combines all info on the actual tx before signing
func BuildSignBytes(signBytes []byte, chainID string, seq int64) []byte {
	// encode nonce as 8 byte, big-endian
	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, uint64(seq))

	// concatentate everything
	output := make([]byte, 0, len(signBytes)+len(chainID)+8)
	output = append(output, signBytes...)
	output = append(output, []byte(chainID)...)
	output = append(output, nonce...)
	return output
}

// BuildSignBytesTx calculates the sign bytes given a tx
func BuildSignBytesTx(tx SignedTx, chainID string, seq int64) []byte {
	signBytes := tx.GetSignBytes()
	return BuildSignBytes(signBytes, chainID, seq)
}

// SignTx creates a signature for the given tx
func SignTx(key crypto.PrivKey, tx SignedTx, chainID string,
	seq int64) StdSignature {

	signBytes := BuildSignBytesTx(tx, chainID, seq)
	sig := key.Sign(signBytes)
	pub := key.PubKey()

	res := StdSignature{
		Signature: sig,
		Sequence:  seq,
	}

	if seq == 0 {
		res.PubKey = pub
	} else {
		res.Address = pub.Address()
	}

	return res
}
