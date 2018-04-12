package crypto

import (
	"github.com/confio/weave"
)

// ExtensionName is used for the Permissions we get from signatures
const ExtensionName = "sigs"

// PubKey represents a crypto public key we use
type PubKey interface {
	Verify(message []byte, sig *Signature) bool
	Permission() weave.Permission
}

// Signer is the functionality we use from a private key
// No serializing to support hardware devices as well.
type Signer interface {
	Sign(message []byte) (*Signature, error)
	PublicKey() *PublicKey
}

//-------- unwrappers --------
// enforce that all of the one-ofs implement some interfaces

// unwrap a PublicKey struct into a PubKey interface
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
	return p.unwrap().Verify(message, sig)
}

// Permission generates a Permission object to represent a valid
// signature.
//    p.Permission().Address()
// will return an Address if needed.
func (p *PublicKey) Permission() weave.Permission {
	in := p.unwrap()
	if in == nil {
		return nil
	}
	return in.Permission()
}

var _ Signer = (*PrivateKey)(nil)

// Sign returns a matching signature for this private key
func (p *PrivateKey) Sign(message []byte) (*Signature, error) {
	return p.unwrap().Sign(message)
}

// PublicKey returns the corresponding PublicKey
func (p *PrivateKey) PublicKey() *PublicKey {
	return p.unwrap().PublicKey()
}
