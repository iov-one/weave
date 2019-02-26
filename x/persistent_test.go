package x

import (
	"testing"

	"github.com/iov-one/weave/coin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistent(t *testing.T) {
	c := &coin.Coin{Whole: 52, Fractional: 12345, Ticker: "FOO"}
	bad := &coin.Coin{Whole: 52, Fractional: -12345, Ticker: "of"}
	should, err := c.Marshal()
	require.NoError(t, err)

	// marshal
	bz := MustMarshal(c)
	assert.Equal(t, should, bz)
	garbage := MustMarshal(bad)
	assert.NotEqual(t, should, garbage)
	copy(garbage, []byte{17, 34, 56})

	// unmarshal
	got := new(coin.Coin)
	MustUnmarshal(got, bz)
	assert.Equal(t, c, got)
	assert.Panics(t, func() { MustUnmarshal(got, garbage) })

	// validate
	assert.Panics(t, func() { MustValidate(bad) })
	assert.NotPanics(t, func() { MustValidate(c) })
	assert.Panics(t, func() { MustMarshalValid(bad) })
	rebz := MustMarshalValid(c)
	assert.Equal(t, should, rebz)
}
