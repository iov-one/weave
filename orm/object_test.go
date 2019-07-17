package orm

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
)

func TestSimpleObj(t *testing.T) {
	key := []byte("foo")
	val, err := NewMultiRef([]byte("bar"), []byte("baz"))
	assert.Nil(t, err)

	obj := NewSimpleObj(key, val)
	assert.Equal(t, key, obj.Key())
	assert.Equal(t, val, obj.Value())
	assert.Nil(t, obj.Validate())

	o2 := obj.Clone()
	assert.Equal(t, key, o2.Key())
	assert.Equal(t, val, o2.Value())
	assert.Nil(t, o2.Validate())

	// now modify original, should not affect clone
	assert.Nil(t, val.Remove([]byte("bar")))
	assert.Nil(t, val.Remove([]byte("baz")))

	assert.Equal(t, val, obj.Value())
	assert.Equal(t, true, obj.Validate() != nil)
	assert.Equal(t, false, reflect.DeepEqual(val, o2.Value()))
	assert.Nil(t, o2.Validate())

	// empty-ness is no good
	v2, err := multiRefFromStrings("dings")
	assert.Nil(t, err)
	nokey := NewSimpleObj([]byte{}, v2)
	assert.Equal(t, true, nokey.Validate() != nil)
	nokey.SetKey([]byte{1, 3})
	assert.Nil(t, nokey.Validate())
}
