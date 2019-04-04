package x

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestPersistent(t *testing.T) {
	good := coin.NewCoinp(52, 12345, "FOO")
	rawGood, err := good.Marshal()
	assert.Nil(t, err)
	assert.Equal(t, rawGood, MustMarshal(good))

	bad := coin.NewCoinp(52, -12345, "of")
	garbage := MustMarshal(bad)
	if reflect.DeepEqual(rawGood, garbage) {
		t.Fatal("garbage serialization worked")
	}
	copy(garbage, []byte{17, 34, 56})

	var got coin.Coin
	MustUnmarshal(&got, rawGood)
	assert.Equal(t, good, &got)

	assert.Panics(t, func() {
		MustUnmarshal(&got, garbage)
	})

	assert.Panics(t, func() {
		MustValidate(bad)
	})

	MustValidate(good)

	assert.Panics(t, func() {
		MustMarshalValid(bad)
	})

	rebz := MustMarshalValid(good)
	assert.Equal(t, rawGood, rebz)
}
