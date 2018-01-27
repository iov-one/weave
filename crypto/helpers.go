package crypto

// PubKey represents a crypto public key we use
type PubKey interface {
	Verify(message []byte, sig Signature) bool
}

// Signer is the functionality we use from a private key
// No serializing to support hardware devices as well.
type Signer interface {
	Sign(message []byte) Signature
	PublicKey() PublicKey
}

//-------- unwrappers --------
// enforce that all of the one-ofs implement some interfaces

// Unwrap unwraps a PublicKey struct into a PubKey interface
func (p PublicKey) Unwrap() PubKey {
	pub := p.GetPub()
	if pub == nil {
		return nil
	}
	return pub.(PubKey)
}

// Unwrap unwraps a PrivateKey struct into a Signer interface
func (p PrivateKey) Unwrap() Signer {
	priv := p.GetPriv()
	if priv == nil {
		return nil
	}
	return priv.(Signer)
}
