package auth

import (
	"encoding/binary"

	"github.com/confio/weave"
	"github.com/confio/weave/crypto"
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

// VerifyTxSignatures checks all the signatures on the tx,
// which must have at least one.
//
// returns list of signer addresses (possibly empty),
// or error if any signature is invalid
func VerifyTxSignatures(store weave.KVStore, tx SignedTx,
	chainID string) ([]weave.Address, error) {

	bz, err := tx.GetSignBytes()
	if err != nil {
		return nil, err
	}
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
}

// VerifySignature checks one signature against signbytes,
// check chain and updates state in the store
func VerifySignature(store weave.KVStore, sig *StdSignature,
	signBytes []byte, chainID string) (weave.Address, error) {

	// we guarantee sequence makes sense and pubkey or address is there
	err := sig.Validate()
	if err != nil {
		return nil, err
	}

	// load account
	pub := sig.PubKey
	key := sig.Address
	if key == nil {
		key = pub.Address()
	}
	user := GetOrCreateUser(store, NewKey(key))

	// make sure we get the key from the store if not from the sig
	if pub == nil {
		pub = user.PubKey()
		if pub == nil {
			return nil, ErrMissingPubKey()
		}
	}

	if !user.HasPubKey() {
		user.SetPubKey(pub)
	}

	toSign, err := BuildSignBytes(signBytes, chainID, sig.Sequence)
	if err != nil {
		return nil, err
	}

	if !pub.Verify(toSign, sig.Signature) {
		return nil, errors.ErrInvalidSignature()
	}

	err = user.CheckAndIncrementSequence(sig.Sequence)
	if err != nil {
		return nil, err
	}

	user.Save()
	return key, nil
}

// BuildSignBytes combines all info on the actual tx before signing
func BuildSignBytes(signBytes []byte, chainID string, seq int64) ([]byte, error) {
	if seq < 0 {
		return nil, ErrInvalidSequence("negative")
	}
	if !weave.IsValidChainID(chainID) {
		return nil, errors.ErrInvalidChainID(chainID)
	}

	// encode nonce as 8 byte, big-endian
	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, uint64(seq))

	// concatentate everything
	output := make([]byte, 0, len(signBytes)+len(chainID)+8)
	output = append(output, signBytes...)
	output = append(output, []byte(chainID)...)
	output = append(output, nonce...)
	return output, nil
}

// BuildSignBytesTx calculates the sign bytes given a tx
func BuildSignBytesTx(tx SignedTx, chainID string, seq int64) ([]byte, error) {
	signBytes, err := tx.GetSignBytes()
	if err != nil {
		return nil, err
	}
	return BuildSignBytes(signBytes, chainID, seq)
}

// SignTx creates a signature for the given tx
func SignTx(signer crypto.Signer, tx SignedTx, chainID string,
	seq int64) (*StdSignature, error) {

	signBytes, err := BuildSignBytesTx(tx, chainID, seq)
	if err != nil {
		return nil, err
	}

	sig, err := signer.Sign(signBytes)
	if err != nil {
		return nil, err
	}
	pub := signer.PublicKey()

	res := &StdSignature{
		Signature: sig,
		Sequence:  seq,
	}

	if seq == 0 {
		res.PubKey = pub
	} else {
		res.Address = pub.Address()
	}

	return res, nil
}
