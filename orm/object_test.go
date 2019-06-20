package orm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleObj(t *testing.T) {
	key := []byte("foo")
	val, err := NewMultiRef([]byte("bar"), []byte("baz"))
	require.NoError(t, err)

	obj := NewSimpleObj(key, val)
	require.Equal(t, key, obj.Key())
	require.EqualValues(t, val, obj.Value())
	require.NoError(t, obj.Validate())

	o2 := obj.Clone()
	require.Equal(t, key, o2.Key())
	require.EqualValues(t, val, o2.Value())
	require.NoError(t, o2.Validate())

	// now modify original, should not affect clone
	assert.Nil(t, val.Remove([]byte("bar")))
	assert.Nil(t, val.Remove([]byte("baz")))

	assert.EqualValues(t, val, obj.Value())
	assert.Error(t, obj.Validate())
	assert.NotEqual(t, val, o2.Value())
	assert.NoError(t, o2.Validate())

	// empty-ness is no good
	v2, err := multiRefFromStrings("dings")
	require.NoError(t, err)
	nokey := NewSimpleObj([]byte{}, v2)
	assert.Error(t, nokey.Validate())
	nokey.SetKey([]byte{1, 3})
	assert.NoError(t, nokey.Validate())
}
