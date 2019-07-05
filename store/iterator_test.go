package store

import (
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
)

func TestCacheIteratorReleaseRaceCondition(t *testing.T) {
	db := MemStore()
	assert.Nil(t, db.Set([]byte("a"), []byte("A")))
	cache := db.CacheWrap()

	it, err := cache.Iterator([]byte("a"), []byte("z"))
	if err != nil {
		t.Fatalf("cannot create iterator: %s", err)
	}
	// Release must be a synchronous operation.
	it.Release()
	assert.Nil(t, db.Delete([]byte("a")))
}

func TestCacheReverseIteratorReleaseRaceCondition(t *testing.T) {
	db := MemStore()
	assert.Nil(t, db.Set([]byte("a"), []byte("A")))
	cache := db.CacheWrap()

	it, err := cache.ReverseIterator([]byte("a"), []byte("z"))
	if err != nil {
		t.Fatalf("cannot create iterator: %s", err)
	}
	// Release must be a synchronous operation.
	it.Release()
	assert.Nil(t, db.Delete([]byte("a")))
}
