package client

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
)

type Query interface {
	Get(key []byte) []byte
	Iterator(start, end []byte) weave.Iterator
	ReverseIterator(start, end []byte) weave.Iterator
}

// ABCIStore exposes the weave abci.Query interface as a ReadonlyKVStore
type ABCIStore struct {
	query Query
}

func NewABCIStore(query Query) weave.ReadOnlyKVStore {
	return &ABCIStore{query: query}
}

// Get will query for exactly one value over the abci store.
// This can be wrapped with a bucket to reuse key/index/parse logic
func (a *ABCIStore) Get(key []byte) []byte {
	return a.query.Get(key)
}

// Has returns true if the given key in in the abci app store
func (a *ABCIStore) Has(key []byte) bool {
	return len(a.Get(key)) > 0
}

// Iterator attempts to do a range iteration over the store
func (a *ABCIStore) Iterator(start, end []byte) weave.Iterator {
	return a.query.Iterator(start, end)
}

func (a *ABCIStore) ReverseIterator(start, end []byte) weave.Iterator {
	return a.query.ReverseIterator(start, end)
}

func toModels(keys, values []byte) ([]weave.Model, error) {
	var k, v app.ResultSet
	if err := k.Unmarshal(keys); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal keys")
	}
	if err := v.Unmarshal(values); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal values")
	}
	return app.JoinResults(&k, &v)
}

// sliceIterator wraps an Iterator over a slice of models
type sliceIterator struct {
	data []weave.Model
	idx  int
	// TODO: add reverse field
}

// NewSliceIterator creates a new Iterator over this slice
func NewSliceIterator(data []weave.Model) weave.Iterator {
	return &sliceIterator{
		data: data,
	}
}

// Valid implements Iterator and returns true iff it can be read
func (s *sliceIterator) Valid() bool {
	return s.idx < len(s.data)
}

// Next moves the iterator to the next sequential key in the database, as
// defined by order of iteration.
//
// If Valid returns false, this method will panic.
func (s *sliceIterator) Next() {
	s.assertValid()
	s.idx++
}

func (s *sliceIterator) assertValid() {
	if s.idx >= len(s.data) {
		panic("Passed end of slice")
	}
}

// Key returns the key of the cursor.
func (s *sliceIterator) Key() (key []byte) {
	s.assertValid()
	return s.data[s.idx].Key
}

// Value returns the value of the cursor.
func (s *sliceIterator) Value() (value []byte) {
	s.assertValid()
	return s.data[s.idx].Value
}

// Close releases the Iterator.
func (s *sliceIterator) Close() {
	s.data = nil
}
