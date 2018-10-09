/*
Package orm provides an easy to use db wrapper

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
	"errors"
	"fmt"
	"regexp"

	"github.com/iov-one/weave"
)

const (
	// SeqID is a constant to use to get a default ID sequence
	SeqID = "id"
)

var (
	isBucketName = regexp.MustCompile(`^[a-z_]{3,10}$`).MatchString
)

type Indexed interface {
	weave.QueryHandler
	Update(db weave.KVStore, prev Object, save Object) error
	GetAt(db weave.ReadOnlyKVStore, index []byte) ([][]byte, error)
	GetLike(db weave.ReadOnlyKVStore, pattern Object) ([][]byte, error)
}

// Bucket is a generic holder that stores data as well
// as references to secondary indexes and sequences.
//
// This is a generic building block that should generally
// be embedded in a type-safe wrapper to ensure all data
// is the same type.
// Bucket is a prefixed subspace of the DB
// proto defines the default Model, all elements of this type
type Bucket struct {
	name    string
	prefix  []byte
	proto   Cloneable
	indexes map[string]Indexed
}

var _ weave.QueryHandler = Bucket{}

// NewBucket creates a bucket to store data
func NewBucket(name string, proto Cloneable) Bucket {
	if !isBucketName(name) {
		panic(fmt.Sprintf("Illegal bucket: %s", name))
	}

	return Bucket{
		name:   name,
		prefix: append([]byte(name), ':'),
		proto:  proto,
	}
}

// Register registers this Bucket and all indexes.
// You can define a name here for queries, which is
// different than the bucket name used to prefix the data
func (b Bucket) Register(name string, r weave.QueryRouter) {
	if name == "" {
		name = b.name
	}
	root := "/" + name
	r.Register(root, b)
	for name, idx := range b.indexes {
		r.Register(root+"/"+name, idx)
	}
}

// Query handles queries from the QueryRouter
func (b Bucket) Query(db weave.ReadOnlyKVStore, mod string,
	data []byte) ([]weave.Model, error) {

	switch mod {
	case weave.KeyQueryMod:
		key := b.DBKey(data)
		value := db.Get(key)
		// return nothing on miss
		if value == nil {
			return nil, nil
		}
		res := []weave.Model{{Key: key, Value: value}}
		return res, nil
	case weave.PrefixQueryMod:
		prefix := b.DBKey(data)
		return queryPrefix(db, prefix), nil
	default:
		return nil, errors.New("not implemented: " + mod)
	}
}

// DBKey is the full key we store in the db, including prefix
// We copy into a new array rather than use append, as we don't
// want consequetive calls to overwrite the same byte array.
func (b Bucket) DBKey(key []byte) []byte {
	// Long story: annoying bug... storing with keys "ABC" and "LED"
	// would overwrite each other, also for queries.... huh?
	// turns out name was 4 char,
	// append([]byte(name), ':') in NewBucket would allocate with
	// capacity 8, using 5.
	// append(b.prefix, key...) would just append to this slice and
	// return b.prefix. The next call would do the same an overwrite it.
	// 3 hours and some dlv-ing later, new code here...
	l := len(b.prefix)
	out := make([]byte, l+len(key))
	copy(out, b.prefix)
	copy(out[l:], key)
	return out
}

// Get one element
func (b Bucket) Get(db weave.ReadOnlyKVStore, key []byte) (Object, error) {
	dbkey := b.DBKey(key)
	bz := db.Get(dbkey)
	if bz == nil {
		return nil, nil
	}
	return b.Parse(key, bz)
}

// Parse takes a key and value data (weave.Model) and
// reconstructs the data this Bucket would return.
//
// Used internally as part of Get.
// It is exposed mainly as a test helper, but can work for
// any code that wants to parse
func (b Bucket) Parse(key, value []byte) (Object, error) {
	obj := b.proto.Clone()
	err := obj.Value().Unmarshal(value)
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
	err = b.updateIndexes(db, model.Key(), model)
	if err != nil {
		return err
	}

	// now save this one
	db.Set(b.DBKey(model.Key()), bz)
	return nil
}

// Delete will remove the value at a key
func (b Bucket) Delete(db weave.KVStore, key []byte) error {
	err := b.updateIndexes(db, key, nil)
	if err != nil {
		return err
	}

	// now save this one
	dbkey := b.DBKey(key)
	db.Delete(dbkey)
	return nil
}

func (b Bucket) updateIndexes(db weave.KVStore, key []byte, model Object) error {
	// update all indexes
	if len(b.indexes) > 0 {
		prev, err := b.Get(db, key)
		if err != nil {
			return err
		}
		for _, idx := range b.indexes {
			err = idx.Update(db, prev, model)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Sequence returns a Sequence by name
func (b Bucket) Sequence(name string) Sequence {
	return NewSequence(b.name, name)
}

// WithIndex returns a copy of this bucket with given index,
// panics if it an index with that name is already registered.
//
// Designed to be chained.
func (b Bucket) WithIndex(name string, indexer Indexer, unique bool) Bucket {
	return b.WithMultiKeyIndex(name, func(obj Object) ([][]byte, error) {
		key, err := indexer(obj)
		if err != nil {
			return nil, err
		}
		return [][]byte{key}, nil
	}, unique)
}

func (b Bucket) WithMultiKeyIndex(name string, indexer MultiKeyIndexer, unique bool) Bucket {
	// no duplicate indexes! (panic on init)
	if _, ok := b.indexes[name]; ok {
		panic(fmt.Sprintf("Index %s registered twice", name))
	}

	iname := b.name + "_" + name
	add := NewMulitiKeyIndex(iname, indexer, unique, b.DBKey)
	indexes := make(map[string]Indexed, len(b.indexes)+1)
	for n, i := range b.indexes {
		indexes[n] = i
	}
	indexes[name] = add
	b.indexes = indexes
	return b
}

// GetIndexed queries the named index for the given key
func (b Bucket) GetIndexed(db weave.ReadOnlyKVStore, name string, key []byte) ([]Object, error) {
	idx, ok := b.indexes[name]
	if !ok {
		return nil, ErrInvalidIndex(name)
	}
	refs, err := idx.GetAt(db, key)
	if err != nil {
		return nil, err
	}
	return b.readRefs(db, refs)
}

// GetIndexedLike querys the named index with the given pattern
func (b Bucket) GetIndexedLike(db weave.ReadOnlyKVStore, name string, pattern Object) ([]Object, error) {
	idx, ok := b.indexes[name]
	if !ok {
		return nil, ErrInvalidIndex(name)
	}
	refs, err := idx.GetLike(db, pattern)
	if err != nil {
		return nil, err
	}
	return b.readRefs(db, refs)
}

func (b Bucket) readRefs(db weave.ReadOnlyKVStore, refs [][]byte) ([]Object, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	var err error
	objs := make([]Object, len(refs))
	for i, key := range refs {
		objs[i], err = b.Get(db, key)
		if err != nil {
			return nil, err
		}
	}
	return objs, nil
}
