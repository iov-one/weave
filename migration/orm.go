package migration

import (
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
func NewBucket(packageName string, bucketName string, model orm.Cloneable) Bucket {
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
	m, ok := obj.Value().(Migratable)
	if !ok {
		return errors.Wrap(errors.ErrModel, "model cannot be migrated")
	}
	currSchemaVer, err := svb.schema.CurrentSchema(db, svb.packageName)
	if err != nil {
		return errors.Wrapf(err, "current schema version of package %q", svb.packageName)
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

	// Migration is applied in place, directly modyfying the instance.
	if err := svb.migrations.Apply(db, m, currSchemaVer); err != nil {
		return errors.Wrap(err, "schema migration")
	}
	return nil
}

func (svb Bucket) WithIndex(name string, indexer orm.Indexer, unique bool) orm.Bucket {
	svb.Bucket = svb.Bucket.WithIndex(name, indexer, unique)
	return svb
}

func (svb Bucket) WithMultiKeyIndex(name string, indexer orm.MultiKeyIndexer, unique bool) orm.Bucket {
	svb.Bucket = svb.Bucket.WithMultiKeyIndex(name, indexer, unique)
	return svb
}
