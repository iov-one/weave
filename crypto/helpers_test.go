package crypto

import (
	"bytes"
	"testing"
)

func TestEd25519PrivateKeySign(t *testing.T) {
	pk := &PrivateKey{
		Priv: &PrivateKey_Ed25519{
			Ed25519: make([]byte, 64),
		},
	}
	sig, err := pk.Sign([]byte("foo bar"))
	if err != nil {
		t.Fatalf("cannot sing: %s", err)
	}
	wantSig := &Signature{
		Sig: &Signature_Ed25519{
			Ed25519: []byte("\273\363\352\214\365\004\271\371|}\272G\316\316K\005\337Bm\340\322\007W\224-9\272\371\226\375DB\325\325\373#e\321^\030\367]\370\334\372\017\223`\036\236Ue\211\244\220\002\004\026K\227\306i\002\017"),
		},
	}
	if !bytes.Equal(wantSig.GetEd25519(), sig.GetEd25519()) {
		t.Logf("want %+v", wantSig)
		t.Logf(" got %+v", sig)
		t.Fatal("invalid signature")
	}
}

func TestEmptyPrivateKeySign(t *testing.T) {
	emptyKey := &PrivateKey{}
	if sig, err := emptyKey.Sign([]byte("foo bar")); err == nil {
		t.Fatalf("want an error, got %q", sig)
	}
}

func TestEmptyPublicKeyVerify(t *testing.T) {
	sig := Signature{
		Sig: &Signature_Ed25519{
			Ed25519: []byte("sig 5"),
		},
	}
	var empty PublicKey
	if empty.Verify([]byte("foo"), &sig) {
		t.Fatal("empty public key must not pass verification")
	}

}
