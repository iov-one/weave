package auth

import (
	"github.com/confio/weave"
	crypto "github.com/tendermint/go-crypto"
)

// SignedTx represents a transaction that contains signatures,
// which can be verified by the auth.Decorator
type SignedTx interface {
	// GetSignBytes returns the canonical byte representation of the Msg.
	// Equivalent to weave.MustMarshal(tx.GetMsg()) if Msg has a deterministic
	// serialization.
	//
	// Helpful to store original, unparsed bytes here, just in case.
	GetSignBytes() []byte

	// Signatures returns the signature of signers who signed the Msg.
	GetSignatures() []StdSignature
}

// StdSignature represents the signature, the identity of the signer
// (either the PubKey or the Address), and a sequence number to
// prevent replay attacks.
//
// A given signer must submit transactions with the sequence number
// increasing by 1 each time (starting at 0)
type StdSignature struct {
	// PubKey required if Sequence == 0
	PubKey crypto.PubKey
	// Address required if PubKey is not present
	Address   weave.Address
	Signature crypto.Signature
	Sequence  int64
}
