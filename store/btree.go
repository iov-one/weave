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
	b.batch.Set(key, value)
	return nil
}

// Delete deletes from the BTree and to the batch
func (b BTreeCacheWrap) Delete(key []byte) error {
	b.bt.ReplaceOrInsert(newDeletedItem(key))
	b.batch.Delete(key)
	return nil
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
	iter := newItemIter(parentIter)

	if start == nil && end == nil {
		b.bt.Ascend(iter.insert)
	} else if start == nil { // end != nil
		b.bt.AscendLessThan(bkey{end}, iter.insert)
	} else if end == nil { // start != nil
		b.bt.AscendGreaterOrEqual(bkey{start}, iter.insert)
	} else { // both != nil
		b.bt.AscendRange(bkey{start}, bkey{end}, iter.insert)
	}
	iter.skipAllDeleted()

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
	iter := newItemIter(parentIter)

	if start == nil && end == nil {
		b.bt.Descend(iter.insert)
	} else if start == nil { // end != nil
		b.bt.DescendLessOrEqual(bkeyLess{end}, iter.insert)
	} else if end == nil { // start != nil
		b.bt.DescendGreaterThan(bkeyLess{start}, iter.insert)
	} else { // both != nil
		b.bt.DescendRange(bkeyLess{end}, bkeyLess{start}, iter.insert)
	}
	iter.skipAllDeleted()

	return iter, nil
}

// First will get the first value in the cache wrap or backing store
// TODO: optimize
func (b BTreeCacheWrap) First(start, end []byte) ([]byte, []byte, error) {
	iter, err := b.Iterator(start, end)
	if err != nil {
		return nil, nil, err
	}
	return ReadOneFromIterator(iter)
}

// Last will get the last value in the cache wrap or backing store
// TODO: optimize
func (b BTreeCacheWrap) Last(start, end []byte) ([]byte, []byte, error) {
	iter, err := b.ReverseIterator(start, end)
	if err != nil {
		return nil, nil, err
	}
	return ReadOneFromIterator(iter)
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

///////////////////////////////////////////////////////
// From Items to Iterator

// TODO: add support for Combine (deleting those below)
type itemIter struct {
	data []btree.Item
	idx  int
	// if we are iterating in a cache-wrap (and who isn't),
	// we need to combine this iterator with the parent
	parent Iterator
}

var _ Iterator = (*itemIter)(nil)

// source marks where the current item comes from
type source int32

const (
	us source = iota
	parent
	both
	none
)

// combine joins our results with those of the parent,
// taking into consideration overwrites and deletes...
func newItemIter(parent Iterator) *itemIter {
	return &itemIter{
		parent: parent,
	}
}

// insert is designed as a callback to add items from the btree.
// Example Usage (to get an iterator over all items on the tree):
//
//  iter := newItemIter(parentIter)
//  tree.Ascend(iter.insert)
//  iter.skipAllDeleted()
func (i *itemIter) insert(item btree.Item) bool {
	i.data = append(i.data, item)
	return true
}

// makes sure the parent is non-nil before checking if it is valid
func (i *itemIter) parentValid() bool {
	return (i.parent != nil) && i.parent.Valid()
}

// makes sure the parent is non-nil before checking if it is valid
func (i *itemIter) weValid() bool {
	return (i.idx < len(i.data))
}

//------- public facing interface ------

// Valid implements Iterator and returns true iff it can be read
func (i *itemIter) Valid() bool {
	return i.weValid() || i.parentValid()
}

// Next moves the iterator to the next sequential key in the database, as
// defined by order of iteration.
//
// If Valid returns false, this method will panic.
func (i *itemIter) Next() error {
	// advance either us, parent, or both
	switch i.firstKey() {
	case us:
		i.idx++
	case both:
		i.idx++
		fallthrough
	case parent:
		err := i.parent.Next()
		if err != nil {
			return err
		}
	default:
		panic("Advanced past the end!")
	}

	// keep advancing over all deleted entries
	return i.skipAllDeleted()
}

// Key returns the key of the cursor.
func (i *itemIter) Key() (key []byte) {
	switch i.firstKey() {
	case us, both:
		return i.get().Key()
	case parent:
		return i.parent.Key()
	default: //none
		panic("Advanced past the end!")
	}
}

// Value returns the value of the cursor.
func (i *itemIter) Value() (value []byte) {
	switch i.firstKey() {
	case us, both:
		return i.get().(setItem).value
	case parent:
		return i.parent.Value()
	default: // none
		panic("Advanced past the end!")
	}
}

// Close releases the Iterator.
func (i *itemIter) Close() {
	i.data = nil
}

// skipAllDeleted loops and skips any number of deleted items
func (i *itemIter) skipAllDeleted() error {
	var err error
	more := true
	for more {
		more, err = i.skipDeleted()
		if err != nil {
			return err
		}
	}
	return nil
}

// skipDeleted jumps over all elements we can safely fast forward
// return true if skipped, so we can skip again
func (i *itemIter) skipDeleted() (bool, error) {
	src := i.firstKey()
	if src == us || src == both {
		// if our next is deleted, advance...
		if _, ok := i.get().(deletedItem); ok {
			i.idx++
			// if parent had the same key, advance parent as well
			if src == both {
				err := i.parent.Next()
				if err != nil {
					return false, err
				}
			}
			return true, nil
		}
	}
	return false, nil
}

// get requires this is valid, gets what we are pointing at
func (i *itemIter) get() keyer {
	return i.data[i.idx].(keyer)
}

// firstKey selects the iterator with the lowest key is any
func (i *itemIter) firstKey() source {
	// if only one or none is valid, it is clear which to use
	if !i.parentValid() {
		if !i.weValid() {
			return none
		}
		return us
	} else if !i.weValid() {
		return parent
	}

	// both are valid... compare keys....
	parKey := i.parent.Key()
	usKey := i.get().Key()

	// let's see which one to do....
	cmp := bytes.Compare(parKey, usKey)
	if cmp < 0 {
		return parent
	} else if cmp > 0 {
		return us
	} else {
		return both
	}
}
