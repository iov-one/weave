package crypto

import (
	"errors"

	"github.com/iov-one/weave"
)

// ExtensionName is used for the Conditions we get from signatures
const ExtensionName = "sigs"

// Pubkey represents a crypto public key we use
type PubKey interface {
	Verify(message []byte, sig *Signature) bool
	Condition() weave.Condition
}

// Signer is the functionality we use from a private key
// No serializing to support hardware devices as well.
type Signer interface {
	Sign(message []byte) (*Signature, error)
	PublicKey() *PublicKey
}

//-------- unwrappers --------
// enforce that all of the one-ofs implement some interfaces

// unwrap a PublicKey struct into a Pubkey interface
func (p PublicKey) unwrap() PubKey {
	pub := p.GetPub()
	if pub == nil {
		return nil
	}
	return pub.(PubKey)
}

// unwrap a PrivateKey struct into a Signer interface
func (p PrivateKey) unwrap() Signer {
	priv := p.GetPriv()
	if priv == nil {
		return nil
	}
	return priv.(Signer)
}

//-------- implement interfaces in protobuf --------------

var _ PubKey = (*PublicKey)(nil)

// Verify verifies the signature was created with this message and public key
func (p *PublicKey) Verify(message []byte, sig *Signature) bool {
	// Absence of a public key is always failing the signature test.
	if p.unwrap() == nil {
		return false
	}
	return p.unwrap().Verify(message, sig)
}

// Condition generates a Condition object to represent a valid
// signature.
func (p *PublicKey) Condition() weave.Condition {
	in := p.unwrap()
	if in == nil {
		return nil
	}
	return in.Condition()
}

// Address is a convenience method to get the Condition then take Address
func (p *PublicKey) Address() weave.Address {
	return p.Condition().Address()
}

var _ Signer = (*PrivateKey)(nil)

// Sign returns a matching signature for this private key
func (p *PrivateKey) Sign(message []byte) (*Signature, error) {
	if p.unwrap() == nil {
		return nil, errors.New("private key missing")
	}
	return p.unwrap().Sign(message)
}

// PublicKey returns the corresponding PublicKey
func (p *PrivateKey) PublicKey() *PublicKey {
	return p.unwrap().PublicKey()
}
