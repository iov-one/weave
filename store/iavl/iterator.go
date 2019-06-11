package iavl

import (
	"fmt"

	"github.com/iov-one/weave/store"
)

type lazyIterator struct {
	data    store.Model
	hasMore bool
	read    chan store.Model
	stop    chan struct{}
}

var _ store.Iterator = (*lazyIterator)(nil)

func newLazyIterator() *lazyIterator {
	read := make(chan store.Model)
	// ensure we never block when we call close()
	stop := make(chan struct{}, 2)
	return &lazyIterator{
		read: read,
		stop: stop,
	}
}

func (i *lazyIterator) add(key []byte, value []byte) bool {
	m := store.Model{Key: key, Value: value}
	select {
	case i.read <- m:
		return false
	case <-i.stop:
		close(i.read)
		fmt.Println("closed")
		return true
	}
}

func (i *lazyIterator) Next() error {
	i.data, i.hasMore = <-i.read
	return nil
}

func (i *lazyIterator) Close() {
	fmt.Println("Close()")
	i.stop <- struct{}{}
}

func (i *lazyIterator) Valid() bool {
	return i.hasMore
}

func (i *lazyIterator) Key() []byte {
	if !i.hasMore {
		panic("read after end of iterator")
	}
	return i.data.Key
}

func (i *lazyIterator) Value() []byte {
	if !i.hasMore {
		panic("read after end of iterator")
	}
	return i.data.Value
}
