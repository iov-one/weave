package weave

import (
	"crypto/sha256"
	// "golang.org/x/crypto/blake2b"

	crypto "github.com/tendermint/go-crypto"
)

var (
	// AddressLength is the length of all addresses
	// You can modify it in init() before any addresses are calculated,
	// but it must not change during the lifetime of the kvstore
	AddressLength = 20
)

// Address represents a collision-free, one-way digest
// of data (usually a public key) that can be used to identify a signer
//
// It will be of size AddressLength
type Address []byte

// NewAddress hashes and truncates into the proper size
func NewAddress(data []byte) Address {
	// h := blake2b.Sum256(data)
	h := sha256.Sum256(data)
	return h[:AddressLength]
}

// Msg is message for the blockchain to take an action
// (Make a state transition). It is just the request, and
// must be validated by the Handlers. All authentication
// information is in the wrapping Tx.
type Msg interface {
	// Return the message path.
	// This is used by the Router to locate the proper Handler.
	// Msg should be created alongside the Handler that corresponds to them.
	//
	// Multiple types may have the same value, and will end up at the
	// same Handler.
	//
	// Must be alphanumeric [0-9A-Za-z_\-]+
	Path() string

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
// (either the PubKey or the Address), and a sequence number to
// prevent replay attacks.
// A given signer must submit transactions with the sequence number
// increasing by 1 each time (starting at 0)
type StdSignature struct {
	PubKey    crypto.PubKey // required iff Sequence == 0
	Address   Address       // required iff PubKey is not present
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
