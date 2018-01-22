package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBTreeCacheGetSet does basic sanity checks on our cache
//
// Other tests should handle deletes, setting same value,
// iterating over ranges, and general fuzzing
func TestBTreeCacheGetSet(t *testing.T) {
	// devnull is a black hole... just to keep our types proper
	devnull := EmptyKVStore{}

	// base is the root of our data, we can layer on top and
	// all queries should work
	base := NewBTreeCacheWrap(devnull, devnull.NewBatch())

	// make sure the btree is empty at start but returns results
	// that are writen to it
	k, v := []byte("french"), []byte("fry")
	assert.Nil(t, base.Get(k))
	base.Set(k, v)
	assert.Equal(t, v, base.Get(k))

	// now layer another btree on top and make sure that we get
	// base data
	cache := base.CacheWrap()
	assert.Equal(t, v, cache.Get(k))

	// writing more data is only visible in the cache
	k2, v2 := []byte("LA"), []byte("Dodgers")
	assert.Nil(t, cache.Get(k2))
	cache.Set(k2, v2)
	assert.Equal(t, v2, cache.Get(k2))
	assert.Nil(t, base.Get(k2))

	// we can write the cache to the base layer...
	cache.Write()
	assert.Equal(t, v, base.Get(k))
	assert.Equal(t, v2, base.Get(k2))
}
