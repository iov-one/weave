package crypto

import (
	"bytes"
	"testing"
)

func TestPrivateKeySign(t *testing.T) {
	cases := map[string]struct {
		pk      *PrivateKey
		message []byte
		wantSig *Signature
		wantErr bool
	}{
		"happy path ed25519": {
			pk: &PrivateKey{
				Priv: &PrivateKey_Ed25519{
					Ed25519: make([]byte, 64),
				},
			},
			message: []byte("foo bar"),
			wantSig: &Signature{
				Sig: &Signature_Ed25519{
					Ed25519: []byte("\273\363\352\214\365\004\271\371|}\272G\316\316K\005\337Bm\340\322\007W\224-9\272\371\226\375DB\325\325\373#e\321^\030\367]\370\334\372\017\223`\036\236Ue\211\244\220\002\004\026K\227\306i\002\017"),
				},
			},
			wantErr: false,
		},
		"empty public key": {
			pk: &PrivateKey{
				// empty
			},
			message: []byte("foo bar"),
			wantSig: nil,
			wantErr: true,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			sig, err := tc.pk.Sign(tc.message)
			if tc.wantErr == (err == nil) {
				t.Fatalf("want error (%v), got %+v", tc.wantErr, err)
			}
			switch {
			case tc.wantSig == nil && sig == nil:
				// All good.
			case (tc.wantSig == nil && sig != nil) ||
				(tc.wantSig != nil && sig == nil):
				t.Logf("want %+v", tc.wantSig)
				t.Logf(" got %+v", sig)
				t.Fatal("invalid signature")
			case !bytes.Equal(tc.wantSig.GetEd25519(), sig.GetEd25519()):
				t.Logf("want %+v", tc.wantSig)
				t.Logf(" got %+v", sig)
				t.Fatal("invalid signature")
			}
		})
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
