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

// VerifyTxSignatures checks all the signatures on the tx,
// which must have at least one.
//
// returns list of signer addresses (possibly empty),
// or error if any signature is invalid
func VerifyTxSignatures(store weave.KVStore, tx SignedTx,
	chainID string) ([]weave.Address, error) {

	bz := tx.GetSignBytes()
	sigs := tx.GetSignatures()

	signers := make([]weave.Address, 0, len(sigs))
	for _, sig := range sigs {
		// TODO: separate into own function (verify one sig)
		signer, err := VerifySignature(store, sig, bz, chainID)
		if err != nil {
			return nil, err
		}
		signers = append(signers, signer)
	}

	return signers, nil
	// return withSigners(ctx, signers), nil
}

// VerifySignature checks one signature against signbytes,
// check chain and updates state in the store
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
func SignTx(signer Signer, tx SignedTx, chainID string,
	seq int64) StdSignature {

	signBytes := BuildSignBytesTx(tx, chainID, seq)
	sig := signer.Sign(signBytes)
	pub := signer.PubKey()

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

// Signer is a generalization of a privkey
type Signer interface {
	Sign([]byte) crypto.Signature
	PubKey() crypto.PubKey
}
