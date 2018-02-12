package crypto

import (
	"testing"

	"github.com/confio/weave"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEd25519Signing(t *testing.T) {
	private := GenPrivKeyEd25519()
	public := private.PublicKey()

	msg := []byte("foobar")
	msg2 := []byte("dingbooms")

	sig, err := private.Sign(msg)
	require.NoError(t, err)
	sig2, err := private.Sign(msg2)
	require.NoError(t, err)

	bz, err := sig.Marshal()
	assert.NoError(t, err)
	bz2, err := sig2.Marshal()
	assert.NoError(t, err)
	assert.NotEqual(t, bz, bz2)

	assert.True(t, public.Verify(msg, sig))
	assert.False(t, public.Verify(msg, sig2))
	assert.False(t, public.Verify(msg2, sig))
	assert.True(t, public.Verify(msg2, sig2))
	assert.False(t, public.Verify(msg, new(Signature)))
	assert.False(t, public.Verify(msg, nil))
}

func TestEd25519Address(t *testing.T) {
	pub := GenPrivKeyEd25519().PublicKey()
	pub2 := GenPrivKeyEd25519().PublicKey()
	empty := PublicKey{}

	assert.Equal(t, weave.AddressLength, len(pub.Address()))
	assert.Equal(t, weave.AddressLength, len(pub2.Address()))
	assert.NotEqual(t, pub.Address(), pub2.Address())
	assert.Nil(t, empty.Address())

	bz, err := pub.Marshal()
	require.Nil(t, err)
	var read PublicKey
	err = read.Unmarshal(bz)
	require.Nil(t, err)
	assert.Equal(t, read.Address(), pub.Address())
}
