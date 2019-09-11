package orm

import (
	"reflect"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// SerialModel is implemented by any entity that can be stored using SerialModelBucket.
// GetID/SetID are used to store and access the Key.
// The ID is always set to nil before serializing and storing the Value.
type SerialModel interface {
	weave.Persistent
	Validate() error
	GetID() []byte
	SetID([]byte) error
}

// SerialModelSlicePtr represents a pointer to a slice of SerialModels. Think of it as
// *[]SerialModel Because of Go type system, using []SerialModel would not work for us.
// Instead we use a placeholder type and the validation is done during the
// runtime.
type SerialModelSlicePtr interface{}

// SerialModelBucket is implemented by buckets that operates on SerialModels rather than
// Objects.
type SerialModelBucket interface {
	// One query the database for a single SerialModel instance. Lookup is done
	// by the primary index key. Result is loaded into given destination
	// SerialModel.
	// This method returns ErrNotFound if the entity does not exist in the
	// database.
	// If given SerialModel type cannot be used to contain stored entity, ErrType
	// is returned.
	One(db weave.ReadOnlyKVStore, key []byte, dest SerialModel) error

	// PrefixScan will scan for all SerialModels with a primary key (ID)
	// that begins with the given prefix.
	// The function returns a (possibly empty) iterator, which can
	// load each SerialModel as it arrives.
	// If reverse is true, iterates in descending order (highest value first),
	// otherwise, it iterates in ascending order.
	PrefixScan(db weave.ReadOnlyKVStore, prefix []byte, reverse bool) (SerialModelIterator, error)

	// ByIndex returns all objects that secondary index with given name and
	// given key. Main index is always unique but secondary indexes can
	// return more than one value for the same key.
	// All matching entities are appended to given destination slice. If no
	// result was found, no error is returned and destination slice is not
	// modified.
	ByIndex(db weave.ReadOnlyKVStore, indexName string, key []byte, dest SerialModelSlicePtr) error

	// IndexScan does a PrefixScan, but on the named index. This would let us eg. load all counters
	// in order of their count. Or easily find the lowest or highest count.
	IndexScan(db weave.ReadOnlyKVStore, indexName string, prefix []byte, reverse bool) (SerialModelIterator, error)

	// Create saves given SerialModel in the database. Before inserting into
	// database, SerialModel is validated using its Validate method.
	// ID field must be unset so auto incremented ID is generated.
	Create(db weave.KVStore, m SerialModel) error

	// Upsert creates given SerialModel in the database or upserts of given SerialModel
	// with given ID exists. ID field must be set.
	Upsert(db weave.KVStore, m SerialModel) error

	// Delete removes an entity with given primary key from the database.
	// Returns ErrNotFound if an entity with given key does not exist.
	Delete(db weave.KVStore, key []byte) error

	// Has returns nil if an entity with given primary key value exists. It
	// returns ErrNotFound if no entity can be found.
	// Has is a cheap operation that that does not read the data and only
	// checks the existence of it.
	Has(db weave.KVStore, key []byte) error

	// Register registers this buckets content to be accessible via query
	// requests under the given name.
	Register(name string, r weave.QueryRouter)
}

// NewSerialModelBucket returns a SerialModelBucket instance. This implementation relies on
// a bucket instance. Final implementation should operate directly on the
// KVStore instead.
func NewSerialModelBucket(name string, m SerialModel, opts ...SerialModelBucketOption) SerialModelBucket {
	b := NewBucket(name, m)

	tp := reflect.TypeOf(m)
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}

	smb := &serialModelBucket{
		b:          b,
		idSeq:      b.Sequence("id"),
		model:      tp,
		bucketName: name,
	}
	for _, fn := range opts {
		fn(smb)
	}
	return smb
}

// SerialModelBucketOption is implemented by any function that can configure
// SerialModelBucket during creation.
type SerialModelBucketOption func(smb *serialModelBucket)

// indexInfo keeps information on index
type indexInfo struct {
	name string
	// prefix is the kvstore prefix used for all items in the index
	prefix []byte
	unique bool
}

