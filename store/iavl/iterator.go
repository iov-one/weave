package iavl

import (
	"sync"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
)

type lazyIterator struct {
	read chan store.Model
	stop chan struct{}
	once sync.Once
}

var _ store.Iterator = (*lazyIterator)(nil)

func newLazyIterator() *lazyIterator {
	return &lazyIterator{
		read: make(chan store.Model),
		stop: make(chan struct{}),
	}
}

func (i *lazyIterator) add(key []byte, value []byte) bool {
	select {
	case i.read <- store.Model{Key: key, Value: value}:
		// Returning false means "don't stop", so add will be called
		// again (if there are more values).
		return false
	case <-i.stop:
		// Returning true means "stop", so add will not be called
		// anymore.
		return true
	}
}

func (i *lazyIterator) Next() ([]byte, []byte, error) {
	select {
	case <-i.stop:
		return nil, nil, errors.Wrap(errors.ErrIteratorDone, "closed")
	case data := <-i.read:
		return data.Key, data.Value, nil
	}
}

func (i *lazyIterator) Release() {
	i.once.Do(func() {
		close(i.stop)
	})
}
