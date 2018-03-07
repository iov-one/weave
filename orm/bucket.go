/*
package orm provides an easy to use db wrapper

Break state space into prefixed sections called Buckets.
* Each bucket contains only one type of object.
* It has a primary index (which may be composite),
and may possess secondary indexes.
* It may possess one or more secondary indexes (1:1 or 1:N)
* Easy queries for one and iteration.

For inspiration, look at [storm](https://github.com/asdine/storm) built on top of [bolt kvstore](https://github.com/boltdb/bolt#using-buckets).
* Do not use so much reflection magic. Better do stuff compile-time static, even if it is a bit of boilerplate.
* Consider general usability flow from that project
*/
package orm

import (
	"fmt"
	"regexp"

	"github.com/confio/weave"
)

const (
	// SeqID is a constant to use to get a default ID sequence
	SeqID = "id"
)

var (
	isBucketName = regexp.MustCompile(`^[A-Za-z]{4}$`).MatchString
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
	if !isBucketName(name) {
		panic(fmt.Sprintf("Illegal bucket: %s", name))
	}

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

	obj := b.proto.Clone()
	err := obj.Value().Unmarshal(bz)
	if err != nil {
		return nil, err
	}
	obj.SetKey(key)
	return obj, nil
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
