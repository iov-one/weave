package store

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/google/btree"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

///////////////////////////////////////////////////////
// From Items to Iterator

type btreeIter struct {
	read     <-chan btree.Item
	stop     chan<- struct{}
	onceStop sync.Once
	onceRead sync.Once
}

// combine joins our results with those of the parent,
// taking into consideration overwrites and deletes...
func ascendBtree(bt *btree.BTree, start, end []byte) *btreeIter {
	read := make(chan btree.Item)
	// ensure we never block when we call close()
	stop := make(chan struct{}, 1)
	iter := &btreeIter{
		read: read,
		stop: stop,
	}

	insert := func(item btree.Item) bool {
		select {
		case read <- item:
			return true
		case <-stop:
			iter.onceRead.Do(func() { close(read) })
			return false
		}
	}

	go func() {
		if start == nil && end == nil {
			bt.Ascend(insert)
		} else if start == nil { // end != nil
			bt.AscendLessThan(bkey{end}, insert)
		} else if end == nil { // start != nil
			bt.AscendGreaterOrEqual(bkey{start}, insert)
		} else { // both != nil
			bt.AscendRange(bkey{start}, bkey{end}, insert)
		}
		iter.onceRead.Do(func() { close(read) })
	}()

	return iter
}

func descendBtree(bt *btree.BTree, start, end []byte) *btreeIter {
	read := make(chan btree.Item)
	// ensure we never block when we call close()
	stop := make(chan struct{}, 1)
	iter := &btreeIter{
		read: read,
		stop: stop,
	}

	insert := func(item btree.Item) bool {
		select {
		case read <- item:
			return true
		case <-stop:
			iter.onceRead.Do(func() { close(read) })
			return false
		}
	}

	go func() {
		if start == nil && end == nil {
			bt.Descend(insert)
		} else if start == nil { // end != nil
			bt.DescendLessOrEqual(bkeyLess{end}, insert)
		} else if end == nil { // start != nil
			bt.DescendGreaterThan(bkeyLess{start}, insert)
		} else { // both != nil
			bt.DescendRange(bkeyLess{end}, bkeyLess{start}, insert)
		}
		iter.onceRead.Do(func() { close(read) })
	}()

	return iter
}

func (b *btreeIter) wrap(parent Iterator) *itemIter {
	iter := &itemIter{
		wrap:   b,
		parent: parent,
	}
	return iter
}

func (b *btreeIter) Next() (keyer, error) {
	data, hasMore := <-b.read
	if !hasMore {
		return nil, errors.Wrap(errors.ErrIteratorDone, "btree iterator")
	}
	key, ok := data.(keyer)
	if !ok {
		return nil, errors.Wrapf(errors.ErrType, "expected keyer, got %T", data)
	}
	return key, nil
}

func (b *btreeIter) Release() {
	b.onceStop.Do(func() {
		close(b.stop)
	})
}

type itemIter struct {
	wrap *btreeIter
	// if we are iterating in a cache-wrap (and who isn't),
	// we need to combine this iterator with the parent
	parent Iterator

	parentDone   bool
	cachedParent Model
	wrapDone     bool
	cachedWrap   keyer
}

//------- public facing interface ------
var _ Iterator = (*itemIter)(nil)

// advanceParent will read next from parent iterators,
// and set cached value as well as done flags.
//
// it will skip closed and missing iterators.
// doesn't return ErrIteratorDone, but only unexpected data errors.
func (i *itemIter) advanceParent() error {
	if i.parent == nil {
		i.parentDone = true
	}
	if i.parentDone || i.cachedParent.Key != nil {
		return nil
	}

	key, value, err := i.parent.Next()
	if errors.ErrIteratorDone.Is(err) {
		i.parentDone = true
	} else if err != nil {
		return errors.Wrap(err, "advance parent")
	} else {
		i.cachedParent = weave.Model{Key: key, Value: value}
	}

	return nil
}

// advance will read next from wrap iterators,
// and set cached value as well as done flags.
//
// It will skip any deleted items before the i.cachedParent.Key value
//
// it will skip closed and missing iterators.
// doesn't return ErrIteratorDone, but only unexpected data errors.
func (i *itemIter) advanceWrap() error {
	for !i.wrapDone && i.cachedWrap == nil {
		var err error
		i.cachedWrap, err = i.wrap.Next()
		if errors.ErrIteratorDone.Is(err) {
			i.wrapDone = true
			return nil
		} else if err != nil {
			return errors.Wrap(err, "advance wrap")
		}
		if _, ok := i.cachedWrap.(deletedItem); ok {
			// if delete, stop if we are at or higher than the parent key
			if i.cachedParent.Key != nil && bytes.Compare(i.cachedWrap.Key(), i.cachedParent.Key) >= 0 {
				return nil
			}
			// otherwise, unset cachedWrap, so we read more
			i.cachedWrap = nil
		}
		// if it is a setItem, we break out with the for loop
	}
	return nil
}

func (i *itemIter) Next() (key, value []byte, err error) {
	// this guarantees that both have xxxDone == true or cachedXxx != nil
	if err := i.advanceParent(); err != nil {
		return nil, nil, errors.Wrap(err, "advanceParent")
	}
	// advances the wrap and skips all deleted up to parent key
	if err := i.advanceWrap(); err != nil {
		return nil, nil, errors.Wrap(err, "advanceWrap")
	}

	if i.wrapDone {
		return i.returnCachedParent()
	}
	if i.parentDone {
		return i.returnCachedWrap()
	}

	// both are valid... try it
	_, isSet := i.cachedWrap.(setItem)
	fmt.Printf("Set: %t (parent: %X, wrap: %X)\n", isSet, i.cachedParent.Key, i.cachedWrap.Key())
	switch bytes.Compare(i.cachedParent.Key, i.cachedWrap.Key()) {
	case 1: // cachedWrap lower
		return i.returnCachedWrap()
	case -1: // cachedParent lower
		return i.returnCachedParent()
	case 0:
		// if it is set, use wrap
		if _, ok := i.cachedWrap.(setItem); ok {
			return i.returnCachedWrap()
		}
		// if it is a delete, then we unset both and continue again
		i.cachedParent = weave.Model{}
		i.cachedWrap = nil
		return i.Next()
	}
	// we should never get here, but compile doesn't know that
	panic("bytes compare should return 1, 0, or -1")
}

// returns cached item from wrap (helper for Next)
func (i *itemIter) returnCachedWrap() (key, value []byte, err error) {
	if i.wrapDone {
		return nil, nil, errors.Wrap(errors.ErrIteratorDone, "itemIter wrap done")
	}
	item := i.cachedWrap.(setItem)
	i.cachedWrap = nil
	return item.key, item.value, nil

}

// returns cached item from parent (helper for Next)
func (i *itemIter) returnCachedParent() (key, value []byte, err error) {
	if i.parentDone {
		return nil, nil, errors.Wrap(errors.ErrIteratorDone, "itemIter parent done")
	}
	key, value = i.cachedParent.Key, i.cachedParent.Value
	i.cachedParent = weave.Model{}
	return key, value, nil
}

// Release releases the Iterator.
func (i *itemIter) Release() {
	i.parent.Release()
	i.wrap.Release()
}
