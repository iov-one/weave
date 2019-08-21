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
	Bucket
	idGen IDGenerator
}

// WithSeqIDGenerator adds a Sequence for primary ID key generation on top fo the given bucket implementation
func WithSeqIDGenerator(b Bucket, seqName string) IDGenBucket {
	seq := b.Sequence(seqName)
	return WithIDGenerator(b, IDGeneratorFunc(func(db weave.KVStore, _ CloneableData) ([]byte, error) {
		return seq.NextVal(db)
	}))
}

// WithIDGenerator creates a bucket with uses the given id generator on top of the given bucket implementation.
func WithIDGenerator(b Bucket, gen IDGenerator) IDGenBucket {
	return IDGenBucket{
		Bucket: b,
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
	return obj, b.Save(db, obj)
}

// IDGeneratorFunc provides IDGenerator interface support.
type IDGeneratorFunc func(db weave.KVStore, obj CloneableData) ([]byte, error)

// NextVal returns a new unique ID key
func (i IDGeneratorFunc) NextVal(db weave.KVStore, obj CloneableData) ([]byte, error) {
	return i(db, obj)
}
