package orm

import (
	"github.com/confio/weave"
)

// Bucket is a generic holder that stores data as well
// as references to secondary indexes and sequences.
//
// This is a generic building block that should generally
// be embedded in a type-safe wrapper to ensure all data
// is the same type.
// Bucket is a prefixed subspace of the DB
// proto defines the default Model, all elements of this type
type Bucket struct {
	prefix []byte
	proto  Cloneable
}

// NewBucket creates a bucket to store data
func NewBucket(name string, proto Cloneable) Bucket {
	// TODO: enforce name as [a-z]{4}?
	prefix := append([]byte(name), ':')
	return Bucket{
		prefix: prefix,
		proto:  proto,
	}
}

// Get one element
func (b Bucket) Get(db weave.KVStore, key []byte) (Object, error) {
	dbkey := append(b.prefix, key...)
	bz := db.Get(dbkey)
	if bz == nil {
		return nil, nil
	}

	proto := b.proto.Clone()
	err := proto.Value().Unmarshal(bz)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

// Save will write a model, it must be of the same type as proto
func (b Bucket) Save(db weave.KVStore, model Object) error {
	err := model.Validate()
	if err != nil {
		return err
	}

	bz, err := model.Value().Marshal()
	if err != nil {
		return err
	}

	dbkey := append(b.prefix, model.Key()...)
	db.Set(dbkey, bz)
	return nil
}

// Sequence returns a Sequence by name
func (b Bucket) Sequence(name string) Sequence {
	id := append(b.prefix, []byte(name)...)
	return NewSequence(id)
}
