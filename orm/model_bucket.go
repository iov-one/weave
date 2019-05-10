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

	// Put saves given model in the database.
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
