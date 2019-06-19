package orm

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// IDGenerator defines an interface for custom id generators
type IDGenerator interface {
	// NextVal returns a new unique ID key
	NextVal(db weave.KVStore, obj CloneableData) ([]byte, error)
}

type IDGenBucket struct {
	bucket BaseBucket
	idGen  IDGenerator
}

// WithSeqIDGenerator adds a Sequence for primary ID key generation on top fo the given bucket implementation
func WithSeqIDGenerator(b BaseBucket, seqName string) XIDGenBucket {
	seq := NewSequence(b.Name(), seqName)
	return WithIDGenerator(b, IDGeneratorFunc(func(db weave.KVStore, _ CloneableData) ([]byte, error) {
		return seq.NextVal(db)
	}))
}

// WithIDGenerator creates a bucket with uses the given id generator on top of the given bucket implementation.
func WithIDGenerator(b BaseBucket, gen IDGenerator) XIDGenBucket {
	return IDGenBucket{
		bucket: b,
		idGen:  gen,
	}
}

// Create saves the given data in a persistent bucket with a new generated ID key.
func (b IDGenBucket) Create(db weave.KVStore, data CloneableData) (Object, error) {
	id, err := b.idGen.NextVal(db, data)
	if err != nil {
		return nil, errors.Wrap(err, "id generation")
	}
	obj := NewSimpleObj(id, data)
	return obj, b.bucket.Save(db, obj)
}

func (b IDGenBucket) Update(db weave.KVStore, id []byte, data CloneableData) (Object, error) {
	obj := NewSimpleObj(id, data)
	err := b.bucket.Save(db, obj)
	if err != nil {
		return nil, err
	}
	return obj, err
}

func (b IDGenBucket) Get(db weave.ReadOnlyKVStore, key []byte) (Object, error) {
	return b.bucket.Get(db, key)
}

func (b IDGenBucket) Delete(db weave.KVStore, key []byte) error {
	return b.bucket.Delete(db, key)
}

func (b IDGenBucket) GetIndexed(db weave.ReadOnlyKVStore, name string, key []byte) ([]Object, error) {
	return b.bucket.GetIndexed(db, name, key)
}

func (b IDGenBucket) nextVal(db weave.KVStore, obj CloneableData) ([]byte, error) {
	return b.idGen.NextVal(db, obj)
}

func (b IDGenBucket) visit(f func(rawBucket BaseBucket)) {
	f(b.bucket)
}

func (b IDGenBucket) parent() EmbeddedBucket {
	return b.bucket
}

// IDGeneratorFunc provides IDGenerator interface support.
type IDGeneratorFunc func(db weave.KVStore, obj CloneableData) ([]byte, error)

// NextVal returns a new unique ID key
func (i IDGeneratorFunc) NextVal(db weave.KVStore, obj CloneableData) ([]byte, error) {
	return i(db, obj)
}
