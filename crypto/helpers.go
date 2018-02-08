package crypto

import "github.com/confio/weave"

// PubKey represents a crypto public key we use
type PubKey interface {
	Verify(message []byte, sig *Signature) bool
	Address() weave.Address
}

// Signer is the functionality we use from a private key
// No serializing to support hardware devices as well.
type Signer interface {
	Sign(message []byte) *Signature
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

// Address hashes the public key into a weave address
func (p *PublicKey) Address() weave.Address {
	in := p.unwrap()
	if in == nil {
		return nil
	}
	return in.Address()
}

var _ Signer = (*PrivateKey)(nil)

// Sign returns a matching signature for this private key
func (p *PrivateKey) Sign(message []byte) *Signature {
	return p.unwrap().Sign(message)
}

// PublicKey returns the corresponding PublicKey
func (p *PrivateKey) PublicKey() *PublicKey {
	return p.unwrap().PublicKey()
}
