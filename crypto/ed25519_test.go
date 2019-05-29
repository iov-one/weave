package crypto

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
)

func TestEd25519Signing(t *testing.T) {
	private := GenPrivKeyEd25519()
	public := private.PublicKey()

	msg := []byte("foobar")
	msg2 := []byte("dingbooms")

	sig, err := private.Sign(msg)
	assert.Nil(t, err)
	sig2, err := private.Sign(msg2)
	assert.Nil(t, err)

	bz, err := sig.Marshal()
	assert.Nil(t, err)
	bz2, err := sig2.Marshal()
	assert.Nil(t, err)

	if bytes.Equal(bz, bz2) {
		t.Fatal("marshaling different signatures produce the same binary representation")
	}

	if !public.Verify(msg, sig) {
		t.Fatal("cannot verify a message signed with this public key")
	}
	if !public.Verify(msg2, sig2) {
		t.Fatal("cannot verify a message signed with this public key")
	}

	if public.Verify(msg, sig2) {
		t.Fatal("verified message signature of the wrong message")
	}
	if public.Verify(msg2, sig) {
		t.Fatal("verified message signature of the wrong message")
	}

	if public.Verify(msg, &Signature{}) {
		t.Fatal("verified an empty signature of a message")
	}
	if public.Verify(msg, nil) {
		t.Fatal("verified a nil signature of a message")
	}
}

func TestEd25519Address(t *testing.T) {
	pub := GenPrivKeyEd25519().PublicKey()
	pub2 := GenPrivKeyEd25519().PublicKey()
	empty := PublicKey{}

	assert.Nil(t, pub.Condition().Validate())
	assert.Nil(t, pub2.Condition().Validate())
	if bytes.Equal(pub.Condition(), pub2.Condition()) {
		t.Fatal("different public keys produce the same condition")
	}
	assert.Nil(t, empty.Condition())
	assert.Nil(t, empty.Address())

	bz, err := pub.Marshal()
	assert.Nil(t, err)
	var read PublicKey
	err = read.Unmarshal(bz)
	assert.Nil(t, err)
	assert.Equal(t, read.Condition(), pub.Condition())
}
