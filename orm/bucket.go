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
	"fmt"
	"regexp"
	"sort"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const (
	// SeqID is a constant to use to get a default ID sequence
	SeqID = "id"
)

var isBucketName = regexp.MustCompile(`^[a-z_]{3,10}$`).MatchString

type Bucket interface {
	weave.QueryHandler

	DBKey(key []byte) []byte
	Delete(db weave.KVStore, key []byte) error
	Get(db weave.ReadOnlyKVStore, key []byte) (Object, error)
	GetIndexed(db weave.ReadOnlyKVStore, name string, key []byte) ([]Object, error)
	GetIndexedLike(db weave.ReadOnlyKVStore, name string, pattern Object) ([]Object, error)
	Parse(key, value []byte) (Object, error)
	Register(name string, r weave.QueryRouter)
	Save(db weave.KVStore, model Object) error
	Sequence(name string) Sequence
	WithIndex(name string, indexer Indexer, unique bool) Bucket
	WithMultiKeyIndex(name string, indexer MultiKeyIndexer, unique bool) Bucket
}

// bucket is a generic holder that stores data as well
// as references to secondary indexes and sequences.
//
// This is a generic building block that should generally
// be embedded in a type-safe wrapper to ensure all data
// is the same type.
// bucket is a prefixed subspace of the DB
// proto defines the default Model, all elements of this type
type bucket struct {
	name   string
	prefix []byte
	proto  Cloneable
	// index is a list of indexes sorted by
	indexes namedIndexes
}

var _ Bucket = (*bucket)(nil)

type namedIndex struct {
	Index
	publicName string
}

type namedIndexes []namedIndex

// Get returns the index with the given (internal/db) name, or nil if not found
func (n namedIndexes) Get(name string) *Index {
	for _, ni := range n {
		if ni.publicName == name {
			return &ni.Index
		}
	}
	return nil
}

// Has returns true iff an index with the given name is already registered
func (n namedIndexes) Has(name string) bool {
	return n.Get(name) != nil
}

// NewBucket creates a bucket to store data
func NewBucket(name string, proto Cloneable) Bucket {
	if !isBucketName(name) {
		panic(fmt.Sprintf("Illegal bucket: %s", name))
	}

	return bucket{
		name:   name,
		prefix: append([]byte(name), ':'),
		proto:  proto,
	}
}

// Register registers this Bucket and all indexes.
// You can define a name here for queries, which is
// different than the bucket name used to prefix the data
func (b bucket) Register(name string, r weave.QueryRouter) {
	if name == "" {
		name = b.name
	}
	root := "/" + name
	r.Register(root, b)
	for _, ni := range b.indexes {
		r.Register(root+"/"+ni.publicName, ni.Index)
	}
}

// Query handles queries from the QueryRouter.
func (b bucket) Query(db weave.ReadOnlyKVStore, mod string, data []byte) ([]weave.Model, error) {
	switch mod {
	case weave.KeyQueryMod:
		key := b.DBKey(data)
		value, err := db.Get(key)
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, nil
		}
		res := []weave.Model{{Key: key, Value: value}}
		return res, nil
	case weave.PrefixQueryMod:
		prefix := b.DBKey(data)
		return queryPrefix(db, prefix)
	default:
		return nil, errors.Wrapf(errors.ErrInput, "unknown mod: %s", mod)
	}
}

// DBKey is the full key we store in the db, including prefix
// We copy into a new array rather than use append, as we don't
// want consecutive calls to overwrite the same byte array.
func (b bucket) DBKey(key []byte) []byte {
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
func (b bucket) Get(db weave.ReadOnlyKVStore, key []byte) (Object, error) {
	dbkey := b.DBKey(key)
	bz, err := db.Get(dbkey)
	if err != nil {
		return nil, err
	}
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
func (b bucket) Parse(key, value []byte) (Object, error) {
	obj := b.proto.Clone()
	if err := obj.Value().Unmarshal(value); err != nil {
		// If the deserialization fails, this is due to corrupted data
		// or more likely, wrong protobuf declaration being used.
		// We can safely use the string representation of the original
		// error as it carries no relevant information.
		return nil, errors.Wrap(errors.ErrState, err.Error())
	}

	// TODO - ensure the value is migrated to the latest version

	obj.SetKey(key)
	return obj, nil
}

// Save will write a model, it must be of the same type as proto
func (b bucket) Save(db weave.KVStore, model Object) error {
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

	// TODO - ensure the metadata is set

	// now save this one
	return db.Set(b.DBKey(model.Key()), bz)
}

// Delete will remove the value at a key
func (b bucket) Delete(db weave.KVStore, key []byte) error {
	err := b.updateIndexes(db, key, nil)
	if err != nil {
		return err
	}

	// now save this one
	dbkey := b.DBKey(key)
	return db.Delete(dbkey)
}

func (b bucket) updateIndexes(db weave.KVStore, key []byte, model Object) error {
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
func (b bucket) Sequence(name string) Sequence {
	return NewSequence(b.name, name)
}

// WithIndex returns a copy of this bucket with given index,
// panics if it an index with that name is already registered.
//
// Designed to be chained.
func (b bucket) WithIndex(name string, indexer Indexer, unique bool) Bucket {
	return b.WithMultiKeyIndex(name, asMultiKeyIndexer(indexer), unique)
}

func (b bucket) WithMultiKeyIndex(name string, indexer MultiKeyIndexer, unique bool) Bucket {
	// no duplicate indexes! (panic on init)
	if b.indexes.Has(name) {
		panic(fmt.Sprintf("Index %s registered twice", name))
	}

	iname := b.name + "_" + name
	add := NewMultiKeyIndex(iname, indexer, unique, b.DBKey)
	idxs := append(b.indexes, namedIndex{Index: add, publicName: name})
	sort.Slice(idxs, func(i int, j int) bool { return idxs[i].name < idxs[j].name })
	b.indexes = idxs
	return b
}

// GetIndexed queries the named index for the given key
func (b bucket) GetIndexed(db weave.ReadOnlyKVStore, name string, key []byte) ([]Object, error) {
	idx := b.indexes.Get(name)
	if idx == nil {
		return nil, errors.Wrap(ErrInvalidIndex, name)
	}
	refs, err := idx.GetAt(db, key)
	if err != nil {
		return nil, err
	}
	return b.readRefs(db, refs)
}

// GetIndexedLike queries the named index with the given pattern
func (b bucket) GetIndexedLike(db weave.ReadOnlyKVStore, name string, pattern Object) ([]Object, error) {
	idx := b.indexes.Get(name)
	if idx == nil {
		return nil, errors.Wrap(ErrInvalidIndex, name)
	}
	refs, err := idx.GetLike(db, pattern)
	if err != nil {
		return nil, err
	}
	return b.readRefs(db, refs)
}

func (b bucket) readRefs(db weave.ReadOnlyKVStore, refs [][]byte) ([]Object, error) {
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
