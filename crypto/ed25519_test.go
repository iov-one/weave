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

func TestPrivKeyEd25519FromSeed(t *testing.T) {
	cases := map[string]struct {
		seed     []byte
		expected []byte
	}{
		"success 1": {
			seed:     []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 59, 106, 39, 188, 206, 182, 164, 45, 98, 163, 168, 208, 42, 111, 13, 115, 101, 50, 21, 119, 29, 226, 67, 166, 58, 192, 72, 161, 139, 89, 218, 41},
		},
		"success 2": {
			seed:     []byte{31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31},
			expected: []byte{31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 31, 67, 4, 107, 254, 64, 146, 179, 233, 73, 148, 234, 218, 21, 220, 194, 13, 138, 170, 7, 182, 88, 253, 57, 84, 235, 142, 14, 251, 139, 220, 165, 222},
		},
		"failure no seed": {
			seed:     nil,
			expected: nil,
		},
		"failure wrong seed size (n<32)": {
			seed:     []byte{0},
			expected: nil,
		},
		"failure wrong seed size (n>32)": {
			seed:     []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expected: nil,
		},
	}
	
	for _, tc := range cases {
		if tc.expected != nil {
			privKey := PrivKeyEd25519FromSeed(tc.seed)
			assert.Equal(t, tc.expected, privKey.GetEd25519())
		} else {
			assert.Panics(t, func() { PrivKeyEd25519FromSeed(tc.seed) })
		}
	}
}
