package store

import "github.com/google/btree"

// BTreeCacheable adds a simple btree-based CacheWrap
// strategy to a KVStore
type BTreeCacheable struct {
	KVStore
}

var _ CacheableKVStore = BTreeCacheable{}

// CacheWrap returns a BTreeCacheWrap that can be later
// writen to this store, or rolled back
func (b BTreeCacheable) CacheWrap() KVCacheWrap {
	// TODO: reuse FreeList between multiple cache wraps....
	// We create/destroy a lot per tx when processing a block
	return NewBTreeCacheWrap(b.KVStore, b.KVStore.NewBatch())
}

///////////////////////////////////////////////
// Actual CacheWrap implementation

// BTreeCacheWrap places a btree cache over a KVStore
type BTreeCacheWrap struct {
	bt    *btree.BTree
	back  ReadOnlyKVStore
	batch Batch
}

var _ KVCacheWrap = BTreeCacheWrap{}

// NewBTreeCacheWrap initializes a BTree to cache around this
// kv store. Use ReadOnlyKVStore to emphasize that all writes
// must go through the Batch.
func NewBTreeCacheWrap(kv ReadOnlyKVStore, batch Batch) BTreeCacheWrap {
	return BTreeCacheWrap{
		bt:    btree.New(2),
		back:  kv,
		batch: batch,
	}
}

// CacheWrap layers another BTree on top of this one.
// Don't change horses in mid-stream....
//
// Uses NonAtomicBatch as it is only backed by another in-memory batch
func (b BTreeCacheWrap) CacheWrap() KVCacheWrap {
	// TODO: reuse FreeList between multiple cache wraps....
	// We create/destroy a lot per tx when processing a block
	return NewBTreeCacheWrap(b.back, b.NewBatch())
}

// NewBatch returns a non-atomic batch that eventually may write to
// our batch
func (b BTreeCacheWrap) NewBatch() Batch {
	return NewNonAtomicBatch(b.batch)
}

// Write syncs with the underlying store.
func (b BTreeCacheWrap) Write() {
	b.batch.Write()
}

// Discard invalidates this CacheWrap and releases all data
//
// TODO: currently noop....leave it to the garbage collector
func (b BTreeCacheWrap) Discard() {
}

// Set writes to the BTree and to the batch
func (b BTreeCacheWrap) Set(key, value []byte) {
	// TODO: btree
	b.batch.Set(key, value)
}

// Delete deletes from the BTree and to the batch
func (b BTreeCacheWrap) Delete(key []byte) {
	// TODO: btree
	b.batch.Delete(key)
}

// Get reads from btree if there, else backing store
func (b BTreeCacheWrap) Get(key []byte) []byte {
	// TODO: btree
	return b.back.Get(key)
}

// Has reads from btree if there, else backing store
func (b BTreeCacheWrap) Has(key []byte) bool {
	// TODO: btree
	return b.back.Has(key)
}

// Iterator over a domain of keys in ascending order.
// Combines results from btree and backing store
func (b BTreeCacheWrap) Iterator(start, end []byte) Iterator {
	// TODO: btree
	return b.back.Iterator(start, end)
}

// ReverseIterator over a domain of keys in descending order.
// Combines results from btree and backing store
func (b BTreeCacheWrap) ReverseIterator(start, end []byte) Iterator {
	// TODO: btree
	return b.back.ReverseIterator(start, end)
}
