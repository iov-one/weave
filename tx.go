package weave

import crypto "github.com/tendermint/go-crypto"

// KeyHash represents a collision-free, one-way digest
// of a public key that can be used to identify a signer
type KeyHash []byte

// Msg is message for the blockchain to take an action
// (Make a state transition). It is just the request, and
// must be validated by the Handlers. All authentication
// information is in the wrapping Tx.
type Msg interface {
	// Return the message type. This is used to locate the proper handler.
	// Must be alphanumeric [0-9A-Za-z_\-]+
	Type() string

	// ValidateBasic does a simple validation check that
	// doesn't require access to any other information.
	ValidateBasic() error
}

// Tx represent the data sent from the user to the chain.
// It includes the actual message, along with information needed
// to authenticate the sender (cryptographic signatures),
// and anything else needed to pass through middleware.
//
// A Tx implementation *may* need to support other methods,
// like GetFee, to satisfy a fee-checker.
type Tx interface {
	// GetMsg returns the action we wish to communicate
	GetMsg() Msg

	// GetSignBytes returns the canonical byte representation of the Msg.
	// Helpful to store original, unparsed bytes here
	GetSignBytes() []byte

	// Signatures returns the signature of signers who signed the Msg.
	GetSignatures() []StdSignature
}

// TxDecoder can parse bytes into a Tx
type TxDecoder func(txBytes []byte) (Tx, error)

// StdSignature represents the signature, the identity of the signer
// (either the PubKey or the KeyHash), and a sequence number to
// prevent replay attacks.
// A given signer must submit transactions with the sequence number
// increasing by 1 each time (starting at 0)
type StdSignature struct {
	PubKey    crypto.PubKey // required iff Sequence == 0
	KeyHash   KeyHash       // required iff PubKey is not present
	Signature crypto.Signature
	Sequence  int64
}

// var _ Tx = (*StdTx)(nil)

// type StdTx struct {
// 	Msg
// 	Signatures []StdSignature
// }

// func (tx StdTx) GetFeePayer() crypto.Address   { return tx.Signatures[0].PubKey.Address() }
// func (tx StdTx) GetSignatures() []StdSignature { return tx.Signatures }
