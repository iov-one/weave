package app

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

// ABCIStore exposes the weave abci.Query interface as a ReadonlyKVStore
type ABCIStore struct {
	app abci.Application
}

var _ weave.ReadOnlyKVStore = (*ABCIStore)(nil)

func NewABCIStore(app abci.Application) *ABCIStore {
	return &ABCIStore{app: app}
}

// Get will query for exactly one value over the abci store.
// This can be wrapped with a bucket to reuse key/index/parse logic
func (a *ABCIStore) Get(key []byte) []byte {
	query := a.app.Query(abci.RequestQuery{
		Path: "/",
		Data: key,
	})
	// if only the interface supported returning errors....
	if query.Code != 0 {
		panic(query.Log)
	}
	var value ResultSet
	if err := value.Unmarshal(query.Value); err != nil {
		panic(errors.Wrap(err, "unmarshal result set"))
	}
	if len(value.Results) == 0 {
		return nil
	}
	// TODO: assert error if len > 1 ???
	return value.Results[0]
}

// Has returns true if the given key in in the abci app store
func (a *ABCIStore) Has(key []byte) bool {
	return len(a.Get(key)) > 0
}

// Iterator attempts to do a range iteration over the store,
// We only support prefix queries in the abci server for now.
// This client only supports listing everything...
func (a *ABCIStore) Iterator(start, end []byte) weave.Iterator {
	// TODO: support all prefix searches (later even more ranges)
	// look at orm/query.go:prefixRange for an idea how we turn prefix->iterator,
	// we should detect this case and reverse it so we can serialize over abci query
	if start != nil || end != nil {
		panic("iterator only implemented for entire range")
	}

	query := a.app.Query(abci.RequestQuery{
		Path: "/?prefix",
		Data: nil,
	})
	if query.Code != 0 {
		panic(query.Log)
	}
	models, err := toModels(query.Key, query.Value)
	if err != nil {
		panic(errors.Wrap(err, "cannot convert to model"))
	}

	return NewSliceIterator(models)
}

func (a *ABCIStore) ReverseIterator(start, end []byte) weave.Iterator {
	// TODO: load normal iterator but then play it backwards?
	panic("not implemented")
}

func toModels(keys, values []byte) ([]weave.Model, error) {
	var k, v ResultSet
	if err := k.Unmarshal(keys); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal keys")
	}
	if err := v.Unmarshal(values); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal values")
	}
	return JoinResults(&k, &v)
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
