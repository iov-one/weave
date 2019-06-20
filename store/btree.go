package store

import (
	"bytes"

	"github.com/google/btree"
	"github.com/iov-one/weave/errors"
)

const (
	// DefaultFreeListSize is the size we hold for free node in btree
	DefaultFreeListSize = btree.DefaultFreeListSize
)

// BTreeCacheable adds a simple btree-based CacheWrap
// strategy to a KVStore
type BTreeCacheable struct {
	KVStore
}

var _ CacheableKVStore = BTreeCacheable{}

// CacheWrap returns a BTreeCacheWrap that can be later
// written to this store, or rolled back
func (b BTreeCacheable) CacheWrap() KVCacheWrap {
	// TODO: reuse FreeList between multiple cache wraps....
	// We create/destroy a lot per tx when processing a block
	return NewBTreeCacheWrap(b.KVStore, b.NewBatch(), nil)
}

// MemStore returns a simple implementation useful for tests.
// There is no persistence here....
func MemStore() CacheableKVStore {
	e := EmptyKVStore{}
	return NewBTreeCacheWrap(e, e.NewBatch(), nil)
}

// ShowOpser returns an ordered list of all operations performed
type ShowOpser interface {
	ShowOps() []Op
}

// LogableStore will return a store, along with insight into all operations that were run on it
func LogableStore() (CacheableKVStore, ShowOpser) {
	e := EmptyKVStore{}
	b := NewNonAtomicBatch(e)
	kv := NewBTreeCacheWrap(e, b, nil)
	return kv, b
}

///////////////////////////////////////////////
// Actual CacheWrap implementation

// BTreeCacheWrap places a btree cache over a KVStore
type BTreeCacheWrap struct {
	bt    *btree.BTree
	free  *btree.FreeList
	back  ReadOnlyKVStore
	batch Batch
}

var _ KVCacheWrap = BTreeCacheWrap{}

// NewBTreeCacheWrap initializes a BTree to cache around this
// kv store. Use ReadOnlyKVStore to emphasize that all writes
// must go through the Batch.
//
// free may be nil, but set to an existing list to reuse it
// for memory savings
func NewBTreeCacheWrap(kv ReadOnlyKVStore, batch Batch,
	free *btree.FreeList) BTreeCacheWrap {

	if free == nil {
		free = btree.NewFreeList(DefaultFreeListSize)
	}
	return BTreeCacheWrap{
		bt:    btree.NewWithFreeList(2, free),
		free:  free,
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
	return NewBTreeCacheWrap(b, b.NewBatch(), b.free)
}

// NewBatch returns a non-atomic batch that eventually may write to
// our cachewrap
func (b BTreeCacheWrap) NewBatch() Batch {
	return NewNonAtomicBatch(b)
}

// Write syncs with the underlying store.
// And then cleans up
func (b BTreeCacheWrap) Write() error {
	err := b.batch.Write()
	b.Discard()
	return err
}

// Discard invalidates this CacheWrap and releases all data
//
// TODO: currently noop....leave it to the garbage collector
func (b BTreeCacheWrap) Discard() {
	// clean up the btree -> freelist
	for stop := false; !stop; {
		rem := b.bt.DeleteMin()
		stop = (rem == nil)
	}
}

// Set writes to the BTree and to the batch
func (b BTreeCacheWrap) Set(key, value []byte) error {
	b.bt.ReplaceOrInsert(newSetItem(key, value))

	return b.batch.Set(key, value)
}

// Delete deletes from the BTree and to the batch
func (b BTreeCacheWrap) Delete(key []byte) error {
	b.bt.ReplaceOrInsert(newDeletedItem(key))

	return b.batch.Delete(key)
}

// Get reads from btree if there, else backing store
func (b BTreeCacheWrap) Get(key []byte) ([]byte, error) {
	res := b.bt.Get(bkey{key})
	if res != nil {
		switch t := res.(type) {
		case setItem:
			return t.value, nil
		case deletedItem:
			return nil, nil
		default:
			return nil, errors.Wrapf(errors.ErrDatabase, "Unknown item in btree: %#v", res)
		}
	}
	return b.back.Get(key)
}

// Has reads from btree if there, else backing store
func (b BTreeCacheWrap) Has(key []byte) (bool, error) {
	res := b.bt.Get(bkey{key})
	if res != nil {
		switch res.(type) {
		case setItem:
			return true, nil
		case deletedItem:
			return false, nil
		default:
			return false, errors.Wrapf(errors.ErrDatabase, "Unknown item in btree: %#v", res)
		}
	}
	return b.back.Has(key)
}

// Iterator over a domain of keys in ascending order.
// Combines results from btree and backing store
func (b BTreeCacheWrap) Iterator(start, end []byte) (Iterator, error) {
	// take the backing iterator for start
	parentIter, err := b.back.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	iter := ascendBtree(b.bt, start, end).wrap(parentIter)
	return iter, nil
}

// ReverseIterator over a domain of keys in descending order.
// Combines results from btree and backing store
func (b BTreeCacheWrap) ReverseIterator(start, end []byte) (Iterator, error) {
	// take the backing iterator for start
	parentIter, err := b.back.ReverseIterator(start, end)
	if err != nil {
		return nil, err
	}
	iter := descendBtree(b.bt, start, end).wrap(parentIter)
	return iter, nil
}

/////////////////////////////////////////////////////////
// Items to write to btree

// we enforce all data in our btree implements keyer so we
// can compare nicely
type keyer interface {
	Key() []byte
}

// bkey implements keyer and btree.Item
// and may be used for queries or embedded in data to store
type bkey struct {
	key []byte
}

var _ keyer = bkey{}
var _ btree.Item = bkey{}

func (k bkey) Key() []byte {
	return k.key
}

// Less returns true iff second argument is greater than first
//
// panics if the item to compare doesn't implement keyer.
func (k bkey) Less(item btree.Item) bool {
	cmp := item.(keyer).Key()
	return bytes.Compare(k.key, cmp) < 0
}

type deletedItem struct {
	bkey
}

func newDeletedItem(key []byte) deletedItem {
	return deletedItem{bkey{key}}
}

type setItem struct {
	bkey
	value []byte
}

func newSetItem(key, value []byte) setItem {
	return setItem{bkey{key}, value}
}

// bkeyLess is used to change how ranges are matched....
// use as a key, so exact match is just above this, anything below is below
type bkeyLess struct {
	key []byte
}

var _ keyer = bkeyLess{}
var _ btree.Item = bkeyLess{}

func (k bkeyLess) Key() []byte {
	return k.key
}

// Less returns true iff second argument is greater than first
//
// panics if the item to compare doesn't implement keyer.
func (k bkeyLess) Less(item btree.Item) bool {
	cmp := item.(keyer).Key()
	return bytes.Compare(k.key, cmp) <= 0
}
