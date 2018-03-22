package x

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistent(t *testing.T) {
	coin := &Coin{Whole: 52, Fractional: 12345, Ticker: "FOO"}
	bad := &Coin{Whole: 52, Fractional: -12345, Ticker: "of"}
	should, err := coin.Marshal()
	require.NoError(t, err)

	// marshal
	bz := MustMarshal(coin)
	assert.Equal(t, should, bz)
	garbage := MustMarshal(bad)
	assert.NotEqual(t, should, garbage)
	copy(garbage, []byte{17, 34, 56})

	// unmarshal
	got := new(Coin)
	MustUnmarshal(got, bz)
	assert.Equal(t, coin, got)
	assert.Panics(t, func() { MustUnmarshal(got, garbage) })

	// validate
	assert.Panics(t, func() { MustValidate(bad) })
	assert.NotPanics(t, func() { MustValidate(coin) })
	assert.Panics(t, func() { MustMarshalValid(bad) })
	rebz := MustMarshalValid(coin)
	assert.Equal(t, should, rebz)
}
