package crypto

import (
	"github.com/confio/weave"
	"golang.org/x/crypto/ed25519"
)

var _ PubKey = (*PublicKey_Ed25519)(nil)

// Verify verifies the signatures
func (p *PublicKey_Ed25519) Verify(message []byte, sig *Signature) bool {
	edsig, ok := sig.GetSig().(*Signature_Ed25519)
	if !ok {
		return false
	}

	publicKey := ed25519.PublicKey(p.Ed25519)
	return ed25519.Verify(publicKey, message, edsig.Ed25519)
}

// Address hashes the public key into a weave address
func (p *PublicKey_Ed25519) Address() weave.Address {
	return weave.NewAddress(p.Ed25519)
}

var _ Signer = (*PrivateKey_Ed25519)(nil)

// Sign returns a matching signature for this private key
func (p *PrivateKey_Ed25519) Sign(message []byte) *Signature {
	privateKey := ed25519.PrivateKey(p.Ed25519)
	bz := ed25519.Sign(privateKey, message)
	return &Signature{
		Sig: &Signature_Ed25519{
			Ed25519: bz,
		},
	}
}

// PublicKey returns the corresponding PublicKey
func (p *PrivateKey_Ed25519) PublicKey() *PublicKey {
	privateKey := ed25519.PrivateKey(p.Ed25519)
	pub := privateKey.Public().(ed25519.PublicKey)
	return &PublicKey{
		Pub: &PublicKey_Ed25519{
			Ed25519: pub,
		},
	}
}

// GenPrivKeyEd25519 returns a random new private key
// (TODO: look at sources of randomness, other than default crypto/rand)
func GenPrivKeyEd25519() PrivateKey {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	return PrivateKey{
		Priv: &PrivateKey_Ed25519{
			Ed25519: priv,
		},
	}
}
