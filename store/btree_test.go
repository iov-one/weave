package store

import (
	"testing"
)

// memStoreConstructor returns a base later for testing
// the MemStore implementation of KVStore interface
func memStoreConstructor(height int64) (base CacheableKVStore, cleanup func()) {
	return MemStore(height), func() {}
}

var suite = NewTestSuite(memStoreConstructor)

func TestMemStoreGetSet(t *testing.T) {
	suite.GetSet(t)
}

func TestMemStoreCacheConflicts(t *testing.T) {
	suite.CacheConflicts(t)
}

func TestMemStoreFuzzIterator(t *testing.T) {
	suite.FuzzIterator(t)
}

func TestMemStoreIteratorWithConflicts(t *testing.T) {
	suite.IteratorWithConflicts(t)
}
