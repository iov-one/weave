package migration

import (
	"reflect"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

// Bucket is a storage engine that supports and requires schema versioning. I
// enforce every model to contain schema version information and where needed
// migrates objects on the fly, before returning to the user.
//
// This bucket does not migrate on the fly the data returned by the queries.
// Both Register and Query methods are using orm.Bucket implementation to
// return data as stored in the database. This is important for the proof to
// work. Query returned data must never be altered.
type Bucket struct {
	orm.Bucket
	packageName string
	schema      *SchemaBucket
	migrations  *register
}

var _ orm.Bucket = (*Bucket)(nil)

// NewBucket returns a new instance of a schema aware bucket implementation.
// Package name is used to track schema version. Bucket name is the namespace
// for the stored entity. Model is the type of the entity this bucket is
// maintaining.
func NewBucket(packageName string, bucketName string, model orm.Model) Bucket {
	return Bucket{
		Bucket:      orm.NewBucket(bucketName, model),
		packageName: packageName,
		schema:      NewSchemaBucket(),
		migrations:  reg,
	}
}

// useRegister will update this bucket to use a custom register instance
// instead of the global one. This is a private method meant to be used for
// tests only.
func (svb Bucket) useRegister(r *register) Bucket {
	svb.migrations = r
	return svb
}

func (svb Bucket) Get(db weave.ReadOnlyKVStore, key []byte) (orm.Object, error) {
	obj, err := svb.Bucket.Get(db, key)
	if err != nil || obj == nil {
		return obj, err
	}
	if err := svb.migrate(db, obj); err != nil {
		return obj, errors.Wrap(err, "migrate")
	}
	return obj, nil
}

func (svb Bucket) Save(db weave.KVStore, obj orm.Object) error {
	if err := svb.migrate(db, obj); err != nil {
		return errors.Wrap(err, "migrate")
	}
	return svb.Bucket.Save(db, obj)
}

func (svb Bucket) migrate(db weave.ReadOnlyKVStore, obj orm.Object) error {
	return migrate(svb.migrations, svb.schema, svb.packageName, db, obj.Value())
}

func (svb Bucket) WithIndex(name string, indexer orm.Indexer, unique bool) orm.Bucket {
	svb.Bucket = svb.Bucket.WithIndex(name, indexer, unique)
	return svb
}

func (svb Bucket) WithMultiKeyIndex(name string, indexer orm.MultiKeyIndexer, unique bool) orm.Bucket {
	svb.Bucket = svb.Bucket.WithMultiKeyIndex(name, indexer, unique)
	return svb
}

// ModelBucket implements the orm.ModelBucket interface and provides the same
// functionality with additional model schema migration.
type ModelBucket struct {
	b           orm.ModelBucket
	packageName string
	schema      *SchemaBucket
	migrations  *register
}

var _ orm.ModelBucket = (*ModelBucket)(nil)

func NewModelBucket(packageName string, b orm.ModelBucket) *ModelBucket {
	return &ModelBucket{
		b:           b,
		packageName: packageName,
		schema:      NewSchemaBucket(),
		migrations:  reg,
	}
}

func (m *ModelBucket) Register(name string, r weave.QueryRouter) {
	m.b.Register(name, r)
}

func (m *ModelBucket) One(db weave.ReadOnlyKVStore, key []byte, dest orm.Model) error {
	if err := m.b.One(db, key, dest); err != nil {
		return err
	}
	if err := m.migrate(db, dest); err != nil {
		return errors.Wrap(err, "migrate")
	}
	return nil
}

func (m *ModelBucket) ByIndex(db weave.ReadOnlyKVStore, indexName string, key []byte, dest orm.ModelSlicePtr) ([][]byte, error) {
	keys, err := m.b.ByIndex(db, indexName, key, dest)
	if err != nil {
		return nil, err
	}

	// The correct type of the dest was already validated by the
	// ModelBucket when getting data by index. We can safely skip checks -
	// dest is a slice of models.
	slice := reflect.ValueOf(dest).Elem()
	for i := 0; i < slice.Len(); i++ {
		item := slice.Index(i)

		// Slice can be both of values and pointer to values. This
		// method must support both notations.
		var model orm.Model
		if m, ok := item.Interface().(orm.Model); ok {
			model = m
		} else {
			model = item.Addr().Interface().(orm.Model)
		}

		if err := m.migrate(db, model); err != nil {
			return nil, errors.Wrapf(err, "migrate %d element", i)
		}
	}
	return keys, nil
}

func (m *ModelBucket) Put(db weave.KVStore, key []byte, model orm.Model) ([]byte, error) {
	if err := m.migrate(db, model); err != nil {
		return nil, errors.Wrap(err, "migrate")
	}
	return m.b.Put(db, key, model)
}

func (m *ModelBucket) Delete(db weave.KVStore, key []byte) error {
	return m.b.Delete(db, key)
}

func (m *ModelBucket) Has(db weave.KVStore, key []byte) error {
	return m.b.Has(db, key)
}

// useRegister will update this bucket to use a custom register instance
// instead of the global one. This is a private method meant to be used for
// tests only.
func (m *ModelBucket) useRegister(r *register) {
	m.migrations = r
}

func (m *ModelBucket) migrate(db weave.ReadOnlyKVStore, model orm.Model) error {
	return migrate(m.migrations, m.schema, m.packageName, db, model)
}

// SerialModelBucket implements the orm.ModelBucket interface and provides the same
// functionality with additional model schema migration.
type SerialModelBucket struct {
	b           orm.SerialModelBucket
	packageName string
	schema      *SchemaBucket
	migrations  *register
}

var _ orm.SerialModelBucket = (*SerialModelBucket)(nil)

func NewSerialModelBucket(packageName string, b orm.SerialModelBucket) *SerialModelBucket {
	return &SerialModelBucket{
		b:           b,
		packageName: packageName,
		schema:      NewSchemaBucket(),
		migrations:  reg,
	}
}

func (smb *SerialModelBucket) Register(name string, r weave.QueryRouter) {
	smb.b.Register(name, r)
}

func (smb *SerialModelBucket) One(db weave.ReadOnlyKVStore, key []byte, dest orm.SerialModel) error {
	if err := smb.b.One(db, key, dest); err != nil {
		return err
	}
	if err := smb.migrate(db, dest); err != nil {
		return errors.Wrap(err, "migrate")
	}
	return nil
}

func (smb *SerialModelBucket) ByIndex(db weave.ReadOnlyKVStore, indexName string, key []byte, dest orm.SerialModelSlicePtr) error {
	if err := smb.b.ByIndex(db, indexName, key, dest); err != nil {
		return err
	}

	// The correct type of the dest was already validated by the
	// SerialModelBucket when getting data by index. We can safely skip checks -
	// dest is a slice of models.
	slice := reflect.ValueOf(dest).Elem()
	for i := 0; i < slice.Len(); i++ {
		item := slice.Index(i)

		// Slice can be both of values and pointer to values. This
		// method must support both notations.
		var model orm.SerialModel
		if m, ok := item.Interface().(orm.SerialModel); ok {
			model = m
		} else {
			model = item.Addr().Interface().(orm.SerialModel)
		}

		if err := smb.migrate(db, model); err != nil {
			return errors.Wrapf(err, "migrate %d element", i)
		}
	}
	return nil
}

func (smb *SerialModelBucket) PrefixScan(db weave.ReadOnlyKVStore, prefix []byte, reverse bool) (orm.SerialModelIterator, error) {
	iter, err := smb.b.PrefixScan(db, prefix, reverse)
	if err != nil {
		return nil, err
	}

	var model *orm.SerialModel
	typ := reflect.TypeOf(model).Elem()
	entity := reflect.New(typ).Interface().(orm.SerialModel)
	// Migrate models
	err = iter.LoadNext(entity)
	if err != nil {
		return nil, errors.Wrap(err, "cannot load iterator")
	}
	for err = iter.LoadNext(entity); err != nil; err = iter.LoadNext(entity) {
		if err := smb.migrate(db, entity); err != nil {
			return nil, errors.Wrapf(err, "migrate %d model", entity)
		}
	}
	iter.Release()

	// Return new iterator. TODO cache
	iter, err = smb.b.PrefixScan(db, prefix, reverse)
	if err != nil {
		return nil, err
	}
	return iter, nil
}

func (smb *SerialModelBucket) IndexScan(db weave.ReadOnlyKVStore, indexName string, prefix []byte, reverse bool) (orm.SerialModelIterator, error) {
	// Get iterator
	iter, err := smb.b.IndexScan(db, indexName, prefix, reverse)
	if err != nil {
		return nil, err
	}

	var model orm.SerialModel
	// Migrate models
	for err = nil; err != nil; err = iter.LoadNext(model) {
		if err := smb.migrate(db, model); err != nil {
			return nil, errors.Wrapf(err, "migrate %d model", model)
		}
	}
	iter.Release()

	// Return new iterator
	iter, err = smb.b.IndexScan(db, indexName, prefix, reverse)
	if err != nil {
		return nil, err
	}
	return iter, nil
}

func (smb *SerialModelBucket) Put(db weave.KVStore, model orm.SerialModel) error {
	if err := smb.migrate(db, model); err != nil {
		return errors.Wrap(err, "migrate")
	}
	return smb.b.Put(db, model)
}

func (smb *SerialModelBucket) Delete(db weave.KVStore, key []byte) error {
	return smb.b.Delete(db, key)
}

func (smb *SerialModelBucket) Has(db weave.KVStore, key []byte) error {
	return smb.b.Has(db, key)
}

// useRegister will update this bucket to use a custom register instance
// instead of the global one. This is a private method meant to be used for
// tests only.
func (smb *SerialModelBucket) useRegister(r *register) {
	smb.migrations = r
}

func (smb *SerialModelBucket) migrate(db weave.ReadOnlyKVStore, model orm.SerialModel) error {
	return migrate(smb.migrations, smb.schema, smb.packageName, db, model)
}

func migrate(
	migrations *register,
	schema *SchemaBucket,
	packageName string,
	db weave.ReadOnlyKVStore,
	value interface{},
) error {
	m, ok := value.(Migratable)
	if !ok {
		return errors.Wrap(errors.ErrModel, "model cannot be migrated")
	}
	currSchemaVer, err := schema.CurrentSchema(db, packageName)
	if err != nil {
		return errors.Wrapf(err, "current schema version of package %q", packageName)
	}

	meta := m.GetMetadata()
	if meta == nil {
		return errors.Wrapf(errors.ErrMetadata, "%T metadata is nil", m)
	}

	// In case of schema not being set we assume the code is expecting the
	// current version. We can therefore set the default to current schema
	// version.
	if meta.Schema == 0 {
		meta.Schema = currSchemaVer
		return nil
	}

	if meta.Schema > currSchemaVer {
		return errors.Wrapf(errors.ErrSchema, "model schema higher than %d", currSchemaVer)
	}

	// Migration is applied in place, directly modifying the instance.
	if err := migrations.Apply(db, m, currSchemaVer); err != nil {
		return errors.Wrap(err, "schema migration")
	}
	return nil
}

// Migrate will query the current schema of the named package and attempt
// to Migrate the passed value up to the current value.
//
// Returns an error if the passed value is not Migratable,
// not registered with migrations, missing Metadata, has a Schema
// higher than currentSchema, if the final migrated value is invalid,
// or other such conditions.
//
// If this returns no error, you can safely use the contents of value in
// code working with the currentSchema.
func Migrate(
	db weave.ReadOnlyKVStore,
	packageName string,
	value interface{},
) error {
	return migrate(reg, NewSchemaBucket(), packageName, db, value)
}