// WithIndexSerial configures the bucket to build an index with given name. All
// entities stored in the bucket are indexed using value returned by the
// indexer function. If an index is unique, there can be only one entity
// referenced per index value.
func WithIndexSerial(name string, indexer Indexer, unique bool) SerialModelBucketOption {
	return func(smb *serialModelBucket) {
		smb.b = smb.b.WithIndex(name, indexer, unique)
		// Until we get better integration with orm, we need to store some info ourselves here...
		info := indexInfo{
			name:   name,
			prefix: indexPrefix(smb.bucketName, name),
			unique: unique,
		}
		smb.indices = append(smb.indices, info)
	}
}

func indexPrefix(bucketName, indexName string) []byte {
	path := "_i." + bucketName + "_" + indexName + ":"
	return []byte(path)
}

// serialModelBucket is concrete implementation of SerialModelBucket
type serialModelBucket struct {
	b     Bucket
	idSeq Sequence

	bucketName string
	indices    []indexInfo

	// model is referencing the structure type. Event if the structure
	// pointer is implementing SerialModel interface, this variable references
	// the structure directly and not the structure's pointer type.
	model reflect.Type
}

func (smb *serialModelBucket) Register(name string, r weave.QueryRouter) {
	smb.b.Register(name, r)
}

func (smb *serialModelBucket) One(db weave.ReadOnlyKVStore, key []byte, dest SerialModel) error {
	obj, err := smb.b.Get(db, key)
	if err != nil {
		return err
	}
	if obj == nil || obj.Value() == nil {
		return errors.Wrapf(errors.ErrNotFound, "%T not in the store", dest)
	}
	res := obj.Value()

	if !reflect.TypeOf(res).AssignableTo(reflect.TypeOf(dest)) {
		return errors.Wrapf(errors.ErrType, "%T cannot be represented as %T", res, dest)
	}

	ptr := reflect.ValueOf(dest)
	ptr.Elem().Set(reflect.ValueOf(res).Elem())
	if err = ptr.Interface().(SerialModel).SetID(key); err != nil {
		return errors.Wrap(errors.ErrType, "cannot set ID")
	}
	return nil
}

func (smb *serialModelBucket) PrefixScan(db weave.ReadOnlyKVStore, prefix []byte, reverse bool) (SerialModelIterator, error) {
	var rawIter weave.Iterator
	var err error

	start, end := prefixRange(smb.b.DBKey(prefix))
	if reverse {
		rawIter, err = db.ReverseIterator(start, end)
		if err != nil {
			return nil, errors.Wrap(err, "reverse prefix scan")
		}
	} else {
		rawIter, err = db.Iterator(start, end)
		if err != nil {
			return nil, errors.Wrap(err, "prefix scan")
		}
	}

	return &idSerialModelIterator{iterator: rawIter, bucketPrefix: smb.b.DBKey(nil)}, nil
}

// getIndexInfo returns the indexInfo from indices by name
func (smb *serialModelBucket) getIndexInfo(name string) *indexInfo {
	for _, info := range smb.indices {
		if info.name == name {
			return &info
		}
	}
	return nil
}

func (smb *serialModelBucket) IndexScan(db weave.ReadOnlyKVStore, indexName string, prefix []byte, reverse bool) (SerialModelIterator, error) {
	// get index
	info := smb.getIndexInfo(indexName)
	if info == nil {
		return nil, errors.Wrapf(errors.ErrDatabase, "no index with name %s", indexName)
	}

	dbPrefix := append(info.prefix, prefix...)
	start, end := prefixRange(dbPrefix)

	var rawIter weave.Iterator
	var err error
	if reverse {
		rawIter, err = db.ReverseIterator(start, end)
		if err != nil {
			return nil, errors.Wrap(err, "reverse prefix scan")
		}
	} else {
		rawIter, err = db.Iterator(start, end)
		if err != nil {
			return nil, errors.Wrap(err, "prefix scan")
		}
	}

	return &indexSerialModelIterator{
		iterator:     rawIter,
		bucketPrefix: smb.b.DBKey(nil),
		unique:       info.unique,
		kv:           db,
	}, nil
}

