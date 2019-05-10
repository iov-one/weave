package orm

import (
	"reflect"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// TODO
// - migrations
// - do not use Bucket but directly access KVStore
// - register for queries

// Model is impelemented by any entity that can be stored using ModelBucket.
//
// This is the same interface as CloneableData. Using the right type names
// provides an easier to read API.
type Model interface {
	weave.Persistent
	Validate() error
	Copy() CloneableData
}

// ModelSlice represents a slice of models. Think of it as []Model
// Because of Go type system, using []Model would not work for us.
type ModelSlice interface{}

// ModelBucket is implemented by buckets that operates on Models rather than
// Objects.
type ModelBucket interface {
	// One query the database for a single model instance. Lookup is done
	// by the primary index key. Result is loaded into given destination
	// model.
	// This method returns ErrNotFound if the entity does not exist in the
	// database.
	// If given model type cannot be used to contain stored entity, ErrType
	// is returned.
	One(db weave.ReadOnlyKVStore, key []byte, dest Model) error

	// ByIndex returns all objects that secondary index with given name and
	// given key. Main index is always unique but secondary indexes can
	// return more than one value for the same key.
	// All matching entities are appended to given destination slice. If no
	// result was found, no error is retured and destination slice is not
	// modified.
	ByIndex(db weave.ReadOnlyKVStore, indexName string, key []byte, dest ModelSlice) error

	// Put saves given model in the database. Before inserting into
	// database, model is validated using its Validate method.
	Put(db weave.KVStore, key []byte, m Model) error

	// Delete removes an entity with given primary key from the database.
	// It returns ErrNotFound if an entity with given key does not exist.
	Delete(db weave.KVStore, key []byte) error
}

// NewModelBucket returns a ModelBucket instance. This implementation relies on
// a bucket instance. Final implementation should operate directly on the
// KVStore instead.
func NewModelBucket(b Bucket) ModelBucket {
	return &modelBucket{
		b: b,
	}
}

type modelBucket struct {
	b Bucket
}

func (mb *modelBucket) One(db weave.ReadOnlyKVStore, key []byte, dest Model) error {
	obj, err := mb.b.Get(db, key)
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

	reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(res).Elem())
	return nil
}

func (mb *modelBucket) ByIndex(db weave.ReadOnlyKVStore, indexName string, key []byte, destination ModelSlice) error {
	objs, err := mb.b.GetIndexed(db, indexName, key)
	if err != nil {
		return err
	}
	if len(objs) == 0 {
		return nil
	}

	dest := reflect.ValueOf(destination)
	if dest.Kind() != reflect.Ptr {
		return errors.Wrap(errors.ErrType, "destination must be a pointer to slice of models")
	}
	if dest.IsNil() {
		return errors.Wrap(errors.ErrImmutable, "got nil pointer")
	}
	dest = dest.Elem()
	if dest.Kind() != reflect.Slice {
		return errors.Wrap(errors.ErrType, "destination must be a pointer to slice of models")
	}

	// It is allowed to pass destination as both []MyModel and []*MyModel
	sliceOfPointers := dest.Type().Elem().Kind() == reflect.Ptr

	for _, obj := range objs {
		if obj == nil || obj.Value() == nil {
			continue
		}
		val := reflect.ValueOf(obj.Value())
		if !sliceOfPointers {
			val = val.Elem()
		}
		dest.Set(reflect.Append(dest, val))
	}
	return nil

}

func (mb *modelBucket) Put(db weave.KVStore, key []byte, m Model) error {
	if err := m.Validate(); err != nil {
		return errors.Wrap(err, "invalid model")
	}
	obj := NewSimpleObj(key, m)
	if err := mb.b.Save(db, obj); err != nil {
		return errors.Wrap(err, "cannot store in the database")
	}
	return nil
}

func (mb *modelBucket) Delete(db weave.KVStore, key []byte) error {
	obj, err := mb.b.Get(db, key)
	if err != nil {
		return err
	}
	if obj == nil || obj.Value() == nil {
		return errors.ErrNotFound
	}
	return mb.b.Delete(db, key)
}

var _ ModelBucket = (*modelBucket)(nil)
