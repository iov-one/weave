package crypto

import (
	"github.com/iov-one/weave"
	"golang.org/x/crypto/ed25519"
)

var _ PubKey = (*PublicKey_Ed25519)(nil)

// Verify verifies the signature was created with this message and public key
func (p *PublicKey_Ed25519) Verify(message []byte, sig *Signature) bool {
	edsig, ok := sig.GetSig().(*Signature_Ed25519)
	if !ok {
		return false
	}

	publicKey := ed25519.PublicKey(p.Ed25519)
	return ed25519.Verify(publicKey, message, edsig.Ed25519)
}

// Condition encodes the public key into a weave permission
func (p *PublicKey_Ed25519) Condition() weave.Condition {
	return weave.NewCondition(ExtensionName, "ed25519", p.Ed25519)
}

var _ Signer = (*PrivateKey_Ed25519)(nil)

// Sign returns a matching signature for this private key
func (p *PrivateKey_Ed25519) Sign(message []byte) (*Signature, error) {
	privateKey := ed25519.PrivateKey(p.Ed25519)
	bz := ed25519.Sign(privateKey, message)
	sig := &Signature{
		Sig: &Signature_Ed25519{
			Ed25519: bz,
		},
	}
	return sig, nil
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
func GenPrivKeyEd25519() *PrivateKey {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	return &PrivateKey{
		Priv: &PrivateKey_Ed25519{
			Ed25519: priv,
		},
	}
}

// PrivKeyEd25519FromSeed will deterministically generate a private key from
// a given seed. Use if you have a strong source of external randomness,
// or for deterministic keys in test cases.
func PrivKeyEd25519FromSeed(seed []byte) *PrivateKey {
	priv := ed25519.NewKeyFromSeed(seed)
	return &PrivateKey{
		Priv: &PrivateKey_Ed25519{
			Ed25519: priv,
		},
	}
}
