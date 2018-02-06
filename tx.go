package weave

import (
	"crypto/sha256"
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
	// GetMsg returns the action we wish to communicate
	GetMsg() Msg
}

// TxDecoder can parse bytes into a Tx
type TxDecoder func(txBytes []byte) (Tx, error)

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

// AuthFunc is a function we can use to extract authentication info
// from the context. This should be passed into the constructor of
// handlers, so we can plug in another authentication system,
// rather than hardcoding x/auth for all extensions.
type AuthFunc func(Context) []Address

func MultiAuth(fns ...AuthFunc) AuthFunc {
	return func(ctx Context) (res []Address) {
		for _, fn := range fns {
			add := fn(ctx)
			if len(add) > 0 {
				res = append(res, add...)
			}
		}
		return res
	}
}

// MainSigner returns the first signed if any, otherwise nil
func MainSigner(ctx Context, fn AuthFunc) Address {
	auth := fn(ctx)
	if len(auth) == 0 {
		return nil
	}
	return auth[0]
}

// HasAllSigners returns true if all elements in required are
// also in signed.
func HasAllSigners(required []Address, signed []Address) bool {
	return HasNSigners(len(required), required, signed)
}

// HasSigner returns true if this address has signed
func HasSigner(required Address, signed []Address) bool {
	return HasNSigners(1, []Address{required}, signed)
}

// HasNSigners returns true if at least n elements in requested are
// also in signed.
// Useful for threshold conditions (1 of 3, 3 of 5, etc...)
func HasNSigners(n int, requested []Address, signed []Address) bool {
	// TODO: Implement when needed
	return false
}

//--------------- serialization stuff ---------------------

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

// MustMarshal will succeed or panic
func MustMarshal(obj Marshaller) []byte {
	bz, err := obj.Marshal()
	if err != nil {
		panic(err)
	}
	return bz
}

// MustUnmarshal will succeed or panic
func MustUnmarshal(obj Persistent, bz []byte) {
	err := obj.Unmarshal(bz)
	if err != nil {
		panic(err)
	}
}

//-------------------- Validation ---------

// Validater is any struct that can be validated.
// Not the same as a Validator, which votes on the blocks.
type Validater interface {
	Validate() error
}

// MustValidate panics if the object is not valid
func MustValidate(obj Validater) {
	err := obj.Validate()
	if err != nil {
		panic(err)
	}
}

type MarshalValidater interface {
	Marshaller
	Validater
}

// MustMarshalValid marshals the object, but panics
// if the object is not valid or has trouble marshalling
func MustMarshalValid(obj MarshalValidater) []byte {
	MustValidate(obj)
	return MustMarshal(obj)
}
