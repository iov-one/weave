package orm

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// RegisterQuery will register a root query (literal keys)
// under "/"
func RegisterQuery(qr weave.QueryRouter) {
	// this never writes, just used to query unprefixed keys
	bucket{}.Register("", qr)
}

// consumeIterator will read all remaining data into an
// array and close the iterator
func consumeIterator(itr weave.Iterator) ([]weave.Model, error) {
	defer itr.Release()

	var res []weave.Model
	key, value, err := itr.Next()
	for err == nil {
		res = append(res, weave.Model{Key: key, Value: value})
		key, value, err = itr.Next()
	}
	if !errors.ErrIteratorDone.Is(err) {
		return nil, err
	}
	return res, nil
}

// prefixRange turns a prefix into (start, end) to create
// and iterator
func prefixRange(prefix []byte) ([]byte, []byte) {
	// special case: no prefix is whole range
	if len(prefix) == 0 {
		return nil, nil
	}

	// copy the prefix and update last byte
	end := make([]byte, len(prefix))
	copy(end, prefix)
	l := len(end) - 1
	end[l]++

	// wait, what if that overflowed?....
	for end[l] == 0 && l > 0 {
		l--
		end[l]++
	}

	// okay, funny guy, you gave us FFF, no end to this range...
	if l == 0 && end[0] == 0 {
		end = nil
	}
	return prefix, end
}

// queryPrefix returns a prefix query as Models
func queryPrefix(db weave.ReadOnlyKVStore, prefix []byte) ([]weave.Model, error) {
	iter, err := db.Iterator(prefixRange(prefix))
	if err != nil {
		return nil, err
	}
	return consumeIterator(iter)
}

var queryRangeLimit = 50

// paginatedIterator wraps an iterator and returns only first X results.
// limitedIterator name is already taken.
type paginatedIterator struct {
	it        weave.Iterator
	remaining int
}

func (i *paginatedIterator) Next() (key []byte, value []byte, err error) {
	if i.remaining == 0 {
		return nil, nil, errors.ErrIteratorDone
	}
	i.remaining--
	return i.it.Next()
}

func (i *paginatedIterator) Release() {
	i.it.Release()
}
