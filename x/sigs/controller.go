package sigs

import (
	"crypto/sha512"
	"encoding/binary"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/errors"
)

// SignCodeV1 is the current way to prefix the bytes we use to build
// a signature
var SignCodeV1 = []byte{0, 0xCA, 0xFE, 0}

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
	chainID string) ([]weave.Condition, error) {

	bz, err := tx.GetSignBytes()
	if err != nil {
		return nil, err
	}
	sigs := tx.GetSignatures()

	signers := make([]weave.Condition, 0, len(sigs))
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
func VerifySignature(db weave.KVStore, sig *StdSignature,
	signBytes []byte, chainID string) (weave.Condition, error) {

	// we guarantee sequence makes sense and pubkey or address is there
	err := sig.Validate()
	if err != nil {
		return nil, err
	}

	bucket := NewBucket()

	// load account
	obj, err := bucket.GetOrCreate(db, sig.Pubkey)
	if err != nil {
		return nil, err
	}

	toSign, err := BuildSignBytes(signBytes, chainID, sig.Sequence)
	if err != nil {
		return nil, err
	}

	user := AsUser(obj)
	if !user.Pubkey.Verify(toSign, sig.Signature) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "invalid signature")
	}

	err = user.CheckAndIncrementSequence(sig.Sequence)
	if err != nil {
		return nil, err
	}
	err = bucket.Save(db, obj)
	if err != nil {
		return nil, err
	}
	return user.Pubkey.Condition(), nil
}

/*
BuildSignBytes combines all info on the actual tx before signing

As specified in https://github.com/iov-one/weave/issues/70,
we use the following format:

version | len(chainID) | chainID      | nonce             | signBytes
4bytes  | uint8        | ascii string | int64 (bigendian) | serialized transaction

This is then prehashed with sha512 before fed into
the public key signing/verification step
*/
func BuildSignBytes(signBytes []byte, chainID string, seq int64) ([]byte, error) {
	if seq < 0 {
		return nil, errors.Wrap(ErrInvalidSequence, "negative")
	}
	if !weave.IsValidChainID(chainID) {
		return nil, errors.Wrapf(errors.ErrInput, "chain id: %v", chainID)
	}

	// encode nonce as 8 byte, big-endian
	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, uint64(seq))

	// concatentate everything
	output := make([]byte, 0, 4+1+len(chainID)+8+len(signBytes))
	output = append(output, []byte(SignCodeV1)...)
	output = append(output, uint8(len(chainID)))
	output = append(output, []byte(chainID)...)
	output = append(output, nonce...)
	output = append(output, signBytes...)

	// now, we take the sha512 hash of the result,
	// so we have a constant length output to feed into eddsa
	// which we need so ledger can support this as well
	hashed := sha512.Sum512(output)
	return hashed[:], nil
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
		Pubkey:    pub,
		Signature: sig,
		Sequence:  seq,
	}

	return res, nil
}
