package ideas

import (
	"github.com/confio/weave"
)

// Bucket is a generic holder that stores data as well
// as references to secondary indexes and sequences.
//
// This is a generic building block that should generally
// be embedded in a type-safe wrapper to ensure all data
// is the same type
// Bucket is a prefixed subspace of the DB
// proto defines the default Model, all elements of this type
type Bucket struct {
	prefix []byte
	empty  Cloneable
	create Cloneable
}

// NewBucket creates a bucket to store data
func NewBucket(name string, empty Cloneable, create Cloneable) Bucket {
	// TODO: enforce name as [a-z]{4}?
	prefix := append([]byte(name), ':')
	return Bucket{
		prefix: prefix,
		empty:  empty,
		create: create,
	}
}

func (b Bucket) Create(key []byte) Object {
	obj := b.create.Clone()
	if key != nil {
		sk, ok := obj.(SetKeyer)
		if ok {
			sk.SetKey(key)
		}
	}
	return obj
}

// Get one element
func (b Bucket) Get(db weave.KVStore, key []byte) (Object, error) {
	dbkey := append(b.prefix, key...)
	bz := db.Get(dbkey)
	if bz == nil {
		return nil, nil
	}

	proto := b.empty.Clone()
	err := proto.Value().Unmarshal(bz)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

func (b Bucket) GetOrCreate(db weave.KVStore, key []byte) (Object, error) {
	// return if there is a result from get
	obj, err := b.Get(db, key)
	if obj != nil || err != nil {
		return obj, err
	}

	// create one with a key
	return b.Create(key), nil
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

	dbkey := append(b.prefix, model.GetKey()...)
	db.Set(dbkey, bz)
	return nil
}

// Sequence returns a Sequence by name
func (b Bucket) Sequence(name string) Sequence {
	id := append(b.prefix, []byte(name)...)
	return NewSequence(id)
}
