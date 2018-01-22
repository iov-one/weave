package weave

//////////////////////////////////////////////////////////
// Defines all public interfaces for interacting with stores
//
// KVStore/Iterator are the basic objects to use in all code

// KVStore is a simple interface to get/set data
//
// For simplicity, we require all backing stores to implement this
// interface. They *may* implement other methods as well, but
// at least these are required.
type KVStore interface {
	// Get returns nil iff key doesn't exist. Panics on nil key.
	Get(key []byte) []byte

	// Has checks if a key exists. Panics on nil key.
	Has(key []byte) bool

	// Set sets the key. Panics on nil key.
	Set(key, value []byte)

	// Delete deletes the key. Panics on nil key.
	Delete(key []byte)

	// Iterator over a domain of keys in ascending order. End is exclusive.
	// Start must be less than end, or the Iterator is invalid.
	// CONTRACT: No writes may happen within a domain while an iterator exists over it.
	Iterator(start, end []byte) Iterator

	// Iterator over a domain of keys in descending order. End is exclusive.
	// Start must be greater than end, or the Iterator is invalid.
	// CONTRACT: No writes may happen within a domain while an iterator exists over it.
	ReverseIterator(start, end []byte) Iterator
}

/*
Iterator allows us to access a set of items within a range of
keys. These may all be preloaded, or loaded on demand.

  Usage:

  var itr Iterator = ...
  defer itr.Close()

  for ; itr.Valid(); itr.Next() {
    k, v := itr.Key(); itr.Value()
    // ...
  }
*/
type Iterator interface {
	// Valid returns whether the current position is valid.
	// Once invalid, an Iterator is forever invalid.
	Valid() bool

	// Next moves the iterator to the next sequential key in the database, as
	// defined by order of iteration.
	//
	// If Valid returns false, this method will panic.
	Next()

	// Key returns the key of the cursor.
	// If Valid returns false, this method will panic.
	// CONTRACT: key readonly []byte
	Key() (key []byte)

	// Value returns the value of the cursor.
	// If Valid returns false, this method will panic.
	// CONTRACT: value readonly []byte
	Value() (value []byte)

	// Close releases the Iterator.
	Close()
}

///////////////////////////////////////////////////////////
// Caching conditional execution
//
// These extend KVStore to allow grouping temporary writes
// which may be committed/discarded together.
// Like Postgresql SAVEPOINT / ROLLBACK TO SAVEPOINT
//
// These should be used instead of KVStore for methods that
// need this functionality

/*
  CacheableKVStore is a KVStore that supports CacheWrapping

  CacheWrap() should not return a Committer, since Commit() on
  cache-wraps make no sense.
*/
type CacheableKVStore interface {
	KVStore
	CacheWrap() KVCacheWrap
}

// KVCacheWrap allows us to maintain a scratch-pad of uncommitted data
// that we can view with all queries.
//
// At the end, call Write to use the cached data, or Discard to drop it.
type KVCacheWrap interface {
	// CacheableKVStore allows us to use this Cache recursively
	CacheableKVStore

	// Write syncs with the underlying store.
	Write()

	// Discard invalidates this CacheWrap and releases all data
	Discard()
}

///////////////////////////////////////////////////////////////
// Loading / committing Data
//
// These reflect stores that can persist state to disk, load on
// start up, and maintain some history

type CommitKVStore interface {
	// Get returns the value at last committed state
	// returns nil iff key doesn't exist. Panics on nil key.
	Get(key []byte) []byte

	// TODO: Get with proof, also historical queries
	// GetVersionedWithProof(key []byte, version int64) (value []byte)

	// func (b *Bonsai) GetWithProof(key []byte) ([]byte, iavl.KeyProof, error) {
	//   return b.Tree.GetWithProof(key)
	// }

	// func (b *Bonsai) GetVersionedWithProof(key []byte, version int64) ([]byte, iavl.KeyProof, error) {
	//   return b.Tree.GetVersionedWithProof(key, uint64(version))
	// }

	// Get a CacheWrap to perform actions
	// TODO: add Batch to atomic writes and efficiency
	// invisibly inside this CacheWrap???
	CacheWrap() KVCacheWrap

	// Commit the next version to disk, and returns info
	Commit() CommitID

	// LoadLatestVersion loads the latest persisted version.
	// If there was a crash during the last commit, it is guaranteed
	// to return a stable state, even if older.
	LoadLatestVersion() error

	// LatestVersion returns info on the latest version saved to disk
	LatestVersion() CommitID

	// ?????
	// LoadVersion loads a specific persisted version.  When you load an old version, or
	// when the last commit attempt didn't complete, the next commit after
	// loading must be idempotent (return the same commit id).  Otherwise the
	// behavior is undefined.
	// LoadVersion(ver int64) error
}

// CommitID contains the tree version number and its merkle root.
type CommitID struct {
	Version int64
	Hash    []byte
}
