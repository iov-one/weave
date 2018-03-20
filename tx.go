package weave

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/confio/weave/errors"
	// "golang.org/x/crypto/blake2b"
)

var (
	// AddressLength is the length of all addresses
	// You can modify it in init() before any addresses are calculated,
	// but it must not change during the lifetime of the kvstore
	AddressLength = 20
)

// Msg is message for the blockchain to take an action
// (Make a state transition). It is just the request, and
// must be validated by the Handlers. All authentication
// information is in the wrapping Tx.
type Msg interface {
	Persistent

	// Return the message path.
	// This is used by the Router to locate the proper Handler.
	// Msg should be created alongside the Handler that corresponds to them.
	//
	// Multiple types may have the same value, and will end up at the
	// same Handler.
	//
	// Must be alphanumeric [0-9A-Za-z_\-]+
	Path() string
}

// Marshaller is anything that can be represented in binary
//
// Marshall may validate the data before serializing it and
// unless you previously validated the struct,
// errors should be expected.
type Marshaller interface {
	Marshal() ([]byte, error)
}

// Persistent supports Marshal and Unmarshal
//
// This is separated from Marshal, as this almost always requires
// a pointer, and functions that only need to marshal bytes can
// use the Marshaller interface to access non-pointers.
//
// As with Marshaller, this may do internal validation on the data
// and errors should be expected.
type Persistent interface {
	Marshaller
	Unmarshal([]byte) error
}

// Tx represent the data sent from the user to the chain.
// It includes the actual message, along with information needed
// to authenticate the sender (cryptographic signatures),
// and anything else needed to pass through middleware.
//
// Each Application must define their own tx type, which
// embeds all the middlewares that we wish to use.
// auth.SignedTx and token.FeeTx are common interfaces that
// many apps will wish to support.
type Tx interface {
	Persistent

	// GetMsg returns the action we wish to communicate
	GetMsg() (Msg, error)
}

// GetPath returns the path of the message, or (missing) if no message
func GetPath(tx Tx) string {
	msg, err := tx.GetMsg()
	if err == nil && msg != nil {
		return msg.Path()
	}
	return "(missing)"
}

// TxDecoder can parse bytes into a Tx
type TxDecoder func(txBytes []byte) (Tx, error)

// Address represents a collision-free, one-way digest
// of data (usually a public key) that can be used to identify a signer
//
// It will be of size AddressLength
type Address []byte

// Equals checks if two addresses are the same
func (a Address) Equals(b Address) bool {
	return bytes.Equal(a, b)
}

// MarshalJSON provides a hex representation for JSON,
// to override the standard base64 []byte encoding
func (a Address) MarshalJSON() ([]byte, error) {
	return marshalHex(a)
}

// UnmarshalJSON parses JSON in hex representation,
// to override the standard base64 []byte encoding
func (a *Address) UnmarshalJSON(src []byte) error {
	dst := (*[]byte)(a)
	return unmarshalHex(src, dst)
}

// String returns a human readable string.
// Currently hex, may move to bech32
func (a Address) String() string {
	if len(a) == 0 {
		return "(nil)"
	}
	return strings.ToUpper(hex.EncodeToString(a))
}

// Validate returns an error if the address is not the valid size
func (a Address) Validate() error {
	if len(a) != AddressLength {
		return errors.ErrUnrecognizedAddress(a)
	}
	return nil
}

// NewAddress hashes and truncates into the proper size
func NewAddress(data []byte) Address {
	// h := blake2b.Sum256(data)
	h := sha256.Sum256(data)
	return h[:AddressLength]
}

// ObjAddress takes the address of an object
func ObjAddress(obj Marshaller) (Address, error) {
	bz, err := obj.Marshal()
	if err != nil {
		return nil, err
	}
	return NewAddress(bz), nil
}

// MustObjAddress is like ObjAddress, but panics instead of returning
// errors. Only use when you control the obj being passed in.
func MustObjAddress(obj Marshaller) Address {
	res, err := ObjAddress(obj)
	if err != nil {
		panic(err)
	}
	return res
}
