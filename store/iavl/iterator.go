package iavl

import (
	"sync"

	"github.com/iov-one/weave/store"
)

type lazyIterator struct {
	data    store.Model
	hasMore bool
	read    chan store.Model
	stop    chan struct{}
	once    sync.Once
}

var _ store.Iterator = (*lazyIterator)(nil)

func newLazyIterator() *lazyIterator {
	read := make(chan store.Model)
	// ensure we never block when we call Close()
	stop := make(chan struct{})
	return &lazyIterator{
		read: read,
		stop: stop,
	}
}

func (i *lazyIterator) add(key []byte, value []byte) bool {
	m := store.Model{Key: key, Value: value}
	select {
	case i.read <- m:
		// false means "don't stop", so add will be called again (if there are more values)
		return false
	case <-i.stop:
		// true means "stop", so add will not be called anymore
		return true
	}
}

func (i *lazyIterator) Next() error {
	i.data, i.hasMore = <-i.read
	return nil
}

func (i *lazyIterator) Close() {
	// make sure we only close once to avoid panics and halts on i.stop
	i.once.Do(func() {
		close(i.stop)
		close(i.read)
	})
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
