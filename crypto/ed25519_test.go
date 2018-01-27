package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEd25519Signing(t *testing.T) {
	private := GenPrivKeyEd25519().Unwrap()
	public := private.PublicKey().Unwrap()

	msg := []byte("foobar")
	msg2 := []byte("dingbooms")

	sig := private.Sign(msg)
	sig2 := private.Sign(msg2)

	bz, err := sig.Marshal()
	assert.NoError(t, err)
	bz2, err := sig2.Marshal()
	assert.NoError(t, err)
	assert.NotEqual(t, bz, bz2)

	assert.True(t, public.Verify(msg, sig))
	assert.False(t, public.Verify(msg, sig2))
	assert.False(t, public.Verify(msg2, sig))
	assert.True(t, public.Verify(msg2, sig2))
}
