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

	assertPanics(t, func() {
		MustUnmarshal(&got, garbage)
	})

	assertPanics(t, func() {
		MustValidate(bad)
	})

	MustValidate(good)

	assertPanics(t, func() {
		MustMarshalValid(bad)
	})

	rebz := MustMarshalValid(good)
	assert.Equal(t, rawGood, rebz)
}

func assertPanics(t testing.TB, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("panic expected")
		}
	}()
	fn()
}
