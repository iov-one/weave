package store

import (
	"bytes"

	"github.com/google/btree"
)

///////////////////////////////////////////////////////
// From Items to Iterator

// TODO: add support for Combine (deleting those below)
type btreeIter struct {
	data []btree.Item
	idx  int
}

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
func ascendBtree(bt *btree.BTree, start, end []byte) *btreeIter {
	var iter btreeIter
	if start == nil && end == nil {
		bt.Ascend(iter.insert)
	} else if start == nil { // end != nil
		bt.AscendLessThan(bkey{end}, iter.insert)
	} else if end == nil { // start != nil
		bt.AscendGreaterOrEqual(bkey{start}, iter.insert)
	} else { // both != nil
		bt.AscendRange(bkey{start}, bkey{end}, iter.insert)
	}
	return &iter
}

func descendBtree(bt *btree.BTree, start, end []byte) *btreeIter {
	var iter btreeIter
	if start == nil && end == nil {
		bt.Descend(iter.insert)
	} else if start == nil { // end != nil
		bt.DescendLessOrEqual(bkeyLess{end}, iter.insert)
	} else if end == nil { // start != nil
		bt.DescendGreaterThan(bkeyLess{start}, iter.insert)
	} else { // both != nil
		bt.DescendRange(bkeyLess{end}, bkeyLess{start}, iter.insert)
	}
	return &iter
}

// insert is designed as a callback to add items from the btree.
// Example Usage (to get an iterator over all items on the tree):
//
//  iter := newItemIter(parentIter)
//  tree.Ascend(iter.insert)
//  iter.skipAllDeleted()
func (b *btreeIter) insert(item btree.Item) bool {
	b.data = append(b.data, item)
	return true
}

func (b *btreeIter) wrap(parent Iterator) *itemIter {
	iter := &itemIter{
		wrap:   b,
		parent: parent,
	}
	iter.skipAllDeleted()
	return iter
}

func (b *btreeIter) next() {
	b.idx++
}

func (b *btreeIter) close() {
	b.data = nil
}

// get requires this is valid, gets what we are pointing at
func (b *btreeIter) get() keyer {
	return b.data[b.idx].(keyer)
}

func (b *btreeIter) valid() bool {
	return (b.idx < len(b.data))
}

type itemIter struct {
	wrap *btreeIter
	// if we are iterating in a cache-wrap (and who isn't),
	// we need to combine this iterator with the parent
	parent Iterator
}

//------- public facing interface ------
var _ Iterator = (*itemIter)(nil)

// Valid implements Iterator and returns true iff it can be read
func (i *itemIter) Valid() bool {
	return i.wrap.valid() || i.parentValid()
}

// Next moves the iterator to the next sequential key in the database, as
// defined by order of iteration.
//
// If Valid returns false, this method will panic.
func (i *itemIter) Next() error {
	// advance either us, parent, or both
	switch i.firstKey() {
	case us:
		i.wrap.next()
	case both:
		i.wrap.next()
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
		return i.wrap.get().Key()
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
		return i.wrap.get().(setItem).value
	case parent:
		return i.parent.Value()
	default: // none
		panic("Advanced past the end!")
	}
}

// Close releases the Iterator.
func (i *itemIter) Close() {
	i.parent.Close()
	i.wrap.close()
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
		if _, ok := i.wrap.get().(deletedItem); ok {
			i.wrap.next()
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

// firstKey selects the iterator with the lowest key is any
func (i *itemIter) firstKey() source {
	// if only one or none is valid, it is clear which to use
	if !i.parentValid() {
		if !i.wrap.valid() {
			return none
		}
		return us
	} else if !i.wrap.valid() {
		return parent
	}

	// both are valid... compare keys....
	parKey := i.parent.Key()
	usKey := i.wrap.get().Key()

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

// makes sure the parent is non-nil before checking if it is valid
func (i *itemIter) parentValid() bool {
	return (i.parent != nil) && i.parent.Valid()
}
