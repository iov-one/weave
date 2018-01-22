package store

////////////////////////////////////////////////
// Slice -> Iterator

// Model groups together key and value to help build Iterators
type Model struct {
	Key   []byte
	Value []byte
}

// SliceIterator wraps an Iterator over a slice of models
//
// TODO: make this private and only expose Iterator interface????
type SliceIterator struct {
	data []Model
	idx  int
}

var _ Iterator = (*SliceIterator)(nil)

// NewSliceIterator creates a new Iterator over this slice
func NewSliceIterator(data []Model) *SliceIterator {
	return &SliceIterator{
		data: data,
	}
}

// Valid implements Iterator and returns true iff it can be read
func (s *SliceIterator) Valid() bool {
	return s.idx < len(s.data)
}

// Next moves the iterator to the next sequential key in the database, as
// defined by order of iteration.
//
// If Valid returns false, this method will panic.
func (s *SliceIterator) Next() {
	s.assertValid()
	s.idx++
}

func (s *SliceIterator) assertValid() {
	if s.idx >= len(s.data) {
		panic("Passed end of slice")
	}
}

// Key returns the key of the cursor.
func (s *SliceIterator) Key() (key []byte) {
	s.assertValid()
	return s.data[s.idx].Key
}

// Value returns the value of the cursor.
func (s *SliceIterator) Value() (value []byte) {
	s.assertValid()
	return s.data[s.idx].Value
}

// Close releases the Iterator.
func (s *SliceIterator) Close() {
	s.data = nil
}

/////////////////////////////////////////////////////
// Empty KVStore

// EmptyKVStore never holds any data, used as a base layer to test caching
type EmptyKVStore struct{}

var _ KVStore = EmptyKVStore{}

// Get always returns nil
func (e EmptyKVStore) Get(key []byte) []byte { return nil }

// Has always returns false
func (e EmptyKVStore) Has(key []byte) bool { return false }

// Set is a noop
func (e EmptyKVStore) Set(key, value []byte) {}

// Delete is a noop
func (e EmptyKVStore) Delete(key []byte) {}

// Iterator is always empty
func (e EmptyKVStore) Iterator(start, end []byte) Iterator {
	return NewSliceIterator(nil)
}

// ReverseIterator is always empty
func (e EmptyKVStore) ReverseIterator(start, end []byte) Iterator {
	return NewSliceIterator(nil)
}