func (smb *serialModelBucket) ByIndex(db weave.ReadOnlyKVStore, indexName string, key []byte, destination SerialModelSlicePtr) error {
	objs, err := smb.b.GetIndexed(db, indexName, key)
	if err != nil {
		return err
	}
	if len(objs) == 0 {
		return nil
	}

	dest := reflect.ValueOf(destination)
	if dest.Kind() != reflect.Ptr {
		return errors.Wrap(errors.ErrType, "destination must be a pointer to slice of SerialModels")
	}
	if dest.IsNil() {
		return errors.Wrap(errors.ErrImmutable, "got nil pointer")
	}
	dest = dest.Elem()
	if dest.Kind() != reflect.Slice {
		return errors.Wrap(errors.ErrType, "destination must be a pointer to slice of SerialModels")
	}

	// It is allowed to pass destination as both []MySerialModel and []*MySerialModel
	sliceOfPointers := dest.Type().Elem().Kind() == reflect.Ptr

	allowed := dest.Type().Elem()
	if sliceOfPointers {
		allowed = allowed.Elem()
	}
	if smb.model != allowed {
		return errors.Wrapf(errors.ErrType, "this bucket operates on %s serialmodel and cannot return %s", smb.model, allowed)
	}

	for _, obj := range objs {
		if obj == nil || obj.Value() == nil {
			continue
		}

		val := reflect.ValueOf(obj.Value())
		if err := val.Interface().(SerialModel).SetID(obj.Key()); err != nil {
			return errors.Wrap(errors.ErrType, "cannot set ID")
		}

		if !sliceOfPointers {
			val = val.Elem()
		}
		// store the key on the SerialModel
		dest.Set(reflect.Append(dest, val))
	}
	return nil

}

func (smb *serialModelBucket) Create(db weave.KVStore, m SerialModel) error {
	mTp := reflect.TypeOf(m)
	if mTp.Kind() != reflect.Ptr {
		return errors.Wrap(errors.ErrType, "serialmodel destination must be a pointer")
	}
	if smb.model != mTp.Elem() {
		return errors.Wrapf(errors.ErrType, "cannot store %T type in this bucket", m)
	}

	if err := m.Validate(); err != nil {
		return errors.Wrap(err, "invalid serialmodel")
	}

	key := m.GetID()
	if len(key) != 0 {
		return errors.Wrap(errors.ErrModel, "ID must be unset")
	}

	var err error
	key, err = smb.idSeq.NextVal(db)
	if err != nil {
		return errors.Wrap(err, "ID sequence")
	}

	obj := NewSimpleObj(key, m)
	if err := smb.b.Save(db, obj); err != nil {
		return errors.Wrap(err, "cannot create in the database")
	}
	// after serialization, return original/generated key on SerialModel
	if err := m.SetID(key); err != nil {
		return errors.Wrap(err, "cannot set ID")
	}
	return nil
}

func (smb *serialModelBucket) Upsert(db weave.KVStore, m SerialModel) error {
	mTp := reflect.TypeOf(m)
	if mTp.Kind() != reflect.Ptr {
		return errors.Wrap(errors.ErrType, "serialmodel destination must be a pointer")
	}
	if smb.model != mTp.Elem() {
		return errors.Wrapf(errors.ErrType, "cannot store %T type in this bucket", m)
	}

	if err := m.Validate(); err != nil {
		return errors.Wrap(err, "invalid serialmodel")
	}

	key := m.GetID()
	if len(key) == 0 {
		return errors.Wrap(errors.ErrModel, "ID must be set")
	}

	obj := NewSimpleObj(key, m)
	if err := smb.b.Save(db, obj); err != nil {
		return errors.Wrap(err, "cannot update in the database")
	}
	// after serialization, return original/generated key on SerialModel
	if err := m.SetID(key); err != nil {
		return errors.Wrap(err, "cannot set ID")
	}
	return nil
}

func (smb *serialModelBucket) Delete(db weave.KVStore, key []byte) error {
	if err := smb.Has(db, key); err != nil {
		return err
	}
	return smb.b.Delete(db, key)
}

func (smb *serialModelBucket) Has(db weave.KVStore, key []byte) error {
	if key == nil {
		// nil key is a special case that would cause the store API to panic.
		return errors.ErrNotFound
	}

	// As long as we rely on the Bucket implementation to access the
	// database, we must refine the key.
	key = smb.b.DBKey(key)

	ok, err := db.Has(key)
	if err != nil {
		return err
	}
	if !ok {
		return errors.ErrNotFound
	}
	return nil
}

var _ SerialModelBucket = (*serialModelBucket)(nil)
