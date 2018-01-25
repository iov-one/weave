package iavl

import (
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/confio/weave/store"
)

// CommitStore manages a iavl committed state
type CommitStore struct {
	tree *iavl.VersionedTree
}

var _ store.CommitKVStore = CommitStore{}

// NewCommitStore creates a new store with disk backing
func NewCommitStore(dir string) CommitStore {
	// TODO: make this db
	var db dbm.DB
	cacheSize := 10000
	tree := iavl.NewVersionedTree(cacheSize, db)
	return CommitStore{tree}
}

// Get returns the value at last committed state
// returns nil iff key doesn't exist. Panics on nil key.
func (s CommitStore) Get(key []byte) []byte {
	version := s.tree.LatestVersion()
	_, val := s.tree.GetVersioned(key, version)
	return val
}

// Commit the next version to disk, and returns info
func (s CommitStore) Commit() store.CommitID {
	version := s.tree.LatestVersion() + 1
	hash, err := s.tree.SaveVersion(version)
	if err != nil {
		panic(err)
	}
	return store.CommitID{
		Version: int64(version),
		Hash:    hash,
	}
}

// LoadLatestVersion loads the latest persisted version.
// If there was a crash during the last commit, it is guaranteed
// to return a stable state, even if older.
func (s CommitStore) LoadLatestVersion() error {
	return s.tree.Load()
}

// LatestVersion returns info on the latest version saved to disk
func (s CommitStore) LatestVersion() store.CommitID {
	return store.CommitID{
		Version: int64(s.tree.LatestVersion()),
		Hash:    s.tree.Hash(),
	}
}

// CacheWrap gives us a savepoint to perform actions
// TODO: add Batch to atomic writes and efficiency
// invisibly inside this CacheWrap???
func (s CommitStore) CacheWrap() store.KVCacheWrap {
	return Cache{
		parent: s,
		tree:   s.tree.Tree(),
	}
}

// func (b *Bonsai) GetVersionedWithProof(key []byte, version int64) ([]byte, iavl.KeyProof, error) {
//   return b.Tree.GetVersionedWithProof(key, uint64(version))
// }

// TODO: create batch and reader and wrap the rest in btree...

// Cache is a working cache on top of this tree
type Cache struct {
	parent CommitStore
	tree   *iavl.Tree
}

var _ store.KVCacheWrap = Cache{}

// Get returns nil iff key doesn't exist. Panics on nil key.
func (c Cache) Get(key []byte) []byte {
	_, val := c.tree.Get(key)
	return val
}

// Has checks if a key exists. Panics on nil key.
func (c Cache) Has(key []byte) bool {
	return c.tree.Has(key)
}

// Set adds a new value
func (c Cache) Set(key, value []byte) {
	c.tree.Set(key, value)
}

// Delete removes from the tree
func (c Cache) Delete(key []byte) {
	c.tree.Remove(key)
}

// NewBatch returns a batch that can write multiple ops atomically
func (c Cache) NewBatch() store.Batch {
	return store.NewNonAtomicBatch(c)
}

// CacheWrap wraps us once again, with btree
func (c Cache) CacheWrap() store.KVCacheWrap {
	return store.NewBTreeCacheWrap(c, c.NewBatch(), nil)
}

// Write syncs with the underlying store.
func (c Cache) Write() {
	c.parent.Commit()
}

// Discard is a no-op... just garbage collect
func (c Cache) Discard() {}

// Iterator over a domain of keys in ascending order. End is exclusive.
// Start must be less than end, or the Iterator is invalid.
// CONTRACT: No writes may happen within a domain while an iterator exists over it.
func (c Cache) Iterator(start, end []byte) store.Iterator {
	var res []store.Model
	add := func(key []byte, value []byte) bool {
		m := store.Model{Key: key, Value: value}
		res = append(res, m)
		return true
	}
	c.tree.IterateRange(start, end, true, add)
	return store.NewSliceIterator(res)
}

// ReverseIterator over a domain of keys in descending order. End is exclusive.
// Start must be greater than end, or the Iterator is invalid.
// CONTRACT: No writes may happen within a domain while an iterator exists over it.
func (c Cache) ReverseIterator(start, end []byte) store.Iterator {
	var res []store.Model
	add := func(key []byte, value []byte) bool {
		m := store.Model{Key: key, Value: value}
		res = append(res, m)
		return true
	}
	c.tree.IterateRange(start, end, false, add)
	return store.NewSliceIterator(res)
}
