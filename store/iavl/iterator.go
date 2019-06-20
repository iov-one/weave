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

func (i *lazyIterator) Next() ([]byte, []byte, error) {
	data, hasMore := <-i.read
	if !hasMore {
		return nil, nil, errors.Wrap(errors.ErrDone, "iavl lazy iterator")
	}
	return data.Key, data.Value, nil
}

func (i *lazyIterator) Return() {
	// make sure we only close once to avoid panics and halts on i.stop
	i.once.Do(func() {
		close(i.stop)
		close(i.read)
	})
}
