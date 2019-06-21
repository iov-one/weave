package store

import "github.com/iov-one/weave/errors"

////////////////////////////////////////////////
// Slice -> Iterator

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

func (s *SliceIterator) Next() (key, value []byte, err error) {
	if s.idx >= len(s.data) {
		return nil, nil, errors.Wrap(errors.ErrIteratorDone, "slice iterator")
	}
	val := s.data[s.idx]
	s.idx++
	return val.Key, val.Value, nil
}

// Release releases the Iterator.
func (s *SliceIterator) Release() {
	s.data = nil
}

/////////////////////////////////////////////////////
// Empty KVStore

// EmptyKVStore never holds any data, used as a base layer to test caching
type EmptyKVStore struct{}

var _ KVStore = EmptyKVStore{}

// Get always returns nil
func (e EmptyKVStore) Get(key []byte) ([]byte, error) { return nil, nil }

// Has always returns false
func (e EmptyKVStore) Has(key []byte) (bool, error) { return false, nil }

// Set is a noop
func (e EmptyKVStore) Set(key, value []byte) error { return nil }

// Delete is a noop
func (e EmptyKVStore) Delete(key []byte) error { return nil }

// Iterator is always empty
func (e EmptyKVStore) Iterator(start, end []byte) (Iterator, error) {
	return NewSliceIterator(nil), nil
}

// ReverseIterator is always empty
func (e EmptyKVStore) ReverseIterator(start, end []byte) (Iterator, error) {
	return NewSliceIterator(nil), nil
}

// NewBatch returns a batch that can write to this tree later
func (e EmptyKVStore) NewBatch() Batch {
	return NewNonAtomicBatch(e)
}

////////////////////////////////////////////////////
// Non-atomic batch (dummy implementation)

type opKind int32

const (
	setKind opKind = iota + 1
	delKind
)

// Op is either set or delete
type Op struct {
	kind  opKind
	key   []byte
	value []byte // only for set
}

// Apply performs the stored operation on a writable store
func (o Op) Apply(out SetDeleter) error {
	switch o.kind {
	case setKind:
		return out.Set(o.key, o.value)
	case delKind:
		return out.Delete(o.key)
	default:
		return errors.Wrapf(errors.ErrDatabase, "Unknown kind: %d", o.kind)
	}
}

// IsSetOp returns true if it is setting (false implies delete)
func (o Op) IsSetOp() bool {
	return o.kind == setKind
}

// Key returns a copy of the Key
func (o Op) Key() []byte {
	return append([]byte(nil), o.key...)
}

// SetOp is a helper to create a set operation
func SetOp(key, value []byte) Op {
	return Op{
		kind:  setKind,
		key:   key,
		value: value,
	}
}

// DelOp is a helper to create a del operation
func DelOp(key []byte) Op {
	return Op{
		kind: delKind,
		key:  key,
	}
}

// NonAtomicBatch just piles up ops and executes them later
// on the underlying store. Can be used when there is no better
// option (for in-memory stores).
type NonAtomicBatch struct {
	out SetDeleter
	ops []Op
}

var _ Batch = (*NonAtomicBatch)(nil)

// NewNonAtomicBatch creates an empty batch to be later written
// to the KVStore
func NewNonAtomicBatch(out SetDeleter) *NonAtomicBatch {
	return &NonAtomicBatch{
		out: out,
	}
}

// Set adds a set operation to the batch
func (b *NonAtomicBatch) Set(key, value []byte) error {
	set := Op{
		kind:  setKind,
		key:   key,
		value: value,
	}
	b.ops = append(b.ops, set)
	return nil
}

// Delete adds a delete operation to the batch
func (b *NonAtomicBatch) Delete(key []byte) error {
	del := Op{
		kind: delKind,
		key:  key,
	}
	b.ops = append(b.ops, del)
	return nil
}

// Write writes all the ops to the underlying store and resets
func (b *NonAtomicBatch) Write() error {
	for _, Op := range b.ops {
		err := Op.Apply(b.out)
		if err != nil {
			return err
		}
	}
	b.ops = nil
	return nil
}

// ShowOps is instrumentation for testing,
// it returns a copy of the internal Ops list
func (b *NonAtomicBatch) ShowOps() []Op {
	ops := make([]Op, len(b.ops))
	copy(ops, b.ops)
	return ops
}
